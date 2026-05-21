package openai

import (
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/model"
)

// ResponseText2Usage 当渠道不提供使用信息时，从响应文本估算 token 使用量。
// 这是一个后备方法，用于处理不返回使用信息的渠道。
// 它计算响应文本中的 token 数，并将其添加到 prompt tokens 中。
func ResponseText2Usage(responseText string, modelName string, promptTokens int) *model.Usage {
	usage := &model.Usage{}
	usage.PromptTokens = promptTokens
	usage.CompletionTokens = CountTokenText(responseText, modelName)
	usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	return usage
}

// GetFullRequestURL 通过组合基础 URL 和请求路径来构建完整的请求 URL。
// 它处理不同渠道类型的特殊情况，特别是 Cloudflare AI Gateway。
func GetFullRequestURL(baseURL string, requestURL string, channelType int) string {
	if channelType == channeltype.OpenAICompatible {
		// OpenAI 兼容渠道不使用 /v1 前缀
		return fmt.Sprintf("%s%s", strings.TrimSuffix(baseURL, "/"), strings.TrimPrefix(requestURL, "/v1"))
	}
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, requestURL)

	// Cloudflare AI Gateway 的特殊处理
	if strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") {
		switch channelType {
		case channeltype.OpenAI:
			// Cloudflare 上的 OpenAI 移除 /v1 前缀
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/v1"))
		case channeltype.Azure:
			// Cloudflare 上的 Azure 移除 /openai/deployments 前缀
			fullRequestURL = fmt.Sprintf("%s%s", baseURL, strings.TrimPrefix(requestURL, "/openai/deployments"))
		}
	}
	return fullRequestURL
}