package adaptor

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

// Adaptor 是上游 AI 服务提供商的适配器接口
// 每个上游服务商（如 OpenAI、Azure、Claude、百度文心等）都实现此接口，
// 从而将统一的内部请求格式转换为各服务商特有的协议格式。
//
// 调用流程（按方法调用顺序）：
//
//	Init → GetRequestURL → ConvertRequest → SetupRequestHeader → DoRequest → DoResponse
//
// 整体思路类似于适配器模式：one-api 对外暴露统一的 OpenAI 兼容 API，
// 对内通过 Adaptor 将请求转换为目标服务商的格式。
type Adaptor interface {
	// Init 初始化适配器，传入请求元信息（渠道、模型、模式等）
	// 在其他方法之前调用，后续方法可依赖 meta 中的信息
	Init(meta *meta.Meta)

	// GetRequestURL 根据元信息构建上游服务商的请求 URL
	// 不同服务商的 API 地址格式不同，由各适配器自行拼接
	GetRequestURL(meta *meta.Meta) (string, error)

	// SetupRequestHeader 设置发送给上游的 HTTP 请求头
	// 主要是设置 Authorization 认证信息（API Key / Access Token 等），
	// 不同服务商的认证方式不同（Bearer Token、API-Key 头、签名等）
	SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error

	// ConvertRequest 将统一的 OpenAI 格式请求转换为目标服务商的请求格式
	// relayMode 标识请求类型（聊天补全、文本补全、Embeddings 等）
	// 返回转换后的请求体（any 类型，因为各服务商的请求结构不同）
	ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error)

	// ConvertImageRequest 将统一的图片生成请求转换为目标服务商的格式
	// 图片生成接口与文本接口差异较大，因此单独提供转换方法
	ConvertImageRequest(request *model.ImageRequest) (any, error)

	// DoRequest 向上游服务商发送 HTTP 请求
	// requestBody 是序列化后的请求体，由调用方准备
	// 返回上游的 HTTP 响应，默认实现使用 http.Client 发送
	DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error)

	// DoResponse 处理上游服务商的响应
	// 负责解析响应体、处理流式(SSE)输出、提取 token 用量等
	// 返回 usage（用于计费）和 err（上游返回的错误信息）
	// 对于流式响应，此方法会逐块将数据写入 gin.Context 的 writer
	DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode)

	// GetModelList 返回该适配器（服务商渠道）支持的模型列表
	// 用于在 /v1/models 接口中聚合展示所有可用模型
	GetModelList() []string

	// GetChannelName 返回该适配器对应的渠道类型名称（如 "openai"、"azure"、"claude"）
	// 用于日志记录和错误提示
	GetChannelName() string
}
