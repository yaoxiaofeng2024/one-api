package ratio

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/songquanpeng/one-api/common/logger"
)

const (
	// USD2RMB 是美元兑人民币的汇率，$1 = ￥7
	USD2RMB = 7

	// USD 是 $1 对应的内部计费单位数。
	// 内部计费单位的基准：1 内部单位 = $0.002 / 1K tokens（即原始 GPT-3 的输入 token 单价）。
	// 因此 $1 = 1 / 0.002 = 500 内部单位。
	// 换算示例：gpt-4 的 ModelRatio = 15，表示 $0.002 × 15 = $0.03 / 1K tokens
	USD = 500

	// MILLI_USD 是 $0.001 对应的内部计费单位数。
	// MILLI_USD = $0.001 / $0.002 = 0.5 内部单位
	// 用于定价以 $/1M tokens 为单位的模型（如 deepseek-chat: $0.14/1M tokens → 0.14 * MILLI_USD）
	MILLI_USD = 1.0 / 1000 * USD

	// RMB 是 ￥1 对应的内部计费单位数。
	// RMB = USD / USD2RMB = 500 / 7 ≈ 71.43 内部单位
	// 换算示例：ERNIE-4.0-8K 定价 ￥0.120 / 1K tokens → 0.120 * RMB ≈ 8.571 内部单位
	RMB = USD / USD2RMB
)

// modelRatioLock 用于保护 ModelRatio 的并发读写安全（运行时可通过 API 动态更新）
var modelRatioLock sync.RWMutex

