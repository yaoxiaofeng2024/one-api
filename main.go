package main

import (
	"embed"   // Go 1.16+ 内置：将静态文件嵌入到编译后的二进制文件中
	"fmt"     // 格式化输入输出
	"os"      // 操作系统功能：读取环境变量等
	"strconv" // 字符串与数值的类型转换

	"github.com/gin-contrib/sessions"        // Gin 的会话管理中间件
	"github.com/gin-contrib/sessions/cookie" // 基于 Cookie 的会话存储实现
	"github.com/gin-gonic/gin"               // Gin Web 框架：高性能 HTTP 框架
	_ "github.com/joho/godotenv/autoload"    // 自动加载 .env 文件中的环境变量（匿名导入，仅执行 init()）

	"github.com/songquanpeng/one-api/common"               // 公共工具：初始化、Redis 客户端、版本号等
	"github.com/songquanpeng/one-api/common/client"        // HTTP 客户端初始化
	"github.com/songquanpeng/one-api/common/config"        // 全局配置常量与开关
	"github.com/songquanpeng/one-api/common/i18n"          // 国际化（多语言）支持
	"github.com/songquanpeng/one-api/common/logger"        // 日志系统
	"github.com/songquanpeng/one-api/controller"           // 控制器：处理业务逻辑（如自动测试渠道）
	"github.com/songquanpeng/one-api/middleware"           // 中间件：请求ID、语言检测、日志等
	"github.com/songquanpeng/one-api/model"                // 数据模型：数据库操作、缓存、选项配置
	"github.com/songquanpeng/one-api/relay/adaptor/openai" // OpenAI 适配器：Token 编码器初始化
	"github.com/songquanpeng/one-api/router"               // 路由定义：URL 与处理函数的映射
)

// 在编译时将 web/build/ 目录下的所有文件嵌入到 buildFS 变量中
// 这样部署时不需要单独携带前端静态文件，只需一个二进制文件即可运行
//
//go:embed web/build/*
var buildFS embed.FS

