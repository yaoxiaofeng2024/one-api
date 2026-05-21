package relay

import (
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/aiproxy"
	"github.com/songquanpeng/one-api/relay/adaptor/ali"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/aws"
	"github.com/songquanpeng/one-api/relay/adaptor/baidu"
	"github.com/songquanpeng/one-api/relay/adaptor/cloudflare"
	"github.com/songquanpeng/one-api/relay/adaptor/cohere"
	"github.com/songquanpeng/one-api/relay/adaptor/coze"
	"github.com/songquanpeng/one-api/relay/adaptor/deepl"
	"github.com/songquanpeng/one-api/relay/adaptor/gemini"
	"github.com/songquanpeng/one-api/relay/adaptor/ollama"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/adaptor/palm"
	"github.com/songquanpeng/one-api/relay/adaptor/proxy"
	"github.com/songquanpeng/one-api/relay/adaptor/replicate"
	"github.com/songquanpeng/one-api/relay/adaptor/tencent"
	"github.com/songquanpeng/one-api/relay/adaptor/vertexai"
	"github.com/songquanpeng/one-api/relay/adaptor/xunfei"
	"github.com/songquanpeng/one-api/relay/adaptor/zhipu"
	"github.com/songquanpeng/one-api/relay/apitype"
)

func GetAdaptor(apiType int) adaptor.Adaptor {
	switch apiType {
	case apitype.AIProxyLibrary:
		// AI Proxy Library 代理库
		return &aiproxy.Adaptor{}
	case apitype.Ali:
		// 阿里云通义千问 (Qwen)
		return &ali.Adaptor{}
	case apitype.Anthropic:
		// Anthropic Claude 系列模型
		return &anthropic.Adaptor{}
	case apitype.AwsClaude:
		// AWS Bedrock 上的 Claude 模型
		return &aws.Adaptor{}
	case apitype.Baidu:
		// 百度文心一言 (ERNIE Bot)
		return &baidu.Adaptor{}
	case apitype.Gemini:
		// Google Gemini 系列模型
		return &gemini.Adaptor{}
	case apitype.OpenAI:
		// OpenAI GPT 系列模型
		return &openai.Adaptor{}
	case apitype.PaLM:
		// Google PaLM 系列模型
		return &palm.Adaptor{}
	case apitype.Tencent:
		// 腾讯混元大模型
		return &tencent.Adaptor{}
	case apitype.Xunfei:
		// 科大讯飞星火大模型
		return &xunfei.Adaptor{}
	case apitype.Zhipu:
		// 智谱AI GLM 系列模型
		return &zhipu.Adaptor{}
	case apitype.Ollama:
		// Ollama 本地运行的大模型
		return &ollama.Adaptor{}
	case apitype.Coze:
		// Coze 扣子平台
		return &coze.Adaptor{}
	case apitype.Cohere:
		// Cohere 大模型
		return &cohere.Adaptor{}
	case apitype.Cloudflare:
		// Cloudflare Workers AI
		return &cloudflare.Adaptor{}
	case apitype.DeepL:
		// DeepL 翻译服务
		return &deepl.Adaptor{}
	case apitype.VertexAI:
		// Google Vertex AI 平台
		return &vertexai.Adaptor{}
	case apitype.Proxy:
		// 通用代理适配器
		return &proxy.Adaptor{}
	case apitype.Replicate:
		// Replicate 平台模型
		return &replicate.Adaptor{}
	}
	return nil
}