// ModelRatio 存储各模型的输入 token 计费比率（内部单位）。
//
// 【内部单位体系】
//   - 1 内部单位 = $0.002 / 1K input tokens（原始 GPT-3 定价作为基准）
//   - 值 = 模型的输入 token 单价 / $0.002
//   - 例如：gpt-4 值为 15 → 输入单价 = $0.002 × 15 = $0.03 / 1K tokens
//
// 【值的表达方式说明】
//   - 裸数字（如 15）：直接表示 $0.002 的倍数，适用于按 $/1K tokens 定价的模型
//   - X * USD：X 是美元金额（$X / 1K tokens 或 $X / image），乘以 USD(500) 转换为内部单位
//     例如："dall-e-2": 0.02 * USD → $0.02/图 × 500 = 10 内部单位
//   - X * MILLI_USD：X 是以 $/1M tokens 为单位的单价，乘以 MILLI_USD(0.5) 转换
//     例如："deepseek-chat": 0.14 * MILLI_USD → $0.14/1M tokens → 0.07 内部单位
//   - X * RMB：X 是以 ￥/1K tokens 为单位的单价，乘以 RMB(≈71.43) 转换为内部单位
//     例如："ERNIE-4.0-8K": 0.120 * RMB → ￥0.120/1K tokens ≈ 8.571 内部单位
//   - X / 1000 * USD：X 是以 $/1K tokens 为单位的单价（用于避免与裸数字混淆的写法）
//     例如："claude-3-opus": 15.0 / 1000 * USD → $15/1K tokens → 7.5 内部单位
//   - X / 1000000 * USD：X 是以 $/1M tokens 为单位的单价
//     例如："llama-3.1-70b-versatile": 0.59 / 1000000 * USD → $0.59/1M tokens
//   - X / 1000 * RMB：X 是以 ￥/1K tokens 为单位的单价
//     例如："yi-34b-chat-0205": 2.5 / 1000 * RMB → ￥2.5/1K tokens
//   - X / Y：直接按原始美元单价除以基准 $0.002，用于 AWS 等渠道的特殊定价
//     例如："llama3-8b-8192(33)": 0.0003 / 0.002 → $0.0003 / 1K tokens = 0.15 内部单位
//
// 【注意】此值仅代表输入(input) token 的计费比率，输出(completion) token 的计费比率见 CompletionRatio。
//
// 参考链接：
// https://platform.openai.com/docs/models/model-endpoint-compatibility
// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Blfmc9dlf
// https://openai.com/pricing
var ModelRatio = map[string]float64{
	// https://openai.com/pricing
	// OpenAI 模型定价（裸数字 = $0.002 的倍数，即 $/1K tokens ÷ $0.002）
	"gpt-4":                  15, // $0.03 / 1K tokens → 15 × $0.002
	"gpt-4-0314":             15,
	"gpt-4-0613":             15,
	"gpt-4-32k":              30, // $0.06 / 1K tokens → 30 × $0.002
	"gpt-4-32k-0314":         30,
	"gpt-4-32k-0613":         30,
	"gpt-4-1106-preview":     5,     // $0.01 / 1K tokens → 5 × $0.002
	"gpt-4-0125-preview":     5,     // $0.01 / 1K tokens → 5 × $0.002
	"gpt-4-turbo-preview":    5,     // $0.01 / 1K tokens → 5 × $0.002
	"gpt-4-turbo":            5,     // $0.01 / 1K tokens → 5 × $0.002
	"gpt-4-turbo-2024-04-09": 5,     // $0.01 / 1K tokens → 5 × $0.002
	"gpt-4o":                 2.5,   // $0.005 / 1K tokens → 2.5 × $0.002
	"chatgpt-4o-latest":      2.5,   // $0.005 / 1K tokens → 2.5 × $0.002
	"gpt-4o-2024-05-13":      2.5,   // $0.005 / 1K tokens → 2.5 × $0.002
	"gpt-4o-2024-08-06":      1.25,  // $0.0025 / 1K tokens → 1.25 × $0.002
	"gpt-4o-2024-11-20":      1.25,  // $0.0025 / 1K tokens → 1.25 × $0.002
	"gpt-4o-mini":            0.075, // $0.00015 / 1K tokens → 0.075 × $0.002
	"gpt-4o-mini-2024-07-18": 0.075, // $0.00015 / 1K tokens → 0.075 × $0.002
	"gpt-4-vision-preview":   5,     // $0.01 / 1K tokens → 5 × $0.002
	"gpt-3.5-turbo":          0.25,  // $0.0005 / 1K tokens → 0.25 × $0.002
	"gpt-3.5-turbo-0301":     0.75,  // $0.0015 / 1K tokens → 0.75 × $0.002
	"gpt-3.5-turbo-0613":     0.75,  // $0.0015 / 1K tokens → 0.75 × $0.002
	"gpt-3.5-turbo-16k":      1.5,   // $0.003 / 1K tokens → 1.5 × $0.002
	"gpt-3.5-turbo-16k-0613": 1.5,   // $0.003 / 1K tokens → 1.5 × $0.002
	"gpt-3.5-turbo-instruct": 0.75,  // $0.0015 / 1K tokens → 0.75 × $0.002
	"gpt-3.5-turbo-1106":     0.5,   // $0.001 / 1K tokens → 0.5 × $0.002
	"gpt-3.5-turbo-0125":     0.25,  // $0.0005 / 1K tokens → 0.25 × $0.002
	// o1/o3 系列定价以 $/1M tokens 为单位，转换为 $/1K tokens 后再计算倍数
	"o1":                    7.5, // $15.00 / 1M input tokens → $0.015 / 1K tokens → 7.5 × $0.002
	"o1-2024-12-17":         7.5,
	"o1-preview":            7.5, // $15.00 / 1M input tokens → 7.5 × $0.002
	"o1-preview-2024-09-12": 7.5,
	"o1-mini":               1.5, // $3.00 / 1M input tokens → $0.003 / 1K tokens → 1.5 × $0.002
	"o1-mini-2024-09-12":    1.5,
	"o3-mini":               1.5, // $3.00 / 1M input tokens → 1.5 × $0.002
	"o3-mini-2025-01-31":    1.5,
	"davinci-002":           1,    // $0.002 / 1K tokens → 1 × $0.002
	"babbage-002":           0.2,  // $0.0004 / 1K tokens → 0.2 × $0.002
	"text-ada-001":          0.2,  // $0.0004 / 1K tokens → 0.2 × $0.002
	"text-babbage-001":      0.25, // $0.0005 / 1K tokens → 0.25 × $0.002
	"text-curie-001":        1,    // $0.002 / 1K tokens → 1 × $0.002
	"text-davinci-002":      10,   // $0.02 / 1K tokens → 10 × $0.002
	"text-davinci-003":      10,   // $0.02 / 1K tokens → 10 × $0.002
	"text-davinci-edit-001": 10,
	"code-davinci-edit-001": 10,
	// whisper 按音频时长计费，经换算后约 $0.03 / 1K tokens → 15 × $0.002
	"whisper-1": 15, // $0.006 / minute → 约 $0.03 / 1K tokens
	// tts 按字符计费，tts-1: $0.015/1K chars → 约 7.5 × $0.002; tts-1-hd: $0.030/1K chars → 约 15 × $0.002
	"tts-1":                   7.5, // $0.015 / 1K characters → 约 $0.015 / 1K tokens
	"tts-1-1106":              7.5,
	"tts-1-hd":                15, // $0.030 / 1K characters → 约 $0.030 / 1K tokens
	"tts-1-hd-1106":           15,
	"davinci":                 10,
	"curie":                   10,
	"babbage":                 10,
	"ada":                     10,
	"text-embedding-ada-002":  0.05,  // $0.0001 / 1K tokens → 0.05 × $0.002
	"text-embedding-3-small":  0.01,  // $0.00002 / 1K tokens → 0.01 × $0.002
	"text-embedding-3-large":  0.065, // $0.00013 / 1K tokens → 0.065 × $0.002
	"text-search-ada-doc-001": 10,
	"text-moderation-stable":  0.1, // $0.0002 / 1K tokens → 0.1 × $0.002
	"text-moderation-latest":  0.1, // $0.0002 / 1K tokens → 0.1 × $0.002
	// 图像生成模型按次/张计费，0.02 * USD = $0.02 × 500 = 10 内部单位
	"dall-e-2": 0.02 * USD, // $0.016 - $0.020 / image → 约 10 内部单位
	"dall-e-3": 0.04 * USD, // $0.040 - $0.120 / image → 约 20 内部单位
	// https://docs.anthropic.com/en/docs/about-claude/models
	// Claude 模型定价以 $/1M tokens 为单位，使用 X / 1000 * USD 转换
	// 计算方式：$X/1M tokens = $X/1000/1K tokens → X/1000 * 500 = X * 0.5 内部单位
	"claude-instant-1.2":         0.8 / 1000 * USD,  // $0.8/1M tokens → 0.4 内部单位
	"claude-2.0":                 8.0 / 1000 * USD,  // $8/1M tokens → 4 内部单位
	"claude-2.1":                 8.0 / 1000 * USD,  // $8/1M tokens → 4 内部单位
	"claude-3-haiku-20240307":    0.25 / 1000 * USD, // $0.25/1M tokens → 0.125 内部单位
	"claude-3-5-haiku-20241022":  1.0 / 1000 * USD,  // $1/1M tokens → 0.5 内部单位
	"claude-3-5-haiku-latest":    1.0 / 1000 * USD,  // $1/1M tokens → 0.5 内部单位
	"claude-3-sonnet-20240229":   3.0 / 1000 * USD,  // $3/1M tokens → 1.5 内部单位
	"claude-3-5-sonnet-20240620": 3.0 / 1000 * USD,  // $3/1M tokens → 1.5 内部单位
	"claude-3-5-sonnet-20241022": 3.0 / 1000 * USD,  // $3/1M tokens → 1.5 内部单位
	"claude-3-5-sonnet-latest":   3.0 / 1000 * USD,  // $3/1M tokens → 1.5 内部单位
	"claude-3-opus-20240229":     15.0 / 1000 * USD, // $15/1M tokens → 7.5 内部单位
	// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/hlrk4akp7
	// 百度文心模型定价以 ￥/1K tokens 为单位，使用 X * RMB 转换
	// 计算方式：￥X/1K tokens → X * (500/7) ≈ X * 71.43 内部单位
	"ERNIE-4.0-8K":       0.120 * RMB, // ￥0.120 / 1K tokens ≈ 8.571 内部单位
	"ERNIE-3.5-8K":       0.012 * RMB, // ￥0.012 / 1K tokens ≈ 0.857 内部单位
	"ERNIE-3.5-8K-0205":  0.024 * RMB, // ￥0.024 / 1K tokens ≈ 1.714 内部单位
	"ERNIE-3.5-8K-1222":  0.012 * RMB, // ￥0.012 / 1K tokens ≈ 0.857 内部单位
	"ERNIE-Bot-8K":       0.024 * RMB, // ￥0.024 / 1K tokens ≈ 1.714 内部单位
	"ERNIE-3.5-4K-0205":  0.012 * RMB, // ￥0.012 / 1K tokens ≈ 0.857 内部单位
	"ERNIE-Speed-8K":     0.004 * RMB, // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"ERNIE-Speed-128K":   0.004 * RMB, // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"ERNIE-Lite-8K-0922": 0.008 * RMB, // ￥0.008 / 1K tokens ≈ 0.571 内部单位
	"ERNIE-Lite-8K-0308": 0.003 * RMB, // ￥0.003 / 1K tokens ≈ 0.214 内部单位
	"ERNIE-Tiny-8K":      0.001 * RMB, // ￥0.001 / 1K tokens ≈ 0.071 内部单位
	"BLOOMZ-7B":          0.004 * RMB, // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"Embedding-V1":       0.002 * RMB, // ￥0.002 / 1K tokens ≈ 0.143 内部单位
	"bge-large-zh":       0.002 * RMB, // ￥0.002 / 1K tokens ≈ 0.143 内部单位
	"bge-large-en":       0.002 * RMB, // ￥0.002 / 1K tokens ≈ 0.143 内部单位
	"tao-8k":             0.002 * RMB, // ￥0.002 / 1K tokens ≈ 0.143 内部单位
	// https://ai.google.dev/pricing
	// https://cloud.google.com/vertex-ai/generative-ai/pricing
	// Gemini 模型定价以 $/1M tokens 为单位，使用 X * MILLI_USD 转换
	// MILLI_USD = 0.5，所以 X * MILLI_USD = X * 0.5 内部单位
	// 计算方式：$X/1M tokens → X * 0.5 内部单位
	// "gemma-2-2b-it":                       0,
	// "gemma-2-9b-it":                       0,
	// "gemma-2-27b-it":                      0,
	"gemini-pro":                          0.25 * MILLI_USD,   // $0.25/1M tokens → 0.125 内部单位
	"gemini-1.0-pro":                      0.125 * MILLI_USD,  // $0.125/1M tokens → 0.0625 内部单位
	"gemini-1.5-pro":                      1.25 * MILLI_USD,   // $1.25/1M tokens → 0.625 内部单位
	"gemini-1.5-pro-001":                  1.25 * MILLI_USD,   // $1.25/1M tokens → 0.625 内部单位
	"gemini-1.5-pro-experimental":         1.25 * MILLI_USD,   // $1.25/1M tokens → 0.625 内部单位
	"gemini-1.5-flash":                    0.075 * MILLI_USD,  // $0.075/1M tokens → 0.0375 内部单位
	"gemini-1.5-flash-001":                0.075 * MILLI_USD,  // $0.075/1M tokens → 0.0375 内部单位
	"gemini-1.5-flash-8b":                 0.0375 * MILLI_USD, // $0.0375/1M tokens → 0.01875 内部单位
	"gemini-2.0-flash-exp":                0.075 * MILLI_USD,  // $0.075/1M tokens → 0.0375 内部单位
	"gemini-2.0-flash":                    0.15 * MILLI_USD,   // $0.15/1M tokens → 0.075 内部单位
	"gemini-2.0-flash-001":                0.15 * MILLI_USD,   // $0.15/1M tokens → 0.075 内部单位
	"gemini-2.0-flash-lite-preview-02-05": 0.075 * MILLI_USD,  // $0.075/1M tokens → 0.0375 内部单位
	"gemini-2.0-flash-thinking-exp-01-21": 0.075 * MILLI_USD,  // $0.075/1M tokens → 0.0375 内部单位
	"gemini-2.0-pro-exp-02-05":            1.25 * MILLI_USD,   // $1.25/1M tokens → 0.625 内部单位
	"aqa":                                 1,                  // $0.002 / 1K tokens → 1 × $0.002
	// https://open.bigmodel.cn/pricing
	// 智谱 GLM 模型定价以 ￥/1K tokens 为单位，使用 X * RMB 转换
	"glm-zero-preview": 0.01 * RMB, // ￥0.01 / 1K tokens ≈ 0.714 内部单位
	"glm-4-plus":       0.05 * RMB,
	"glm-4-0520":       0.1 * RMB,
	"glm-4-airx":       0.01 * RMB,
	"glm-4-air":        0.0005 * RMB,
	"glm-4-long":       0.001 * RMB,
	"glm-4-flashx":     0.0001 * RMB,
	"glm-4-flash":      0,
	"glm-4":            0.1 * RMB,   // deprecated model, available until 2025/06
	"glm-3-turbo":      0.001 * RMB, // deprecated model, available until 2025/06
	"glm-4v-plus":      0.004 * RMB,
	"glm-4v":           0.05 * RMB,
	"glm-4v-flash":     0,
	"cogview-3-plus":   0.06 * RMB,
	"cogview-3":        0.1 * RMB,
	"cogview-3-flash":  0,
	"cogviewx":         0.5 * RMB,
	"cogviewx-flash":   0,
	"charglm-4":        0.001 * RMB,
	"emohaa":           0.015 * RMB,
	"codegeex-4":       0.0001 * RMB,
	"embedding-2":      0.0005 * RMB,
	"embedding-3":      0.0005 * RMB,
	// https://help.aliyun.com/zh/dashscope/developer-reference/tongyi-thousand-questions-metering-and-billing
	// 阿里通义千问模型定价以 ￥/1K tokens 为单位，使用 X * RMB 转换
	"qwen-turbo":           0.0003 * RMB, // ￥0.0003 / 1K tokens ≈ 0.021 内部单位
	"qwen-turbo-latest":    0.0003 * RMB,
	"qwen-plus":            0.0008 * RMB,
	"qwen-plus-latest":     0.0008 * RMB,
	"qwen-max":             0.0024 * RMB,
	"qwen-max-latest":      0.0024 * RMB,
	"qwen-max-longcontext": 0.0005 * RMB,
	"qwen-vl-max":          0.003 * RMB,
	"qwen-vl-max-latest":   0.003 * RMB,
	"qwen-vl-plus":         0.0015 * RMB,
	"qwen-vl-plus-latest":  0.0015 * RMB,
	"qwen-vl-ocr":          0.005 * RMB,
	"qwen-vl-ocr-latest":   0.005 * RMB,
	// qwen-audio-turbo 特殊定价，1.4286 直接是内部单位值（约 ￥0.02/1K tokens）
	"qwen-audio-turbo":            1.4286, // ￥0.018 - ￥0.02 / 1K tokens
	"qwen-math-plus":              0.004 * RMB,
	"qwen-math-plus-latest":       0.004 * RMB,
	"qwen-math-turbo":             0.002 * RMB,
	"qwen-math-turbo-latest":      0.002 * RMB,
	"qwen-coder-plus":             0.0035 * RMB,
	"qwen-coder-plus-latest":      0.0035 * RMB,
	"qwen-coder-turbo":            0.002 * RMB,
	"qwen-coder-turbo-latest":     0.002 * RMB,
	"qwen-mt-plus":                0.015 * RMB,
	"qwen-mt-turbo":               0.001 * RMB,
	"qwq-32b-preview":             0.002 * RMB,
	"qwen2.5-72b-instruct":        0.004 * RMB,
	"qwen2.5-32b-instruct":        0.03 * RMB,
	"qwen2.5-14b-instruct":        0.001 * RMB,
	"qwen2.5-7b-instruct":         0.0005 * RMB,
	"qwen2.5-3b-instruct":         0.006 * RMB,
	"qwen2.5-1.5b-instruct":       0.0003 * RMB,
	"qwen2.5-0.5b-instruct":       0.0003 * RMB,
	"qwen2-72b-instruct":          0.004 * RMB,
	"qwen2-57b-a14b-instruct":     0.0035 * RMB,
	"qwen2-7b-instruct":           0.001 * RMB,
	"qwen2-1.5b-instruct":         0.001 * RMB,
	"qwen2-0.5b-instruct":         0.001 * RMB,
	"qwen1.5-110b-chat":           0.007 * RMB,
	"qwen1.5-72b-chat":            0.005 * RMB,
	"qwen1.5-32b-chat":            0.0035 * RMB,
	"qwen1.5-14b-chat":            0.002 * RMB,
	"qwen1.5-7b-chat":             0.001 * RMB,
	"qwen1.5-1.8b-chat":           0.001 * RMB,
	"qwen1.5-0.5b-chat":           0.001 * RMB,
	"qwen-72b-chat":               0.02 * RMB,
	"qwen-14b-chat":               0.008 * RMB,
	"qwen-7b-chat":                0.006 * RMB,
	"qwen-1.8b-chat":              0.006 * RMB,
	"qwen-1.8b-longcontext-chat":  0.006 * RMB,
	"qvq-72b-preview":             0.012 * RMB,
	"qwen2.5-vl-72b-instruct":     0.016 * RMB,
	"qwen2.5-vl-7b-instruct":      0.002 * RMB,
	"qwen2.5-vl-3b-instruct":      0.0012 * RMB,
	"qwen2-vl-7b-instruct":        0.016 * RMB,
	"qwen2-vl-2b-instruct":        0.002 * RMB,
	"qwen-vl-v1":                  0.002 * RMB,
	"qwen-vl-chat-v1":             0.002 * RMB,
	"qwen2-audio-instruct":        0.002 * RMB,
	"qwen-audio-chat":             0.002 * RMB,
	"qwen2.5-math-72b-instruct":   0.004 * RMB,
	"qwen2.5-math-7b-instruct":    0.001 * RMB,
	"qwen2.5-math-1.5b-instruct":  0.001 * RMB,
	"qwen2-math-72b-instruct":     0.004 * RMB,
	"qwen2-math-7b-instruct":      0.001 * RMB,
	"qwen2-math-1.5b-instruct":    0.001 * RMB,
	"qwen2.5-coder-32b-instruct":  0.002 * RMB,
	"qwen2.5-coder-14b-instruct":  0.002 * RMB,
	"qwen2.5-coder-7b-instruct":   0.001 * RMB,
	"qwen2.5-coder-3b-instruct":   0.001 * RMB,
	"qwen2.5-coder-1.5b-instruct": 0.001 * RMB,
	"qwen2.5-coder-0.5b-instruct": 0.001 * RMB,
	"text-embedding-v1":           0.0007 * RMB, // ￥0.0007 / 1K tokens ≈ 0.05 内部单位
	"text-embedding-v3":           0.0007 * RMB, // ￥0.0007 / 1K tokens
	"text-embedding-v2":           0.0007 * RMB, // ￥0.0007 / 1K tokens
	"text-embedding-async-v2":     0.0007 * RMB, // ￥0.0007 / 1K tokens
	"text-embedding-async-v1":     0.0007 * RMB, // ￥0.0007 / 1K tokens
	// 阿里图像生成模型，按次计费，直接使用内部单位值
	"ali-stable-diffusion-xl":   8.00, // 约 8 内部单位/次
	"ali-stable-diffusion-v1.5": 8.00, // 约 8 内部单位/次
	"wanx-v1":                   8.00, // 约 8 内部单位/次
	// DeepSeek 模型定价以 ￥/1K tokens 为单位
	"deepseek-r1":                   0.002 * RMB,  // ￥0.002 / 1K tokens ≈ 0.143 内部单位
	"deepseek-v3":                   0.001 * RMB,  // ￥0.001 / 1K tokens ≈ 0.071 内部单位
	"deepseek-r1-distill-qwen-1.5b": 0.001 * RMB,  // ￥0.001 / 1K tokens
	"deepseek-r1-distill-qwen-7b":   0.0005 * RMB, // ￥0.0005 / 1K tokens
	"deepseek-r1-distill-qwen-14b":  0.001 * RMB,  // ￥0.001 / 1K tokens
	"deepseek-r1-distill-qwen-32b":  0.002 * RMB,  // ￥0.002 / 1K tokens
	"deepseek-r1-distill-llama-8b":  0.0005 * RMB, // ￥0.0005 / 1K tokens
	"deepseek-r1-distill-llama-70b": 0.004 * RMB,  // ￥0.004 / 1K tokens
	// 讯飞星火模型，直接使用内部单位值
	"SparkDesk":           1.2858, // ￥0.018 / 1K tokens → 约 1.2858 内部单位
	"SparkDesk-v1.1":      1.2858, // ￥0.018 / 1K tokens
	"SparkDesk-v2.1":      1.2858, // ￥0.018 / 1K tokens
	"SparkDesk-v3.1":      1.2858, // ￥0.018 / 1K tokens
	"SparkDesk-v3.1-128K": 1.2858, // ￥0.018 / 1K tokens
	"SparkDesk-v3.5":      1.2858, // ￥0.018 / 1K tokens
	"SparkDesk-v3.5-32K":  1.2858, // ￥0.018 / 1K tokens
	"SparkDesk-v4.0":      1.2858, // ￥0.018 / 1K tokens
	// 360 智脑模型，直接使用内部单位值
	"360GPT_S2_V9":              0.8572, // ￥0.012 / 1K tokens → 约 0.8572 内部单位
	"embedding-bert-512-v1":     0.0715, // ￥0.001 / 1K tokens → 约 0.0715 内部单位
	"embedding_s1_v1":           0.0715, // ￥0.001 / 1K tokens
	"semantic_similarity_s1_v1": 0.0715, // ￥0.001 / 1K tokens
	// https://cloud.tencent.com/document/product/1729/97731#e0e6be58-60c8-469f-bdeb-6c264ce3b4d0
	// 腾讯混元模型定价以 ￥/1K tokens 为单位
	"hunyuan-turbo":             0.015 * RMB,  // ￥0.015 / 1K tokens ≈ 1.071 内部单位
	"hunyuan-large":             0.004 * RMB,  // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"hunyuan-large-longcontext": 0.006 * RMB,  // ￥0.006 / 1K tokens ≈ 0.429 内部单位
	"hunyuan-standard":          0.0008 * RMB, // ￥0.0008 / 1K tokens ≈ 0.057 内部单位
	"hunyuan-standard-256K":     0.0005 * RMB, // ￥0.0005 / 1K tokens ≈ 0.036 内部单位
	"hunyuan-translation-lite":  0.005 * RMB,  // ￥0.005 / 1K tokens ≈ 0.357 内部单位
	"hunyuan-role":              0.004 * RMB,  // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"hunyuan-functioncall":      0.004 * RMB,  // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"hunyuan-code":              0.004 * RMB,  // ￥0.004 / 1K tokens ≈ 0.286 内部单位
	"hunyuan-turbo-vision":      0.08 * RMB,   // ￥0.08 / 1K tokens ≈ 5.714 内部单位
	"hunyuan-vision":            0.018 * RMB,  // ￥0.018 / 1K tokens ≈ 1.286 内部单位
	"hunyuan-embedding":         0.0007 * RMB, // ￥0.0007 / 1K tokens ≈ 0.05 内部单位
	// https://platform.moonshot.cn/pricing
	// Moonshot（月之暗面）模型定价以 ￥/1K tokens 为单位
	"moonshot-v1-8k":   0.012 * RMB, // ￥0.012 / 1K tokens ≈ 0.857 内部单位
	"moonshot-v1-32k":  0.024 * RMB, // ￥0.024 / 1K tokens ≈ 1.714 内部单位
	"moonshot-v1-128k": 0.06 * RMB,  // ￥0.06 / 1K tokens ≈ 4.286 内部单位
	// https://platform.baichuan-ai.com/price
	// 百川模型定价以 ￥/1K tokens 为单位
	"Baichuan2-Turbo":      0.008 * RMB, // ￥0.008 / 1K tokens ≈ 0.571 内部单位
	"Baichuan2-Turbo-192k": 0.016 * RMB, // ￥0.016 / 1K tokens ≈ 1.143 内部单位
	"Baichuan2-53B":        0.02 * RMB,  // ￥0.02 / 1K tokens ≈ 1.429 内部单位
	// https://api.minimax.chat/document/price
	// MiniMax 模型定价以 ￥/1K tokens 为单位
	"abab6.5-chat":  0.03 * RMB,  // ￥0.03 / 1K tokens ≈ 2.143 内部单位
	"abab6.5s-chat": 0.01 * RMB,  // ￥0.01 / 1K tokens ≈ 0.714 内部单位
	"abab6-chat":    0.1 * RMB,   // ￥0.1 / 1K tokens ≈ 7.143 内部单位
	"abab5.5-chat":  0.015 * RMB, // ￥0.015 / 1K tokens ≈ 1.071 内部单位
	"abab5.5s-chat": 0.005 * RMB, // ￥0.005 / 1K tokens ≈ 0.357 内部单位
	// https://docs.mistral.ai/platform/pricing/
	// Mistral 模型定价以 $/1M tokens 为单位，使用 X / 1000 * USD 转换
	"open-mistral-7b":       0.25 / 1000 * USD, // $0.25/1M tokens → 0.125 内部单位
	"open-mixtral-8x7b":     0.7 / 1000 * USD,  // $0.7/1M tokens → 0.35 内部单位
	"mistral-small-latest":  2.0 / 1000 * USD,  // $2/1M tokens → 1 内部单位
	"mistral-medium-latest": 2.7 / 1000 * USD,  // $2.7/1M tokens → 1.35 内部单位
	"mistral-large-latest":  8.0 / 1000 * USD,  // $8/1M tokens → 4 内部单位
	"mistral-embed":         0.1 / 1000 * USD,  // $0.1/1M tokens → 0.05 内部单位
	// https://wow.groq.com/#:~:text=inquiries%C2%A0here.-,Model,-Current%20Speed
	// Groq 模型定价以 $/1M tokens 为单位，使用 X / 1000000 * USD 转换
	// 计算方式：$X/1M tokens → X/1000000 * 500 = X * 0.0005 内部单位
	"gemma-7b-it":                           0.07 / 1000000 * USD, // $0.07/1M tokens → 0.035 内部单位
	"gemma2-9b-it":                          0.20 / 1000000 * USD, // $0.20/1M tokens → 0.1 内部单位
	"llama-3.1-70b-versatile":               0.59 / 1000000 * USD, // $0.59/1M tokens → 0.295 内部单位
	"llama-3.1-8b-instant":                  0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama-3.2-11b-text-preview":            0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama-3.2-11b-vision-preview":          0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama-3.2-1b-preview":                  0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama-3.2-3b-preview":                  0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama-3.2-90b-text-preview":            0.59 / 1000000 * USD, // $0.59/1M tokens → 0.295 内部单位
	"llama-guard-3-8b":                      0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama3-70b-8192":                       0.59 / 1000000 * USD, // $0.59/1M tokens → 0.295 内部单位
	"llama3-8b-8192":                        0.05 / 1000000 * USD, // $0.05/1M tokens → 0.025 内部单位
	"llama3-groq-70b-8192-tool-use-preview": 0.89 / 1000000 * USD, // $0.89/1M tokens → 0.445 内部单位
	"llama3-groq-8b-8192-tool-use-preview":  0.19 / 1000000 * USD, // $0.19/1M tokens → 0.095 内部单位
	"mixtral-8x7b-32768":                    0.24 / 1000000 * USD, // $0.24/1M tokens → 0.12 内部单位

	// https://platform.lingyiwanwu.com/docs#-计费单元
	// 零一万物模型定价以 ￥/1M tokens 为单位，使用 X / 1000 * RMB 转换
	"yi-34b-chat-0205": 2.5 / 1000 * RMB,  // ￥2.5/1M tokens ≈ 0.179 内部单位
	"yi-34b-chat-200k": 12.0 / 1000 * RMB, // ￥12/1M tokens ≈ 0.857 内部单位
	"yi-vl-plus":       6.0 / 1000 * RMB,  // ￥6/1M tokens ≈ 0.429 内部单位
	// https://platform.stepfun.com/docs/pricing/details
	// 阶跃星辰模型定价以 ￥/1M tokens 为单位，使用 X / 1000 * RMB 转换
	"step-1-8k":    0.005 / 1000 * RMB, // ￥0.005/1M tokens
	"step-1-32k":   0.015 / 1000 * RMB, // ￥0.015/1M tokens
	"step-1-128k":  0.040 / 1000 * RMB, // ￥0.04/1M tokens
	"step-1-256k":  0.095 / 1000 * RMB, // ￥0.095/1M tokens
	"step-1-flash": 0.001 / 1000 * RMB, // ￥0.001/1M tokens
	"step-2-16k":   0.038 / 1000 * RMB, // ￥0.038/1M tokens
	"step-1v-8k":   0.005 / 1000 * RMB, // ￥0.005/1M tokens（视觉模型）
	"step-1v-32k":  0.015 / 1000 * RMB, // ￥0.015/1M tokens（视觉模型）
	// aws llama3 https://aws.amazon.com/cn/bedrock/pricing/
	// AWS Bedrock 模型，名称后缀 (33) 为渠道类型编号。使用原始单价 / 基准价 $0.002 计算倍数
	"llama3-8b-8192(33)":  0.0003 / 0.002,  // $0.0003 / 1K tokens ÷ $0.002 = 0.15 内部单位
	"llama3-70b-8192(33)": 0.00265 / 0.002, // $0.00265 / 1K tokens ÷ $0.002 = 1.325 内部单位
	// https://cohere.com/pricing
	// Cohere 旧版模型直接使用内部单位值，新版模型以 $/1M tokens 为单位
	"command":               0.5, // 0.5 内部单位（约 $0.001 / 1K tokens）
	"command-nightly":       0.5,
	"command-light":         0.5,
	"command-light-nightly": 0.5,
	"command-r":             0.5 / 1000 * USD, // $0.5/1M tokens → 0.25 内部单位
	"command-r-plus":        3.0 / 1000 * USD, // $3/1M tokens → 1.5 内部单位
	// https://platform.deepseek.com/api-docs/pricing/
	// DeepSeek 官方 API 定价以 $/1M tokens 为单位，使用 X * MILLI_USD 转换
	"deepseek-chat":     0.14 * MILLI_USD, // $0.14/1M tokens → 0.07 内部单位
	"deepseek-reasoner": 0.55 * MILLI_USD, // $0.55/1M tokens → 0.275 内部单位
	// https://www.deepl.com/pro?cta=header-prices
	// DeepL 翻译模型定价以 $/1M tokens 为单位
	"deepl-zh": 25.0 / 1000 * USD, // $25/1M tokens → 12.5 内部单位
	"deepl-en": 25.0 / 1000 * USD, // $25/1M tokens → 12.5 内部单位
	"deepl-ja": 25.0 / 1000 * USD, // $25/1M tokens → 12.5 内部单位
	// https://console.x.ai/
	// xAI Grok 定价以 $/1M tokens 为单位
	"grok-beta": 5.0 / 1000 * USD, // $5/1M tokens → 2.5 内部单位
	// replicate charges based on the number of generated images
	// https://replicate.com/pricing
	// Replicate 图像模型按次计费，使用 X * USD 转换（X 是美元单价/次）
	"black-forest-labs/flux-1.1-pro":                0.04 * USD, // $0.04/次 → 20 内部单位
	"black-forest-labs/flux-1.1-pro-ultra":          0.06 * USD,
	"black-forest-labs/flux-canny-dev":              0.025 * USD,
	"black-forest-labs/flux-canny-pro":              0.05 * USD,
	"black-forest-labs/flux-depth-dev":              0.025 * USD,
	"black-forest-labs/flux-depth-pro":              0.05 * USD,
	"black-forest-labs/flux-dev":                    0.025 * USD,
	"black-forest-labs/flux-dev-lora":               0.032 * USD,
	"black-forest-labs/flux-fill-dev":               0.04 * USD,
	"black-forest-labs/flux-fill-pro":               0.05 * USD,
	"black-forest-labs/flux-pro":                    0.055 * USD,
	"black-forest-labs/flux-redux-dev":              0.025 * USD,
	"black-forest-labs/flux-redux-schnell":          0.003 * USD,
	"black-forest-labs/flux-schnell":                0.003 * USD,
	"black-forest-labs/flux-schnell-lora":           0.02 * USD,
	"ideogram-ai/ideogram-v2":                       0.08 * USD,
	"ideogram-ai/ideogram-v2-turbo":                 0.05 * USD,
	"recraft-ai/recraft-v3":                         0.04 * USD,
	"recraft-ai/recraft-v3-svg":                     0.08 * USD,
	"stability-ai/stable-diffusion-3":               0.035 * USD,
	"stability-ai/stable-diffusion-3.5-large":       0.065 * USD,
	"stability-ai/stable-diffusion-3.5-large-turbo": 0.04 * USD,
	"stability-ai/stable-diffusion-3.5-medium":      0.035 * USD,
	// Replicate 对话模型按次计费，使用 X * USD 转换（X 是美元单价/次）
	"ibm-granite/granite-20b-code-instruct-8k":  0.100 * USD, // $0.10/次 → 50 内部单位
	"ibm-granite/granite-3.0-2b-instruct":       0.030 * USD,
	"ibm-granite/granite-3.0-8b-instruct":       0.050 * USD,
	"ibm-granite/granite-8b-code-instruct-128k": 0.050 * USD,
	"meta/llama-2-13b":                          0.100 * USD,
	"meta/llama-2-13b-chat":                     0.100 * USD,
	"meta/llama-2-70b":                          0.650 * USD,
	"meta/llama-2-70b-chat":                     0.650 * USD,
	"meta/llama-2-7b":                           0.050 * USD,
	"meta/llama-2-7b-chat":                      0.050 * USD,
	"meta/meta-llama-3.1-405b-instruct":         9.500 * USD,
	"meta/meta-llama-3-70b":                     0.650 * USD,
	"meta/meta-llama-3-70b-instruct":            0.650 * USD,
	"meta/meta-llama-3-8b":                      0.050 * USD,
	"meta/meta-llama-3-8b-instruct":             0.050 * USD,
	"mistralai/mistral-7b-instruct-v0.2":        0.050 * USD,
	"mistralai/mistral-7b-v0.1":                 0.050 * USD,
	"mistralai/mixtral-8x7b-instruct-v0.1":      0.300 * USD,
	//https://openrouter.ai/models
	// OpenRouter 模型定价统一为内部单位值，来自 OpenRouter 的 $/1M tokens 报价换算
	// 换算方式：$X/1M tokens → X * 0.5 内部单位（即 X * MILLI_USD）
	"01-ai/yi-large":                                  1.5, // 约 $3/1M tokens → 1.5 内部单位
	"aetherwiing/mn-starcannon-12b":                   0.6,
	"ai21/jamba-1-5-large":                            4.0,
	"ai21/jamba-1-5-mini":                             0.2,
	"ai21/jamba-instruct":                             0.35,
	"aion-labs/aion-1.0":                              6.0,
	"aion-labs/aion-1.0-mini":                         1.2,
	"aion-labs/aion-rp-llama-3.1-8b":                  0.1,
	"allenai/llama-3.1-tulu-3-405b":                   5.0,
	"alpindale/goliath-120b":                          4.6875,
	"alpindale/magnum-72b":                            1.125,
	"amazon/nova-lite-v1":                             0.12,
	"amazon/nova-micro-v1":                            0.07,
	"amazon/nova-pro-v1":                              1.6,
	"anthracite-org/magnum-v2-72b":                    1.5,
	"anthracite-org/magnum-v4-72b":                    1.125,
	"anthropic/claude-2":                              12.0,
	"anthropic/claude-2.0":                            12.0,
	"anthropic/claude-2.0:beta":                       12.0,
	"anthropic/claude-2.1":                            12.0,
	"anthropic/claude-2.1:beta":                       12.0,
	"anthropic/claude-2:beta":                         12.0,
	"anthropic/claude-3-haiku":                        0.625,
	"anthropic/claude-3-haiku:beta":                   0.625,
	"anthropic/claude-3-opus":                         37.5,
	"anthropic/claude-3-opus:beta":                    37.5,
	"anthropic/claude-3-sonnet":                       7.5,
	"anthropic/claude-3-sonnet:beta":                  7.5,
	"anthropic/claude-3.5-haiku":                      2.0,
	"anthropic/claude-3.5-haiku-20241022":             2.0,
	"anthropic/claude-3.5-haiku-20241022:beta":        2.0,
	"anthropic/claude-3.5-haiku:beta":                 2.0,
	"anthropic/claude-3.5-sonnet":                     7.5,
	"anthropic/claude-3.5-sonnet-20240620":            7.5,
	"anthropic/claude-3.5-sonnet-20240620:beta":       7.5,
	"anthropic/claude-3.5-sonnet:beta":                7.5,
	"cognitivecomputations/dolphin-mixtral-8x22b":     0.45,
	"cognitivecomputations/dolphin-mixtral-8x7b":      0.25,
	"cohere/command":                                  0.95,
	"cohere/command-r":                                0.7125,
	"cohere/command-r-03-2024":                        0.7125,
	"cohere/command-r-08-2024":                        0.285,
	"cohere/command-r-plus":                           7.125,
	"cohere/command-r-plus-04-2024":                   7.125,
	"cohere/command-r-plus-08-2024":                   4.75,
	"cohere/command-r7b-12-2024":                      0.075,
	"databricks/dbrx-instruct":                        0.6,
	"deepseek/deepseek-chat":                          0.445,
	"deepseek/deepseek-chat-v2.5":                     1.0,
	"deepseek/deepseek-chat:free":                     0.0,
	"deepseek/deepseek-r1":                            1.2,
	"deepseek/deepseek-r1-distill-llama-70b":          0.345,
	"deepseek/deepseek-r1-distill-llama-70b:free":     0.0,
	"deepseek/deepseek-r1-distill-llama-8b":           0.02,
	"deepseek/deepseek-r1-distill-qwen-1.5b":          0.09,
	"deepseek/deepseek-r1-distill-qwen-14b":           0.075,
	"deepseek/deepseek-r1-distill-qwen-32b":           0.09,
	"deepseek/deepseek-r1:free":                       0.0,
	"eva-unit-01/eva-llama-3.33-70b":                  3.0,
	"eva-unit-01/eva-qwen-2.5-32b":                    1.7,
	"eva-unit-01/eva-qwen-2.5-72b":                    3.0,
	"google/gemini-2.0-flash-001":                     0.2,
	"google/gemini-2.0-flash-exp:free":                0.0,
	"google/gemini-2.0-flash-lite-preview-02-05:free": 0.0,
	"google/gemini-2.0-flash-thinking-exp-1219:free":  0.0,
	"google/gemini-2.0-flash-thinking-exp:free":       0.0,
	"google/gemini-2.0-pro-exp-02-05:free":            0.0,
	"google/gemini-exp-1206:free":                     0.0,
	"google/gemini-flash-1.5":                         0.15,
	"google/gemini-flash-1.5-8b":                      0.075,
	"google/gemini-flash-1.5-8b-exp":                  0.0,
	"google/gemini-pro":                               0.75,
	"google/gemini-pro-1.5":                           2.5,
	"google/gemini-pro-vision":                        0.75,
	"google/gemma-2-27b-it":                           0.135,
	"google/gemma-2-9b-it":                            0.03,
	"google/gemma-2-9b-it:free":                       0.0,
	"google/gemma-7b-it":                              0.075,
	"google/learnlm-1.5-pro-experimental:free":        0.0,
	"google/palm-2-chat-bison":                        1.0,
	"google/palm-2-chat-bison-32k":                    1.0,
	"google/palm-2-codechat-bison":                    1.0,
	"google/palm-2-codechat-bison-32k":                1.0,
	"gryphe/mythomax-l2-13b":                          0.0325,
	"gryphe/mythomax-l2-13b:free":                     0.0,
	"huggingfaceh4/zephyr-7b-beta:free":               0.0,
	"infermatic/mn-inferor-12b":                       0.6,
	"inflection/inflection-3-pi":                      5.0,
	"inflection/inflection-3-productivity":            5.0,
	"jondurbin/airoboros-l2-70b":                      0.25,
	"liquid/lfm-3b":                                   0.01,
	"liquid/lfm-40b":                                  0.075,
	"liquid/lfm-7b":                                   0.005,
	"mancer/weaver":                                   1.125,
	"meta-llama/llama-2-13b-chat":                     0.11,
	"meta-llama/llama-2-70b-chat":                     0.45,
	"meta-llama/llama-3-70b-instruct":                 0.2,
	"meta-llama/llama-3-8b-instruct":                  0.03,
	"meta-llama/llama-3-8b-instruct:free":             0.0,
	"meta-llama/llama-3.1-405b":                       1.0,
	"meta-llama/llama-3.1-405b-instruct":              0.4,
	"meta-llama/llama-3.1-70b-instruct":               0.15,
	"meta-llama/llama-3.1-8b-instruct":                0.025,
	"meta-llama/llama-3.2-11b-vision-instruct":        0.0275,
	"meta-llama/llama-3.2-11b-vision-instruct:free":   0.0,
	"meta-llama/llama-3.2-1b-instruct":                0.005,
	"meta-llama/llama-3.2-3b-instruct":                0.0125,
	"meta-llama/llama-3.2-90b-vision-instruct":        0.8,
	"meta-llama/llama-3.3-70b-instruct":               0.15,
	"meta-llama/llama-3.3-70b-instruct:free":          0.0,
	"meta-llama/llama-guard-2-8b":                     0.1,
	"microsoft/phi-3-medium-128k-instruct":            0.5,
	"microsoft/phi-3-medium-128k-instruct:free":       0.0,
	"microsoft/phi-3-mini-128k-instruct":              0.05,
	"microsoft/phi-3-mini-128k-instruct:free":         0.0,
	"microsoft/phi-3.5-mini-128k-instruct":            0.05,
	"microsoft/phi-4":                                 0.07,
	"microsoft/wizardlm-2-7b":                         0.035,
	"microsoft/wizardlm-2-8x22b":                      0.25,
	"minimax/minimax-01":                              0.55,
	"mistralai/codestral-2501":                        0.45,
	"mistralai/codestral-mamba":                       0.125,
	"mistralai/ministral-3b":                          0.02,
	"mistralai/ministral-8b":                          0.05,
	"mistralai/mistral-7b-instruct":                   0.0275,
	"mistralai/mistral-7b-instruct-v0.1":              0.1,
	"mistralai/mistral-7b-instruct-v0.3":              0.0275,
	"mistralai/mistral-7b-instruct:free":              0.0,
	"mistralai/mistral-large":                         3.0,
	"mistralai/mistral-large-2407":                    3.0,
	"mistralai/mistral-large-2411":                    3.0,
	"mistralai/mistral-medium":                        4.05,
	"mistralai/mistral-nemo":                          0.04,
	"mistralai/mistral-nemo:free":                     0.0,
	"mistralai/mistral-small":                         0.3,
	"mistralai/mistral-small-24b-instruct-2501":       0.07,
	"mistralai/mistral-small-24b-instruct-2501:free":  0.0,
	"mistralai/mistral-tiny":                          0.125,
	"mistralai/mixtral-8x22b-instruct":                0.45,
	"mistralai/mixtral-8x7b":                          0.3,
	"mistralai/mixtral-8x7b-instruct":                 0.12,
	"mistralai/pixtral-12b":                           0.05,
	"mistralai/pixtral-large-2411":                    3.0,
	"neversleep/llama-3-lumimaid-70b":                 2.25,
	"neversleep/llama-3-lumimaid-8b":                  0.5625,
	"neversleep/llama-3-lumimaid-8b:extended":         0.5625,
	"neversleep/llama-3.1-lumimaid-70b":               2.25,
	"neversleep/llama-3.1-lumimaid-8b":                0.5625,
	"neversleep/noromaid-20b":                         1.125,
	"nothingiisreal/mn-celeste-12b":                   0.6,
	"nousresearch/hermes-2-pro-llama-3-8b":            0.02,
	"nousresearch/hermes-3-llama-3.1-405b":            0.4,
	"nousresearch/hermes-3-llama-3.1-70b":             0.15,
	"nousresearch/nous-hermes-2-mixtral-8x7b-dpo":     0.3,
	"nousresearch/nous-hermes-llama2-13b":             0.085,
	"nvidia/llama-3.1-nemotron-70b-instruct":          0.15,
	"nvidia/llama-3.1-nemotron-70b-instruct:free":     0.0,
	"openai/chatgpt-4o-latest":                        7.5,
	"openai/gpt-3.5-turbo":                            0.75,
	"openai/gpt-3.5-turbo-0125":                       0.75,
	"openai/gpt-3.5-turbo-0613":                       1.0,
	"openai/gpt-3.5-turbo-1106":                       1.0,
	"openai/gpt-3.5-turbo-16k":                        2.0,
	"openai/gpt-3.5-turbo-instruct":                   1.0,
	"openai/gpt-4":                                    30.0,
	"openai/gpt-4-0314":                               30.0,
	"openai/gpt-4-1106-preview":                       15.0,
	"openai/gpt-4-32k":                                60.0,
	"openai/gpt-4-32k-0314":                           60.0,
	"openai/gpt-4-turbo":                              15.0,
	"openai/gpt-4-turbo-preview":                      15.0,
	"openai/gpt-4o":                                   5.0,
	"openai/gpt-4o-2024-05-13":                        7.5,
	"openai/gpt-4o-2024-08-06":                        5.0,
	"openai/gpt-4o-2024-11-20":                        5.0,
	"openai/gpt-4o-mini":                              0.3,
	"openai/gpt-4o-mini-2024-07-18":                   0.3,
	"openai/gpt-4o:extended":                          9.0,
	"openai/o1":                                       30.0,
	"openai/o1-mini":                                  2.2,
	"openai/o1-mini-2024-09-12":                       2.2,
	"openai/o1-preview":                               30.0,
	"openai/o1-preview-2024-09-12":                    30.0,
	"openai/o3-mini":                                  2.2,
	"openai/o3-mini-high":                             2.2,
	"openchat/openchat-7b":                            0.0275,
	"openchat/openchat-7b:free":                       0.0,
	"openrouter/auto":                                 -500000.0,
	"perplexity/llama-3.1-sonar-huge-128k-online":     2.5,
	"perplexity/llama-3.1-sonar-large-128k-chat":      0.5,
	"perplexity/llama-3.1-sonar-large-128k-online":    0.5,
	"perplexity/llama-3.1-sonar-small-128k-chat":      0.1,
	"perplexity/llama-3.1-sonar-small-128k-online":    0.1,
	"perplexity/sonar":                                0.5,
	"perplexity/sonar-reasoning":                      2.5,
	"pygmalionai/mythalion-13b":                       0.6,
	"qwen/qvq-72b-preview":                            0.25,
	"qwen/qwen-2-72b-instruct":                        0.45,
	"qwen/qwen-2-7b-instruct":                         0.027,
	"qwen/qwen-2-7b-instruct:free":                    0.0,
	"qwen/qwen-2-vl-72b-instruct":                     0.2,
	"qwen/qwen-2-vl-7b-instruct":                      0.05,
	"qwen/qwen-2.5-72b-instruct":                      0.2,
	"qwen/qwen-2.5-7b-instruct":                       0.025,
	"qwen/qwen-2.5-coder-32b-instruct":                0.08,
	"qwen/qwen-max":                                   3.2,
	"qwen/qwen-plus":                                  0.6,
	"qwen/qwen-turbo":                                 0.1,
	"qwen/qwen-vl-plus:free":                          0.0,
	"qwen/qwen2.5-vl-72b-instruct:free":               0.0,
	"qwen/qwq-32b-preview":                            0.09,
	"raifle/sorcererlm-8x22b":                         2.25,
	"sao10k/fimbulvetr-11b-v2":                        0.6,
	"sao10k/l3-euryale-70b":                           0.4,
	"sao10k/l3-lunaris-8b":                            0.03,
	"sao10k/l3.1-70b-hanami-x1":                       1.5,
	"sao10k/l3.1-euryale-70b":                         0.4,
	"sao10k/l3.3-euryale-70b":                         0.4,
	"sophosympatheia/midnight-rose-70b":               0.4,
	"sophosympatheia/rogue-rose-103b-v0.2:free":       0.0,
	"teknium/openhermes-2.5-mistral-7b":               0.085,
	"thedrummer/rocinante-12b":                        0.25,
	"thedrummer/unslopnemo-12b":                       0.25,
	"undi95/remm-slerp-l2-13b":                        0.6,
	"undi95/toppy-m-7b":                               0.035,
	"undi95/toppy-m-7b:free":                          0.0,
	"x-ai/grok-2-1212":                                5.0,
	"x-ai/grok-2-vision-1212":                         5.0,
	"x-ai/grok-beta":                                  7.5,
	"x-ai/grok-vision-beta":                           7.5,
	"xwin-lm/xwin-lm-70b":                             1.875,
}