// main 是程序的主入口函数
// 整体启动流程：
//  1. 初始化通用组件（日志、配置）
//  2. 初始化数据库（主库 + 日志库）
//  3. 初始化 Redis 缓存
//  4. 初始化系统选项与渠道缓存
//  5. 启动后台协程（选项同步、渠道测试、批量更新）
//  6. 初始化 Token 编码器和 HTTP 客户端
//  7. 初始化国际化
//  8. 配置 Gin 服务器和中间件
//  9. 启动 HTTP 服务器
func main() {
	// ==================== 第一阶段：基础初始化 ====================
	// 初始化通用组件：解析命令行参数、设置版本号、端口号等
	common.Init()

	// 配置日志系统：设置日志输出格式和目标
	logger.SetupLogger()

	// 打印启动日志，包含当前版本号
	logger.SysLogf("One API %s started", common.Version)

	// ==================== 第二阶段：运行模式配置 ====================
	// 如果环境变量 GIN_MODE 不是 debug 模式，则设置为 release 模式
	// release 模式下 Gin 会减少日志输出，提升性能
	if os.Getenv("GIN_MODE") != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// 如果启用了调试模式，打印提示信息
	if config.DebugEnabled {
		logger.SysLog("running in debug mode")
	}

	// ==================== 第三阶段：数据库初始化 ====================
	// 初始化主数据库连接（支持 SQLite / MySQL / PostgreSQL）
	model.InitDB()

	// 初始化日志数据库连接（可与主库分离，减轻主库压力）
	model.InitLogDB()

	var err error
	// 如果数据库中没有 root 账户，自动创建默认的管理员账户
	// 首次部署时，默认用户名和密码通常为 root / 123456
	err = model.CreateRootAccountIfNeed()
	if err != nil {
		logger.FatalLog("database init error: " + err.Error())
	}
	// 使用 defer 确保程序退出时关闭数据库连接，避免资源泄露
	defer func() {
		err := model.CloseDB()
		if err != nil {
			logger.FatalLog("failed to close database: " + err.Error())
		}
	}()

	// ==================== 第四阶段：Redis 缓存初始化 ====================
	// 初始化 Redis 客户端连接（如果配置了 Redis）
	// Redis 用于：会话共享、限流、渠道缓存等
	err = common.InitRedisClient()
	if err != nil {
		logger.FatalLog("failed to initialize Redis: " + err.Error())
	}

	// ==================== 第五阶段：系统选项与内存缓存初始化 ====================
	// 从数据库加载系统选项到内存（如主题、计费规则等配置项）
	model.InitOptionMap()
	logger.SysLog(fmt.Sprintf("using theme %s", config.Theme))

	// 如果启用了 Redis，自动开启内存缓存（兼容旧版本行为）
	if common.RedisEnabled {
		config.MemoryCacheEnabled = true
	}

	// 内存缓存模式：将渠道信息缓存到内存中，减少数据库查询
	// 适用于高并发场景，通过定时同步保持数据一致性
	if config.MemoryCacheEnabled {
		logger.SysLog("memory cache enabled")
		logger.SysLog(fmt.Sprintf("sync frequency: %d seconds", config.SyncFrequency))
		// 初始化渠道缓存：从数据库加载所有渠道到内存
		model.InitChannelCache()
	}

	// 启动后台协程，定时同步系统选项和渠道缓存
	// 这样即使多个实例运行，配置变更也能在 SyncFrequency 秒内生效
	if config.MemoryCacheEnabled {
		go model.SyncOptions(config.SyncFrequency)      // 定时同步系统选项
		go model.SyncChannelCache(config.SyncFrequency) // 定时同步渠道缓存
	}

	// ==================== 第六阶段：可选后台任务 ====================
	// 渠道自动测试：定期自动测试所有渠道的可用性
	// 通过环境变量 CHANNEL_TEST_FREQUENCY 设置测试间隔（秒）
	if os.Getenv("CHANNEL_TEST_FREQUENCY") != "" {
		frequency, err := strconv.Atoi(os.Getenv("CHANNEL_TEST_FREQUENCY"))
		if err != nil {
			logger.FatalLog("failed to parse CHANNEL_TEST_FREQUENCY: " + err.Error())
		}
		go controller.AutomaticallyTestChannels(frequency)
	}
	// 批量更新模式：将多次数据库写操作合并为一次批量写入
	// 适用于高写入场景，减少数据库压力
	if os.Getenv("BATCH_UPDATE_ENABLED") == "true" {
		config.BatchUpdateEnabled = true
		logger.SysLog("batch update enabled with interval " + strconv.Itoa(config.BatchUpdateInterval) + "s")
		model.InitBatchUpdater()
	}
	// 指标监控：当渠道请求失败率过高时自动禁用该渠道
	if config.EnableMetric {
		logger.SysLog("metric enabled, will disable channel if too much request failed")
	}

	// ==================== 第七阶段：Token 编码器与 HTTP 客户端初始化 ====================
	// 初始化 OpenAI Token 编码器（用于计算请求的 Token 数量）
	// 不同模型使用不同的编码器（如 cl100k_base, o200k_base 等）
	openai.InitTokenEncoders()

	// 初始化 HTTP 客户端（设置超时、代理等）
	client.Init()

	// ==================== 第八阶段：国际化初始化 ====================
	// 初始化多语言支持，加载翻译文件
	if err := i18n.Init(); err != nil {
		logger.FatalLog("failed to initialize i18n: " + err.Error())
	}

	// ==================== 第九阶段：HTTP 服务器配置与启动 ====================
	// 创建 Gin 引擎实例（gin.New() 不含默认中间件，需手动添加）
	server := gin.New()
	// Recovery 中间件：捕获 panic 并返回 500 错误，防止整个服务崩溃
	server.Use(gin.Recovery())
	// 注意：gzip 压缩中间件会导致 SSE（Server-Sent Events）流式响应失效
	// 因为 SSE 需要逐步推送数据，而 gzip 会缓冲全部数据后统一压缩
	//server.Use(gzip.Gzip(gzip.DefaultCompression))
	// RequestId 中间件：为每个请求生成唯一 ID，方便日志追踪和问题排查
	server.Use(middleware.RequestId())
	// Language 中间件：检测请求的语言偏好（从 Accept-Language 头或查询参数）
	server.Use(middleware.Language())
	// 设置 Gin 的访问日志中间件
	middleware.SetUpLogger(server)

	// 初始化基于 Cookie 的会话存储
	// SessionSecret 用于加密 Cookie 中的会话数据，防止被篡改
	store := cookie.NewStore([]byte(config.SessionSecret))
	// 启用会话管理中间件，会话名称为 "session"
	server.Use(sessions.Sessions("session", store))

	// 设置路由：将 URL 路径映射到对应的处理函数
	// 同时注册前端静态文件服务（使用嵌入的 buildFS）
	router.SetRouter(server, buildFS)

	// 确定监听端口：优先使用环境变量 PORT，否则使用命令行参数指定的端口
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	logger.SysLogf("server started on http://localhost:%s", port)

	// 启动 HTTP 服务器（阻塞调用，程序将在此处持续运行）
	err = server.Run(":" + port)
	if err != nil {
		logger.FatalLog("failed to start HTTP server: " + err.Error())
	}
}
