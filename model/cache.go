package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/random"
)

var (
	TokenCacheSeconds         = config.SyncFrequency
	UserId2GroupCacheSeconds  = config.SyncFrequency
	UserId2QuotaCacheSeconds  = config.SyncFrequency
	UserId2StatusCacheSeconds = config.SyncFrequency
	GroupModelsCacheSeconds   = config.SyncFrequency
)

// CacheGetTokenByKey 根据 key 字段查询令牌信息，支持 Redis 缓存
//
// 查询的是 tokens 表（由 GORM 根据 Token 结构体自动映射）。
// 表结构关键字段：
//   - key:  令牌的唯一标识（char(48)，唯一索引），即 API Key 中 sk- 后面的部分
//   - user_id, status, remain_quota, models, subnet 等：令牌的属性
//
// 查询策略（经典 cache-aside 模式）：
//
//	Redis 未启用 → 直接查 DB
//	Redis 启用 → 先查 Redis → 命中则返回 → 未命中则查 DB 并回填 Redis
func CacheGetTokenByKey(key string) (*Token, error) {
	// keyCol: SQL 中 `key` 列名的引用方式
	// MySQL/SQLite 用反引号 `key`，PostgreSQL 用双引号 "key"
	// 因为 key 是 SQL 保留字，必须加引号避免语法错误
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}

	var token Token

	// ===== 分支一：Redis 未启用，直接查数据库 =====
	// 每次请求都打 DB，适用于单实例或未配置 Redis 的部署
	if !common.RedisEnabled {
		// GORM 默认表名规则：结构体名复数 → Token → tokens 表
		// 等价 SQL: SELECT * FROM tokens WHERE `key` = ? LIMIT 1
		err := DB.Where(keyCol+" = ?", key).First(&token).Error
		return &token, err
	}

	// ===== 分支二：Redis 已启用，先查缓存 =====
	// Redis key 格式: "token:{key}"，如 "token:abc123"
	tokenObjectString, err := common.RedisGet(fmt.Sprintf("token:%s", key))
	if err != nil {
		// 缓存未命中 → 回源查 DB
		// 等价 SQL: SELECT * FROM tokens WHERE `key` = ? LIMIT 1
		err := DB.Where(keyCol+" = ?", key).First(&token).Error
		if err != nil {
			return nil, err // DB 也没查到，返回错误
		}

		// DB 查到了，回填 Redis 缓存（过期时间 TokenCacheSeconds 秒）
		// 后续相同 key 的请求就能命中缓存，避免重复查 DB
		jsonBytes, err := json.Marshal(token)
		if err != nil {
			return nil, err
		}
		err = common.RedisSet(fmt.Sprintf("token:%s", key), string(jsonBytes), time.Duration(TokenCacheSeconds)*time.Second)
		if err != nil {
			logger.SysError("Redis set token error: " + err.Error())
		}
		return &token, nil
	}

	// 缓存命中 → 反序列化 JSON 字符串为 Token 对象，直接返回
	err = json.Unmarshal([]byte(tokenObjectString), &token)
	return &token, err
}

func CacheGetUserGroup(id int) (group string, err error) {
	if !common.RedisEnabled {
		return GetUserGroup(id)
	}
	group, err = common.RedisGet(fmt.Sprintf("user_group:%d", id))
	if err != nil {
		group, err = GetUserGroup(id)
		if err != nil {
			return "", err
		}
		err = common.RedisSet(fmt.Sprintf("user_group:%d", id), group, time.Duration(UserId2GroupCacheSeconds)*time.Second)
		if err != nil {
			logger.SysError("Redis set user group error: " + err.Error())
		}
	}
	return group, err
}

func fetchAndUpdateUserQuota(ctx context.Context, id int) (quota int64, err error) {
	quota, err = GetUserQuota(id)
	if err != nil {
		return 0, err
	}
	err = common.RedisSet(fmt.Sprintf("user_quota:%d", id), fmt.Sprintf("%d", quota), time.Duration(UserId2QuotaCacheSeconds)*time.Second)
	if err != nil {
		logger.Error(ctx, "Redis set user quota error: "+err.Error())
	}
	return
}

func CacheGetUserQuota(ctx context.Context, id int) (quota int64, err error) {
	if !common.RedisEnabled {
		return GetUserQuota(id)
	}
	quotaString, err := common.RedisGet(fmt.Sprintf("user_quota:%d", id))
	if err != nil {
		return fetchAndUpdateUserQuota(ctx, id)
	}
	quota, err = strconv.ParseInt(quotaString, 10, 64)
	if err != nil {
		return 0, nil
	}
	if quota <= config.PreConsumedQuota { // when user's quota is less than pre-consumed quota, we need to fetch from db
		logger.Infof(ctx, "user %d's cached quota is too low: %d, refreshing from db", quota, id)
		return fetchAndUpdateUserQuota(ctx, id)
	}
	return quota, nil
}

func CacheUpdateUserQuota(ctx context.Context, id int) error {
	if !common.RedisEnabled {
		return nil
	}
	quota, err := CacheGetUserQuota(ctx, id)
	if err != nil {
		return err
	}
	err = common.RedisSet(fmt.Sprintf("user_quota:%d", id), fmt.Sprintf("%d", quota), time.Duration(UserId2QuotaCacheSeconds)*time.Second)
	return err
}

func CacheDecreaseUserQuota(id int, quota int64) error {
	if !common.RedisEnabled {
		return nil
	}
	err := common.RedisDecrease(fmt.Sprintf("user_quota:%d", id), int64(quota))
	return err
}