// CompletionRatio 存储各模型的输出(completion) token 与输入(input) token 的价格比率。
//
// 【含义说明】
//   - 值 = 输出 token 单价 / 输入 token 单价
//   - 例如："deepseek-chat" 值为 0.28/0.14 = 2，表示输出 token 价格是输入的 2 倍
//   - 如果模型未在此表中列出，则 GetCompletionRatio() 会根据模型名称前缀返回默认比率
//
// 【最终计费公式】
//   - 输入 token 费用 = ModelRatio × $0.002 / 1K tokens × token 数量
//   - 输出 token 费用 = ModelRatio × CompletionRatio × $0.002 / 1K tokens × token 数量
//
// 【注意】仅在输出/输入价格不同时才需要在此配置，否则默认比率为 1
var CompletionRatio = map[string]float64{
	// aws llama3 — 输出/输入价格比
	"llama3-8b-8192(33)":  0.0006 / 0.0003,  // 输出 $0.0006 / 输入 $0.0003 = 2
	"llama3-70b-8192(33)": 0.0035 / 0.00265, // 输出 $0.0035 / 输入 $0.00265 ≈ 1.32
	// whisper 仅计算输入 token（语音转文字），输出 token 不计费
	"whisper-1": 0, // only count input tokens
	// deepseek 输出/输入价格比
	"deepseek-chat":     0.28 / 0.14, // 输出 $0.28/1M / 输入 $0.14/1M = 2
	"deepseek-reasoner": 2.19 / 0.55, // 输出 $2.19/1M / 输入 $0.55/1M ≈ 3.98
}

