package relaymode

// 转发模式枚举：根据请求路径确定使用哪种转发处理器
// 每种模式对应不同的请求/响应格式和转发逻辑（见 controller/relay.go 的 relayHelper）
const (
	Unknown            = iota // 0: 未知模式（无法识别的路径）
	ChatCompletions           // 1: 对话补全 → /v1/chat/completions（最常用）
	Completions               // 2: 文本补全 → /v1/completions（旧版）
	Embeddings                // 3: 向量嵌入 → /v1/embeddings
	Moderations               // 4: 内容审核 → /v1/moderations
	ImagesGenerations         // 5: 图像生成 → /v1/images/generations
	Edits                     // 6: 文本编辑 → /v1/edits（已弃用）
	AudioSpeech               // 7: 文字转语音 → /v1/audio/speech
	AudioTranscription        // 8: 语音转文字 → /v1/audio/transcriptions
	AudioTranslation          // 9: 语音翻译 → /v1/audio/translations
	Proxy                     // 10: 自定义代理 → /v1/oneapi/proxy/:channelid/*target（直接转发到指定渠道）
)