// CacheIsUserEnabled 缓存用户是否启用
func CacheIsUserEnabled(userId int) (bool, error) {
	if !common.RedisEnabled {
		return IsUserEnabled(userId)
	}
	enabled, err := common.RedisGet(fmt.Sprintf("user_enabled:%d", userId))
	if err == nil {
		return enabled == "1", nil
	}

	userEnabled, err := IsUserEnabled(userId)
	if err != nil {
		return false, err
	}
	enabled = "0"
	if userEnabled {
		enabled = "1"
	}
	err = common.RedisSet(fmt.Sprintf("user_enabled:%d", userId), enabled, time.Duration(UserId2StatusCacheSeconds)*time.Second)
	if err != nil {
		logger.SysError("Redis set user enabled error: " + err.Error())
	}
	return userEnabled, err
}

func CacheGetGroupModels(ctx context.Context, group string) ([]string, error) {
	if !common.RedisEnabled {
		return GetGroupModels(ctx, group)
	}
	modelsStr, err := common.RedisGet(fmt.Sprintf("group_models:%s", group))
	if err == nil {
		return strings.Split(modelsStr, ","), nil
	}
	models, err := GetGroupModels(ctx, group)
	if err != nil {
		return nil, err
	}
	err = common.RedisSet(fmt.Sprintf("group_models:%s", group), strings.Join(models, ","), time.Duration(GroupModelsCacheSeconds)*time.Second)
	if err != nil {
		logger.SysError("Redis set group models error: " + err.Error())
	}
	return models, nil
}

var group2model2channels map[string]map[string][]*Channel
var channelSyncLock sync.RWMutex

// InitChannelCache 从数据库全量加载渠道数据，构建内存缓存
// 最终生成全局变量 group2model2channels，结构为：分组 -> 模型 -> 渠道列表（按优先级降序）
// 用于请求路由时快速查找：某个分组下某个模型可用的渠道有哪些
func InitChannelCache() {
	// ===== 第一步：加载所有启用的渠道 =====
	newChannelId2channel := make(map[int]*Channel)
	var channels []*Channel
	DB.Where("status = ?", ChannelStatusEnabled).Find(&channels)
	for _, channel := range channels {
		newChannelId2channel[channel.Id] = channel
	}

	// ===== 第二步：收集所有分组（group） =====
	// 从 abilities 表中获取所有已注册的分组名称
	var abilities []*Ability
	DB.Find(&abilities)
	groups := make(map[string]bool) // 用 map 去重
	for _, ability := range abilities {
		groups[ability.Group] = true
	}

	// ===== 第三步：构建 分组 -> 模型 -> 渠道列表 的三级映射 =====
	newGroup2model2channels := make(map[string]map[string][]*Channel)
	// 先初始化每个分组的二级 map
	for group := range groups {
		newGroup2model2channels[group] = make(map[string][]*Channel)
	}

	// 遍历所有渠道，将其按所属分组和支持的模型进行归类
	for _, channel := range channels {
		// 一个渠道可能属于多个分组（逗号分隔，如 "default,vip"）
		groups := strings.Split(channel.Group, ",")
		for _, group := range groups {
			// 一个渠道可能支持多个模型（逗号分隔，如 "gpt-4,gpt-3.5-turbo"）
			models := strings.Split(channel.Models, ",")
			for _, model := range models {
				// 懒初始化：首次遇到该 group+model 组合时创建切片
				if _, ok := newGroup2model2channels[group][model]; !ok {
					newGroup2model2channels[group][model] = make([]*Channel, 0)
				}
				newGroup2model2channels[group][model] = append(newGroup2model2channels[group][model], channel)
			}
		}
	}

	// ===== 第四步：按优先级降序排序 =====
	// 优先级高的渠道排在前面，路由时会优先选择
	for group, model2channels := range newGroup2model2channels {
		for model, channels := range model2channels {
			sort.Slice(channels, func(i, j int) bool {
				return channels[i].GetPriority() > channels[j].GetPriority()
			})
			newGroup2model2channels[group][model] = channels
		}
	}

	// ===== 第五步：原子替换全局缓存 =====
	// 使用写锁保护，确保并发读取不会读到半完成的数据
	channelSyncLock.Lock()
	group2model2channels = newGroup2model2channels
	channelSyncLock.Unlock()
	logger.SysLog("channels synced from database")
}

func SyncChannelCache(frequency int) {
	for {
		time.Sleep(time.Duration(frequency) * time.Second)
		logger.SysLog("syncing channels from database")
		InitChannelCache()
	}
}

func CacheGetRandomSatisfiedChannel(group string, model string, ignoreFirstPriority bool) (*Channel, error) {
	if !config.MemoryCacheEnabled {
		return GetRandomSatisfiedChannel(group, model, ignoreFirstPriority)
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	channels := group2model2channels[group][model]
	if len(channels) == 0 {
		return nil, errors.New("channel not found")
	}
	endIdx := len(channels)
	// choose by priority
	firstChannel := channels[0]
	if firstChannel.GetPriority() > 0 {
		for i := range channels {
			if channels[i].GetPriority() != firstChannel.GetPriority() {
				endIdx = i
				break
			}
		}
	}
	idx := rand.Intn(endIdx)
	if ignoreFirstPriority {
		if endIdx < len(channels) { // which means there are more than one priority
			idx = random.RandRange(endIdx, len(channels))
		}
	}
	return channels[idx], nil
}