// DefaultModelRatio 是 ModelRatio 的初始副本，运行时不会被修改，
// 用于在用户自定义 ModelRatio 中找不到模型时回退到默认值
var DefaultModelRatio map[string]float64

// DefaultCompletionRatio 是 CompletionRatio 的初始副本，运行时不会被修改，
// 用于在用户自定义 CompletionRatio 中找不到模型时回退到默认值
var DefaultCompletionRatio map[string]float64

// init 在包初始化时保存 ModelRatio 和 CompletionRatio 的默认值，
// 以便运行时更新后仍可回退查询
func init() {
	DefaultModelRatio = make(map[string]float64)
	for k, v := range ModelRatio {
		DefaultModelRatio[k] = v
	}
	DefaultCompletionRatio = make(map[string]float64)
	for k, v := range CompletionRatio {
		DefaultCompletionRatio[k] = v
	}
}

// AddNewMissingRatio 将默认 ModelRatio 中缺失的模型比率补充到旧的 JSON 字符串中。
// oldRatio 是用户自定义的 ModelRatio JSON 字符串，函数会合并默认值中新增的模型。
func AddNewMissingRatio(oldRatio string) string {
	newRatio := make(map[string]float64)
	err := json.Unmarshal([]byte(oldRatio), &newRatio)
	if err != nil {
		logger.SysError("error unmarshalling old ratio: " + err.Error())
		return oldRatio
	}
	for k, v := range DefaultModelRatio {
		if _, ok := newRatio[k]; !ok {
			newRatio[k] = v
		}
	}
	jsonBytes, err := json.Marshal(newRatio)
	if err != nil {
		logger.SysError("error marshalling new ratio: " + err.Error())
		return oldRatio
	}
	return string(jsonBytes)
}

