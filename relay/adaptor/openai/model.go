package openai

import "github.com/songquanpeng/one-api/relay/model"

// TextContent 表示消息中的文本内容。
// 用于聊天补全中的结构化内容类型。
type TextContent struct {
	Type string `json:"type,omitempty"` // 内容类型，通常为 "text"
	Text string `json:"text,omitempty"` // 实际文本内容
}

// ImageContent 表示消息中的图片内容。
// 用于包含图片的多模态消息。
type ImageContent struct {
	Type     string          `json:"type,omitempty"`      // 内容类型，通常为 "image_url"
	ImageURL *model.ImageURL `json:"image_url,omitempty"` // 图片 URL 或 base64 数据
}

// ChatRequest 表示聊天补全请求。
type ChatRequest struct {
	Model     string          `json:"model"`     // 模型标识符
	Messages  []model.Message `json:"messages"`  // 消息对象数组
	MaxTokens int             `json:"max_tokens"` // 要生成的最大 token 数
}

// TextRequest 表示文本补全请求。
type TextRequest struct {
	Model     string          `json:"model"`     // 模型标识符
	Messages  []model.Message `json:"messages"`  // 消息对象数组
	Prompt    string          `json:"prompt"`    // 文本提示词
	MaxTokens int             `json:"max_tokens"` // 要生成的最大 token 数
}

// ImageRequest 表示图片生成请求。
// 文档：https://platform.openai.com/docs/api-reference/images/create
type ImageRequest struct {
	Model          string `json:"model"`                    // 用于图片生成的模型
	Prompt         string `json:"prompt" binding:"required"` // 所需图片的文本描述
	N              int    `json:"n,omitempty"`              // 要生成的图片数量
	Size           string `json:"size,omitempty"`           // 图片大小（如 "1024x1024"）
	Quality        string `json:"quality,omitempty"`        // 图片质量（"standard" 或 "hd"）
	ResponseFormat string `json:"response_format,omitempty"` // 响应格式（"url" 或 "b64_json"）
	Style          string `json:"style,omitempty"`          // 图片风格（"vivid" 或 "natural"）
	User           string `json:"user,omitempty"`           // 终端用户的唯一标识符
}

// WhisperJSONResponse 表示简单的 Whisper 语音转文本响应。
type WhisperJSONResponse struct {
	Text string `json:"text,omitempty"` // 转录文本
}

// WhisperVerboseJSONResponse 表示详细的 Whisper 语音转文本响应。
type WhisperVerboseJSONResponse struct {
	Task     string    `json:"task,omitempty"`       // 执行的任务（如 "transcribe"）
	Language string    `json:"language,omitempty"`   // 检测到的语言
	Duration float64   `json:"duration,omitempty"`   // 音频时长（秒）
	Text     string    `json:"text,omitempty"`       // 转录文本
	Segments []Segment `json:"segments,omitempty"`   // 详细转录片段
}

// Segment 表示一段带有时间信息的转录音频。
type Segment struct {
	Id               int     `json:"id"`                 // 片段 ID
	Seek             int     `json:"seek"`               // 音频文件中的搜索位置
	Start            float64 `json:"start"`              // 片段开始时间
	End              float64 `json:"end"`                // 片段结束时间
	Text             string  `json:"text"`               // 此片段的转录文本
	Tokens           []int   `json:"tokens"`             // Token ID 列表
	Temperature      float64 `json:"temperature"`        // 采样温度
	AvgLogprob       float64 `json:"avg_logprob"`        // 平均对数概率
	CompressionRatio float64 `json:"compression_ratio"`  // 压缩比
	NoSpeechProb     float64 `json:"no_speech_prob"`     // 无语音概率
}

// TextToSpeechRequest 表示文本转语音请求。
type TextToSpeechRequest struct {
	Model          string  `json:"model" binding:"required"`          // 要使用的 TTS 模型
	Input          string  `json:"input" binding:"required"`          // 要转换为语音的文本
	Voice          string  `json:"voice" binding:"required"`          // 要使用的语音
	Speed          float64 `json:"speed"`                             // 语音速度（0.25 到 4.0）
	ResponseFormat string  `json:"response_format"`                   // 音频格式（mp3, opus, aac, flac）
}

