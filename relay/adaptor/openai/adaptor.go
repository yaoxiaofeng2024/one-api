// Package openai 提供 OpenAI 兼容 API 渠道的适配器实现。
// 该适配器处理遵循 OpenAI API 格式的各种渠道的请求和响应，
// 包括 OpenAI 官方、Azure OpenAI 以及其他兼容提供商。
package openai

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/alibailian"
	"github.com/songquanpeng/one-api/relay/adaptor/baiduv2"
	"github.com/songquanpeng/one-api/relay/adaptor/doubao"
	"github.com/songquanpeng/one-api/relay/adaptor/geminiv2"
	"github.com/songquanpeng/one-api/relay/adaptor/minimax"
	"github.com/songquanpeng/one-api/relay/adaptor/novita"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

// Adaptor 实现 OpenAI 兼容渠道的适配器接口。
// 它处理请求转换、URL 生成、请求头设置和响应处理。
type Adaptor struct {
	ChannelType int // 渠道类型（如 OpenAI、Azure 等）
}

// Init 使用渠道配置的元数据初始化适配器。
// 它存储渠道类型，供其他方法使用以确定行为。
func (a *Adaptor) Init(meta *meta.Meta) {
	a.ChannelType = meta.ChannelType
}

// GetRequestURL 根据渠道类型和请求元数据构建完整的请求 URL。
// 不同渠道有不同的 URL 格式，特别是 Azure 需要特殊处理部署名称和 API 版本。
func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	switch meta.ChannelType {
	case channeltype.Azure:
		// Azure OpenAI 图片生成有特殊的 URL 格式
		if meta.Mode == relaymode.ImagesGenerations {
			// https://learn.microsoft.com/en-us/azure/ai-services/openai/dall-e-quickstart?tabs=dalle3%2Ccommand-line&pivots=rest-api
			// https://{resource_name}.openai.azure.com/openai/deployments/dall-e-3/images/generations?api-version=2024-03-01-preview
			fullRequestURL := fmt.Sprintf("%s/openai/deployments/%s/images/generations?api-version=%s", meta.BaseURL, meta.ActualModelName, meta.Config.APIVersion)
			return fullRequestURL, nil
		}

		// Azure OpenAI chat/completions URL 格式
		// https://learn.microsoft.com/en-us/azure/cognitive-services/openai/chatgpt-quickstart?pivots=rest-api&tabs=command-line#rest-api
		requestURL := strings.Split(meta.RequestURLPath, "?")[0]
		requestURL = fmt.Sprintf("%s?api-version=%s", requestURL, meta.Config.APIVersion)
		task := strings.TrimPrefix(requestURL, "/v1/")
		model_ := meta.ActualModelName
		model_ = strings.Replace(model_, ".", "", -1) // 移除模型名称中的点（Azure 要求）
		//https://github.com/songquanpeng/one-api/issues/1191
		// {your endpoint}/openai/deployments/{your azure_model}/chat/completions?api-version={api_version}
		requestURL = fmt.Sprintf("/openai/deployments/%s/%s", model_, task)
		return GetFullRequestURL(meta.BaseURL, requestURL, meta.ChannelType), nil
	case channeltype.Minimax:
		return minimax.GetRequestURL(meta)
	case channeltype.Doubao:
		return doubao.GetRequestURL(meta)
	case channeltype.Novita:
		return novita.GetRequestURL(meta)
	case channeltype.BaiduV2:
		return baiduv2.GetRequestURL(meta)
	case channeltype.AliBailian:
		return alibailian.GetRequestURL(meta)
	case channeltype.GeminiOpenAICompatible:
		return geminiv2.GetRequestURL(meta)
	default:
		// 标准 OpenAI 兼容 URL 格式
		return GetFullRequestURL(meta.BaseURL, meta.RequestURLPath, meta.ChannelType), nil
	}
}

// SetupRequestHeader 配置出站请求的 HTTP 头。
// 它设置公共头和渠道特定的头（如认证信息）。
// Azure 使用 'api-key' 头，而其他渠道使用标准的 'Authorization: Bearer' 头。
func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	if meta.ChannelType == channeltype.Azure {
		// Azure OpenAI 使用 'api-key' 头而不是 Authorization
		req.Header.Set("api-key", meta.APIKey)
		return nil
	}
	// 标准 Bearer token 认证
	req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	if meta.ChannelType == channeltype.OpenRouter {
		// OpenRouter 需要特定的标识头
		req.Header.Set("HTTP-Referer", "https://github.com/songquanpeng/one-api")
		req.Header.Set("X-Title", "One API")
	}
	return nil
}

// ConvertRequest 将传入请求转换为目标渠道所需的格式。
// 对于 OpenAI 兼容渠道，这主要确保流式请求通过设置适当的流选项来包含使用信息。
func (a *Adaptor) ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	if request.Stream {
		// 在流式模式下始终返回使用信息以进行准确的 token 追踪
		if request.StreamOptions == nil {
			request.StreamOptions = &model.StreamOptions{}
		}
		request.StreamOptions.IncludeUsage = true
	}
	return request, nil
}

// ConvertImageRequest 将图片生成请求转换为目标渠道格式。
// 对于 OpenAI 兼容渠道，请求格式已经兼容。
func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

// DoRequest 执行到目标渠道的 HTTP 请求。
// 它使用辅助函数执行实际的 HTTP 请求（包含重试逻辑）。
func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

// DoResponse 处理来自渠道的 HTTP 响应。
// 它处理流式和非流式响应，提取使用信息并处理错误。
// 对于流式响应，如果未提供使用信息，可能会进行估算。
func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	if meta.IsStream {
		// 处理流式响应
		var responseText string
		err, responseText, usage = StreamHandler(c, resp, meta.Mode)
		// 如果未提供使用信息，从响应文本估算
		if usage == nil || usage.TotalTokens == 0 {
			usage = ResponseText2Usage(responseText, meta.ActualModelName, meta.PromptTokens)
		}
		// 某些渠道不单独返回 prompt/completion tokens
		if usage.TotalTokens != 0 && usage.PromptTokens == 0 {
			usage.PromptTokens = meta.PromptTokens
			usage.CompletionTokens = usage.TotalTokens - meta.PromptTokens
		}
	} else {
		// 处理非流式响应
		switch meta.Mode {
		case relaymode.ImagesGenerations:
			err, _ = ImageHandler(c, resp)
		default:
			err, usage = Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
		}
	}
	return
}

// GetModelList 返回此渠道支持的模型列表。
// 它根据渠道类型获取兼容的模型。
func (a *Adaptor) GetModelList() []string {
	_, modelList := GetCompatibleChannelMeta(a.ChannelType)
	return modelList
}

// GetChannelName 返回渠道的可读名称。
// 用于在 UI 和日志中显示。
func (a *Adaptor) GetChannelName() string {
	channelName, _ := GetCompatibleChannelMeta(a.ChannelType)
	return channelName
}