// ModelRatio2JSONString 将当前 ModelRatio 序列化为 JSON 字符串
func ModelRatio2JSONString() string {
	jsonBytes, err := json.Marshal(ModelRatio)
	if err != nil {
		logger.SysError("error marshalling model ratio: " + err.Error())
	}
	return string(jsonBytes)
}

// UpdateModelRatioByJSONString 用 JSON 字符串覆盖更新 ModelRatio（运行时动态配置）
func UpdateModelRatioByJSONString(jsonStr string) error {
	modelRatioLock.Lock()
	defer modelRatioLock.Unlock()
	ModelRatio = make(map[string]float64)
	return json.Unmarshal([]byte(jsonStr), &ModelRatio)
}

// GetModelRatio 获取指定模型的输入 token 计费比率（内部单位）。
// 查找优先级：
//  1. 用户自定义 ModelRatio 中按 "模型名(渠道类型)" 查找
//  2. 默认 DefaultModelRatio 中按 "模型名(渠道类型)" 查找
//  3. 用户自定义 ModelRatio 中按模型名查找
//  4. 默认 DefaultModelRatio 中按模型名查找
//  5. 未找到则返回 30（即 $0.06 / 1K tokens，按高价计费防止漏计）
func GetModelRatio(name string, channelType int) float64 {
	modelRatioLock.RLock()
	defer modelRatioLock.RUnlock()

	// 联网搜索后缀处理：qwen-xxx-internet 和 command-xxx-internet 使用原模型价格
	if strings.HasPrefix(name, "qwen-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	if strings.HasPrefix(name, "command-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}

	// 先按 "模型名(渠道类型)" 查找，支持同一模型在不同渠道使用不同定价
	model := fmt.Sprintf("%s(%d)", name, channelType)
	if ratio, ok := ModelRatio[model]; ok {
		return ratio
	}
	if ratio, ok := DefaultModelRatio[model]; ok {
		return ratio
	}
	if ratio, ok := ModelRatio[name]; ok {
		return ratio
	}
	if ratio, ok := DefaultModelRatio[name]; ok {
		return ratio
	}
	logger.SysError("model ratio not found: " + name)
	return 30 // 未找到模型时返回高价默认值 30（即 $0.06 / 1K tokens），防止漏计
}

