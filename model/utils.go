// Package model 中的批量更新器（Batch Updater）
//
// 设计目的：将高频的数据库增量写操作（配额、计数等）先攒在内存中，
// 定时合并为批量写入，减少数据库压力。本质是"写合并"（write coalescing）模式。
//
// ⚠️ 已知风险：
//   - 进程崩溃或 k8s 滚动更新时，内存中未刷盘的数据会丢失
//   - 数据库写入失败时无重试/回退机制，数据也会丢失
//   - 当前没有 WAL 或优雅关停来缓解上述问题
package model

import (
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

// 批量更新的类型枚举
// 使用 iota 自动递增，每种类型对应一个独立的 map 和一把锁
// 新增类型时必须加在 BatchUpdateTypeCount 之前，否则 map/锁数组不会为它分配空间
const (
	BatchUpdateTypeUserQuota        = iota // 0: 用户配额增量
	BatchUpdateTypeTokenQuota              // 1: Token 配额增量
	BatchUpdateTypeUsedQuota               // 2: 用户已用配额增量
	BatchUpdateTypeChannelUsedQuota        // 3: 渠道已用配额增量
	BatchUpdateTypeRequestCount            // 4: 用户请求次数增量
	BatchUpdateTypeCount                   // 5: 类型总数（哨兵值，用于初始化数组长度）
)

// batchUpdateStores: 每种更新类型对应一个 map[int]int64
//
//	map 的 key 是记录 ID（用户ID / TokenID / 渠道ID）
//	map 的 value 是该 ID 的累计增量值
//
// 例如：用户 3 的配额增加了 100，又增加了 200，map 中存的是 300，最终一次写入数据库
var batchUpdateStores []map[int]int64

// batchUpdateLocks: 每种更新类型对应一把互斥锁
//
// 保证 addNewRecord（写入内存）和 batchUpdate（换出 map 写数据库）之间的并发安全
// 不同类型之间互不影响，可以并行操作
var batchUpdateLocks []sync.Mutex

// init 在包加载时自动执行，初始化每种类型的 map 和锁
func init() {
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateStores = append(batchUpdateStores, make(map[int]int64))
		batchUpdateLocks = append(batchUpdateLocks, sync.Mutex{})
	}
}

// InitBatchUpdater 启动后台定时刷盘协程
//
// 由 main.go 在 BATCH_UPDATE_ENABLED=true 时调用
// 协程每隔 BatchUpdateInterval 秒（默认 5 秒）执行一次 batchUpdate()
// 该协程会在进程生命周期内一直运行，无法从外部停止
func InitBatchUpdater() {
	go func() {
		for {
			time.Sleep(time.Duration(config.BatchUpdateInterval) * time.Second)
			batchUpdate()
		}
	}()
}

// addNewRecord 向内存中追加一条增量记录
//
// 调用方（如 relay 层的计费逻辑）在每次请求完成后调用此函数，
// 而不是直接写数据库。同 ID 的增量会累加合并。
//
// 参数：
//   - type_: 更新类型（BatchUpdateTypeXxx 常量之一）
//   - id:   记录 ID（用户ID / TokenID / 渠道ID）
//   - value: 增量值（正数表示增加，负数表示减少）
//
// 并发安全：通过对应类型的 Mutex 保护，多个请求可以同时写不同类型的 map
func addNewRecord(type_ int, id int, value int64) {
	batchUpdateLocks[type_].Lock()
	defer batchUpdateLocks[type_].Unlock()
	if _, ok := batchUpdateStores[type_][id]; !ok {
		batchUpdateStores[type_][id] = value // 首次写入，直接赋值
	} else {
		batchUpdateStores[type_][id] += value // 已有记录，累加增量（这就是"写合并"的核心）
	}
}

// batchUpdate 执行一次批量刷盘：将内存中积累的所有增量写入数据库
//
// 关键设计：先"换出"旧 map，再遍历写入数据库
//
//	加锁 → 取出旧 map → 换上新空 map → 解锁 → 遍历旧 map 写数据库
//
// 这样做的好处是：写数据库的过程不持锁，新的请求可以继续往新 map 里追加，
// 不会因为数据库慢而阻塞业务请求。
//
// ⚠️ 风险：
//   - 换出旧 map 后、写数据库之前，如果进程崩溃，这部分数据会丢失
//   - 数据库写入失败时只记日志，没有重试或回退，数据同样会丢失
func batchUpdate() {
	logger.SysLog("batch update started")
	for i := 0; i < BatchUpdateTypeCount; i++ {
		// 第一步：加锁，取出当前 map，替换为新的空 map，然后立即解锁
		// 这样新的请求写入新 map，不受后续数据库操作的影响
		batchUpdateLocks[i].Lock()
		store := batchUpdateStores[i]
		batchUpdateStores[i] = make(map[int]int64)
		batchUpdateLocks[i].Unlock()

		// 第二步：遍历换出的旧 map，逐条写入数据库
		// TODO: 也许可以将相同 key 的更新合并（当前已经通过 addNewRecord 合并了同 ID 的增量，
		//       但不同类型的更新如果涉及同一张表的同一条记录，仍然是多次 UPDATE）
		for key, value := range store {
			switch i {
			case BatchUpdateTypeUserQuota:
				err := increaseUserQuota(key, value)
				if err != nil {
					logger.SysError("failed to batch update user quota: " + err.Error())
				}
			case BatchUpdateTypeTokenQuota:
				err := increaseTokenQuota(key, value)
				if err != nil {
					logger.SysError("failed to batch update token quota: " + err.Error())
				}
			case BatchUpdateTypeUsedQuota:
				updateUserUsedQuota(key, value)
			case BatchUpdateTypeRequestCount:
				updateUserRequestCount(key, int(value))
			case BatchUpdateTypeChannelUsedQuota:
				updateChannelUsedQuota(key, value)
			}
		}
	}
	logger.SysLog("batch update finished")
}