// UsageOrResponseText 表示使用信息或响应文本。
// 用于灵活的响应处理。
type UsageOrResponseText struct {
	*model.Usage
	ResponseText string
}

// SlimTextResponse 表示用于解析的最小文本响应。
// 用于提取使用信息和错误信息，无需完整响应解析。
type SlimTextResponse struct {
	Choices     []TextResponseChoice `json:"choices"` // 响应选项
	model.Usage `json:"usage"`       // Token 使用信息
	Error       model.Error          `json:"error"`   // 如果有错误，返回错误信息
}

// TextResponseChoice 表示文本响应中的单个选项。
type TextResponseChoice struct {
	Index         int `json:"index"` // 选项索引
	model.Message `json:"message"` // 响应消息
	FinishReason  string `json:"finish_reason"` // 完成原因（stop、length 等）
}

// TextResponse 表示完整的文本补全响应。
type TextResponse struct {
	Id          string               `json:"id"`          // 唯一响应标识符
	Model       string               `json:"model,omitempty"` // 使用的模型
	Object      string               `json:"object"`      // 对象类型（如 "chat.completion"）
	Created     int64                `json:"created"`     // 创建时间戳
	Choices     []TextResponseChoice `json:"choices"`     // 响应选项
	model.Usage `json:"usage"`       // Token 使用信息
}

// EmbeddingResponseItem 表示单个嵌入向量。
type EmbeddingResponseItem struct {
	Object    string    `json:"object"`    // 对象类型（如 "embedding"）
	Index     int       `json:"index"`     // 项目索引
	Embedding []float64 `json:"embedding"` // 嵌入向量
}

// EmbeddingResponse 表示嵌入生成响应。
type EmbeddingResponse struct {
	Object      string                  `json:"object"` // 对象类型（如 "list"）
	Data        []EmbeddingResponseItem `json:"data"`   // 嵌入项目数组
	Model       string                  `json:"model"`  // 使用的模型
	model.Usage `json:"usage"`          // Token 使用信息
}

// ImageData 表示生成的图片。
type ImageData struct {
	Url           string `json:"url,omitempty"`           // 图片 URL
	B64Json       string `json:"b64_json,omitempty"`      // Base64 编码的图片
	RevisedPrompt string `json:"revised_prompt,omitempty"` // 修订后的提示词（如适用）
}

// ImageResponse 表示图片生成响应。
type ImageResponse struct {
	Created int64       `json:"created"` // 创建时间戳
	Data    []ImageData `json:"data"`    // 生成的图片数组
}

// ChatCompletionsStreamResponseChoice 表示流式聊天补全中的一个选项。
type ChatCompletionsStreamResponseChoice struct {
	Index        int           `json:"index"`        // 选项索引
	Delta        model.Message `json:"delta"`        // 增量消息内容
	FinishReason *string       `json:"finish_reason,omitempty"` // 如果完成，返回完成原因
}

// ChatCompletionsStreamResponse 表示流式聊天补全响应块。
type ChatCompletionsStreamResponse struct {
	Id      string                                `json:"id"`      // 流的唯一标识符
	Object  string                                `json:"object"`  // 对象类型（如 "chat.completion.chunk"）
	Created int64                                 `json:"created"` // 创建时间戳
	Model   string                                `json:"model"`   // 使用的模型
	Choices []ChatCompletionsStreamResponseChoice `json:"choices"` // 选项数组
	Usage   *model.Usage                          `json:"usage,omitempty"` // 使用信息（仅在最后一块中）
}

// CompletionsStreamResponse 表示流式文本补全响应块。
type CompletionsStreamResponse struct {
	Choices []struct {
		Text         string `json:"text"`         // 增量文本
		FinishReason string `json:"finish_reason"` // 如果完成，返回完成原因
	} `json:"choices"` // 选项数组
}