// CompletionRatio2JSONString 将当前 CompletionRatio 序列化为 JSON 字符串
func CompletionRatio2JSONString() string {
	jsonBytes, err := json.Marshal(CompletionRatio)
	if err != nil {
		logger.SysError("error marshalling completion ratio: " + err.Error())
	}
	return string(jsonBytes)
}

// UpdateCompletionRatioByJSONString 用 JSON 字符串覆盖更新 CompletionRatio（运行时动态配置）
func UpdateCompletionRatioByJSONString(jsonStr string) error {
	CompletionRatio = make(map[string]float64)
	return json.Unmarshal([]byte(jsonStr), &CompletionRatio)
}

// GetCompletionRatio 获取指定模型的输出/输入 token 价格比率。
// 查找优先级：
//  1. 用户自定义 CompletionRatio 中按 "模型名(渠道类型)" 查找
//  2. 默认 DefaultCompletionRatio 中按 "模型名(渠道类型)" 查找
//  3. 用户自定义 CompletionRatio 中按模型名查找
//  4. 默认 DefaultCompletionRatio 中按模型名查找
//  5. 根据模型名前缀返回默认比率（如 gpt-4o 系列=4，claude-3 系列=5 等）
//  6. 最终默认返回 1（输出与输入同价）
func GetCompletionRatio(name string, channelType int) float64 {
	// 联网搜索后缀处理
	if strings.HasPrefix(name, "qwen-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	// 先按 "模型名(渠道类型)" 查找，支持同一模型在不同渠道使用不同比率
	model := fmt.Sprintf("%s(%d)", name, channelType)
	if ratio, ok := CompletionRatio[model]; ok {
		return ratio
	}
	if ratio, ok := DefaultCompletionRatio[model]; ok {
		return ratio
	}
	if ratio, ok := CompletionRatio[name]; ok {
		return ratio
	}
	if ratio, ok := DefaultCompletionRatio[name]; ok {
		return ratio
	}
	// 以下为根据模型名前缀推断的默认 CompletionRatio
	if strings.HasPrefix(name, "gpt-3.5") {
		if name == "gpt-3.5-turbo" || strings.HasSuffix(name, "0125") {
			// https://openai.com/blog/new-embedding-models-and-api-updates
			// Updated GPT-3.5 Turbo model and lower pricing
			// 输入 $0.0005/1K，输出 $0.0015/1K → 比率 = 3
			return 3
		}
		if strings.HasSuffix(name, "1106") {
			return 2
		}
		return 4.0 / 3.0
	}
	if strings.HasPrefix(name, "gpt-4") {
		if strings.HasPrefix(name, "gpt-4o") {
			if name == "gpt-4o-2024-05-13" {
				return 3 // 输入 $0.005/1K，输出 $0.015/1K → 比率 = 3
			}
			return 4 // gpt-4o 其他版本：输入 $0.0025/1K，输出 $0.01/1K → 比率 = 4
		}
		if strings.HasPrefix(name, "gpt-4-turbo") ||
			strings.HasSuffix(name, "preview") {
			return 3 // gpt-4-turbo：输入 $0.01/1K，输出 $0.03/1K → 比率 = 3
		}
		return 2 // 原始 gpt-4：输入 $0.03/1K，输出 $0.06/1K → 比率 = 2
	}
	// o1/o1-preview/o1-mini 系列：输出价格是输入的 4 倍
	// 例如 o1：输入 $15/1M，输出 $60/1M → 比率 = 4
	if strings.HasPrefix(name, "o1") {
		return 4
	}
	if name == "chatgpt-4o-latest" {
		return 3 // 输入 $0.005/1K，输出 $0.015/1K → 比率 = 3
	}
	// Claude 3 系列：输出价格是输入的 5 倍
	// 例如 claude-3-sonnet：输入 $3/1M，输出 $15/1M → 比率 = 5
	if strings.HasPrefix(name, "claude-3") {
		return 5
	}
	// Claude 旧版系列：输出价格是输入的 3 倍
	if strings.HasPrefix(name, "claude-") {
		return 3
	}
	if strings.HasPrefix(name, "mistral-") {
		return 3 // Mistral 系列输出/输入比率 ≈ 3
	}
	if strings.HasPrefix(name, "gemini-") {
		return 3 // Gemini 系列输出/输入比率 ≈ 3
	}
	if strings.HasPrefix(name, "deepseek-") {
		return 2 // DeepSeek 系列输出/输入比率 = 2（如 deepseek-chat）
	}

	// 以下为特定模型的 CompletionRatio，按输出/输入单价计算
	switch name {
	case "llama2-70b-4096":
		return 0.8 / 0.64 // Groq: 输出 $0.8/1M / 输入 $0.64/1M ≈ 1.25
	case "llama3-8b-8192":
		return 2 // Groq: 输出 $0.1/1M / 输入 $0.05/1M = 2
	case "llama3-70b-8192":
		return 0.79 / 0.59 // Groq: 输出 $0.79/1M / 输入 $0.59/1M ≈ 1.34
	case "command", "command-light", "command-nightly", "command-light-nightly":
		return 2 // Cohere 旧版：输出/输入比率 = 2
	case "command-r":
		return 3 // 输入 $0.5/1M，输出 $1.5/1M → 比率 = 3
	case "command-r-plus":
		return 5 // 输入 $3/1M，输出 $15/1M → 比率 = 5
	case "grok-beta":
		return 3 // xAI Grok：输出/输入比率 = 3
	// Replicate Models
	// https://replicate.com/pricing
	case "ibm-granite/granite-20b-code-instruct-8k":
		return 5
	case "ibm-granite/granite-3.0-2b-instruct":
		return 8.333333333333334
	case "ibm-granite/granite-3.0-8b-instruct",
		"ibm-granite/granite-8b-code-instruct-128k":
		return 5
	case "meta/llama-2-13b",
		"meta/llama-2-13b-chat",
		"meta/llama-2-7b",
		"meta/llama-2-7b-chat",
		"meta/meta-llama-3-8b",
		"meta/meta-llama-3-8b-instruct":
		return 5
	case "meta/llama-2-70b",
		"meta/llama-2-70b-chat",
		"meta/meta-llama-3-70b",
		"meta/meta-llama-3-70b-instruct":
		return 2.750 / 0.650 // ≈4.230769
	case "meta/meta-llama-3.1-405b-instruct":
		return 1
	case "mistralai/mistral-7b-instruct-v0.2",
		"mistralai/mistral-7b-v0.1":
		return 5
	case "mistralai/mixtral-8x7b-instruct-v0.1":
		return 1.000 / 0.300 // ≈3.333333
	}

	return 1 // 默认：输出与输入同价
}
