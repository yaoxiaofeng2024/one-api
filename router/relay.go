package router

import (
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"

	"github.com/gin-gonic/gin"
)

// SetRelayRouter 配置 AI 模型转发路由（核心网关）
//
// 这是 one-api 最核心的路由模块，所有 /v1 路径的请求都在此注册。
// 请求经过 TokenAuth 鉴权 → Distribute 渠道分发 → Relay 转发到上游 AI 供应商。
//
// 接口命名和路径完全兼容 OpenAI API 规范：
// https://platform.openai.com/docs/api-reference/introduction
//
// 路由分为两类处理：
//   - controller.Relay:             已实现的转发接口，会代理到上游供应商
//   - controller.RelayNotImplemented: 占位接口，返回"未实现"，保留路径以便后续扩展
func SetRelayRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.GzipDecodeMiddleware())

	// ==================== 模型列表查询 ====================
	// 独立路由组，只做 Token 鉴权，不走 Distribute（不需要分配渠道）
	modelsRouter := router.Group("/v1/models")
	modelsRouter.Use(middleware.TokenAuth())
	{
		modelsRouter.GET("", controller.ListModels)           // 列出当前可用模型
		modelsRouter.GET("/:model", controller.RetrieveModel) // 查询单个模型详情
	}

	// ==================== AI 模型转发（核心） ====================
	// 中间件链：Panic 恢复 → Token 鉴权 → Distribute（渠道分发，选择最优渠道）
	relayV1Router := router.Group("/v1")
	relayV1Router.Use(middleware.RelayPanicRecover(), middleware.TokenAuth(), middleware.Distribute())
	{
		// ---------- 自定义代理路由 ----------
		relayV1Router.Any("/oneapi/proxy/:channelid/*target", controller.Relay)
		// 通用代理：指定渠道ID和目标路径，直接转发到该渠道，跳过自动渠道选择

		// ---------- Completions 文本补全 ----------
		relayV1Router.POST("/completions", controller.Relay)
		// 文本补全（旧版）：给定 prompt，模型续写文本
		// https://platform.openai.com/docs/api-reference/completions

		relayV1Router.POST("/chat/completions", controller.Relay)
		// 对话补全（最常用）：给定消息列表，模型生成对话回复
		// https://platform.openai.com/docs/api-reference/chat

		relayV1Router.POST("/edits", controller.Relay)
		// 文本编辑：给定指令和文本，模型修改文本（已弃用，建议用 chat 替代）
		// https://platform.openai.com/docs/api-reference/edits

		// ---------- Images 图像生成 ----------
		relayV1Router.POST("/images/generations", controller.Relay)
		// 图像生成：根据文本描述生成图片（DALL·E）
		// https://platform.openai.com/docs/api-reference/images

		relayV1Router.POST("/images/edits", controller.RelayNotImplemented)
		// 图像编辑：在已有图片上根据提示进行修改（暂未实现）

		relayV1Router.POST("/images/variations", controller.RelayNotImplemented)
		// 图像变体：生成已有图片的变体版本（暂未实现）

		// ---------- Embeddings 向量嵌入 ----------
		relayV1Router.POST("/embeddings", controller.Relay)
		// 文本向量化：将文本转换为向量表示，用于搜索、聚类、分类等
		// https://platform.openai.com/docs/api-reference/embeddings

		relayV1Router.POST("/engines/:model/embeddings", controller.Relay)
		// 旧版 embeddings 端点（兼容早期 OpenAI API 格式）

		// ---------- Audio 语音 ----------
		relayV1Router.POST("/audio/transcriptions", controller.Relay)
		// 语音转文字（Whisper）：上传音频文件，返回文本转录
		// https://platform.openai.com/docs/api-reference/audio

		relayV1Router.POST("/audio/translations", controller.Relay)
		// 语音翻译：上传音频文件，翻译为英文文本

		relayV1Router.POST("/audio/speech", controller.Relay)
		// 文字转语音（TTS）：给定文本，生成语音音频

		// ---------- Files 文件管理 ----------
		// 以下均为占位路由，暂未实现
		relayV1Router.GET("/files", controller.RelayNotImplemented)             // 列出文件
		relayV1Router.POST("/files", controller.RelayNotImplemented)            // 上传文件
		relayV1Router.DELETE("/files/:id", controller.RelayNotImplemented)      // 删除文件
		relayV1Router.GET("/files/:id", controller.RelayNotImplemented)         // 查询文件信息
		relayV1Router.GET("/files/:id/content", controller.RelayNotImplemented) // 获取文件内容

		// ---------- Fine-tuning 微调 ----------
		// 以下均为占位路由，暂未实现
		relayV1Router.POST("/fine_tuning/jobs", controller.RelayNotImplemented)            // 创建微调任务
		relayV1Router.GET("/fine_tuning/jobs", controller.RelayNotImplemented)             // 列出微调任务
		relayV1Router.GET("/fine_tuning/jobs/:id", controller.RelayNotImplemented)         // 查询微调任务
		relayV1Router.POST("/fine_tuning/jobs/:id/cancel", controller.RelayNotImplemented) // 取消微调任务
		relayV1Router.GET("/fine_tuning/jobs/:id/events", controller.RelayNotImplemented)  // 查询微调事件

		relayV1Router.DELETE("/models/:model", controller.RelayNotImplemented)
		// 删除自定义微调模型（暂未实现）

		// ---------- Moderation 内容审核 ----------
		relayV1Router.POST("/moderations", controller.Relay)
		// 内容审核：检查文本是否包含违规内容（仇恨、暴力、色情等）
		// https://platform.openai.com/docs/api-reference/moderations

		// ---------- Assistants 助手 ----------
		// OpenAI Assistants API：可构建有状态的 AI 助手，支持工具调用和文件检索
		// 以下均为占位路由，暂未实现
		relayV1Router.POST("/assistants", controller.RelayNotImplemented)                     // 创建助手
		relayV1Router.GET("/assistants/:id", controller.RelayNotImplemented)                  // 查询助手
		relayV1Router.POST("/assistants/:id", controller.RelayNotImplemented)                 // 修改助手
		relayV1Router.DELETE("/assistants/:id", controller.RelayNotImplemented)               // 删除助手
		relayV1Router.GET("/assistants", controller.RelayNotImplemented)                      // 列出助手
		relayV1Router.POST("/assistants/:id/files", controller.RelayNotImplemented)           // 上传助手文件
		relayV1Router.GET("/assistants/:id/files/:fileId", controller.RelayNotImplemented)    // 查询助手文件
		relayV1Router.DELETE("/assistants/:id/files/:fileId", controller.RelayNotImplemented) // 删除助手文件
		relayV1Router.GET("/assistants/:id/files", controller.RelayNotImplemented)            // 列出助手文件

		// ---------- Threads 对话线程 ----------
		// Assistants API 的对话线程管理，助手在此线程中交互
		// 以下均为占位路由，暂未实现
		relayV1Router.POST("/threads", controller.RelayNotImplemented)                                       // 创建线程
		relayV1Router.GET("/threads/:id", controller.RelayNotImplemented)                                    // 查询线程
		relayV1Router.POST("/threads/:id", controller.RelayNotImplemented)                                   // 修改线程
		relayV1Router.DELETE("/threads/:id", controller.RelayNotImplemented)                                 // 删除线程
		relayV1Router.POST("/threads/:id/messages", controller.RelayNotImplemented)                          // 添加线程消息
		relayV1Router.GET("/threads/:id/messages/:messageId", controller.RelayNotImplemented)                // 查询线程消息
		relayV1Router.POST("/threads/:id/messages/:messageId", controller.RelayNotImplemented)               // 修改线程消息
		relayV1Router.GET("/threads/:id/messages/:messageId/files/:filesId", controller.RelayNotImplemented) // 查询消息文件
		relayV1Router.GET("/threads/:id/messages/:messageId/files", controller.RelayNotImplemented)          // 列出消息文件
		relayV1Router.POST("/threads/:id/runs", controller.RelayNotImplemented)                              // 创建运行
		relayV1Router.GET("/threads/:id/runs/:runsId", controller.RelayNotImplemented)                       // 查询运行
		relayV1Router.POST("/threads/:id/runs/:runsId", controller.RelayNotImplemented)                      // 修改运行
		relayV1Router.GET("/threads/:id/runs", controller.RelayNotImplemented)                               // 列出运行
		relayV1Router.POST("/threads/:id/runs/:runsId/submit_tool_outputs", controller.RelayNotImplemented)  // 提交工具输出
		relayV1Router.POST("/threads/:id/runs/:runsId/cancel", controller.RelayNotImplemented)               // 取消运行
		relayV1Router.GET("/threads/:id/runs/:runsId/steps/:stepId", controller.RelayNotImplemented)         // 查询运行步骤
		relayV1Router.GET("/threads/:id/runs/:runsId/steps", controller.RelayNotImplemented)                 // 列出运行步骤
	}
}
