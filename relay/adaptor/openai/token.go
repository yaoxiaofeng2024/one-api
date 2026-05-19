// ============================================================
// token.go —— Token 计数模块（计费的核心依赖）
// ============================================================
//
// 【这个文件的作用】
// 在 One-API 的计费流程中，必须先知道用户消耗了多少 token，才能扣除对应配额。
// 但上游模型厂商只在"响应完成后"才返回 usage（实际消耗的 token 数），
// 而我们需要在"请求发出前"就预估 token 数来做预扣费。
//
// 所以这个文件的作用是：在请求发出前，本地计算 prompt 的 token 数，
// 用于预扣配额；请求完成后，再拿上游返回的实际 usage 做结算。
//
// 【核心问题：token 怎么算？】
// 不同模型使用不同的分词器（tokenizer），同一个文本在不同模型下 token 数不同。
// 例如 "hello world" 在 GPT-4 的分词器下可能是 2 个 token，
// 在 Claude 的分词器下可能是 3 个 token。
//
// 本文件使用 OpenAI 开源的 tiktoken 库来做分词计算，
// 因为大多数模型（包括 DeepSeek、豆包等）都兼容 OpenAI 的分词方式。
// 对于非 OpenAI 系的模型，会回退到 GPT-3.5 的分词器做近似估算。
//
// ============================================================

package openai

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/pkoukk/tiktoken-go"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/image"
	"github.com/songquanpeng/one-api/common/logger"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/model"
)

// ============================================================
// 第一部分：分词器（Tokenizer）的缓存管理
// ============================================================
//
// 【为什么需要缓存？】
// tiktoken.EncodingForModel() 每次调用都要加载分词词表（BPE 编码表），
// 这是一个较重的操作。所以我们在启动时预加载常用的分词器，缓存起来复用。
//
// 【分词器映射策略】
// GPT 系列模型实际上只用了 3 种分词器：
//   - cl100k_base：GPT-3.5、GPT-4 使用
//   - o200k_base：GPT-4o 使用（更新的词表）
// 而其他模型（DeepSeek、Claude 等）没有自己的 tiktoken 分词器，
// 会回退到 defaultTokenEncoder（GPT-3.5 的分词器）做近似估算。

// tokenEncoderMap 缓存每个模型对应的分词器实例
// 启动时预填充，运行时按需懒加载（见 getTokenEncoder）
var tokenEncoderMap = map[string]*tiktoken.Tiktoken{}

// defaultTokenEncoder 默认分词器，未知模型都用它估算
var defaultTokenEncoder *tiktoken.Tiktoken

// InitTokenEncoders 启动时调用，预加载 GPT 系列的分词器
// 这是因为 GPT 模型的 token 计数必须精确（直接影响计费）
func InitTokenEncoders() {
	logger.SysLog("initializing token encoders")

	// 预加载 GPT-3.5 的分词器（cl100k_base），同时作为默认分词器
	gpt35TokenEncoder, err := tiktoken.EncodingForModel("gpt-3.5-turbo")
	if err != nil {
		logger.FatalLog(fmt.Sprintf("failed to get gpt-3.5-turbo token encoder: %s, "+
			"if you are using in offline environment, please set TIKTOKEN_CACHE_DIR to use exsited files, check this link for more information: https://stackoverflow.com/questions/76106366/how-to-use-tiktoken-in-offline-mode-computer ", err.Error()))
	}
	defaultTokenEncoder = gpt35TokenEncoder

	// 预加载 GPT-4o 的分词器（o200k_base，更新的词表）
	gpt4oTokenEncoder, err := tiktoken.EncodingForModel("gpt-4o")
	if err != nil {
		logger.FatalLog(fmt.Sprintf("failed to get gpt-4o token encoder: %s", err.Error()))
	}

	// 预加载 GPT-4 的分词器（cl100k_base，跟 GPT-3.5 一样）
	gpt4TokenEncoder, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		logger.FatalLog(fmt.Sprintf("failed to get gpt-4 token encoder: %s", err.Error()))
	}

	// 遍历所有已知模型（来自计费倍率表），按前缀分配分词器
	// 非 GPT 系列的模型设为 nil，等到实际使用时再懒加载
	for model := range billingratio.ModelRatio {
		if strings.HasPrefix(model, "gpt-3.5") {
			tokenEncoderMap[model] = gpt35TokenEncoder
		} else if strings.HasPrefix(model, "gpt-4o") {
			tokenEncoderMap[model] = gpt4oTokenEncoder
		} else if strings.HasPrefix(model, "gpt-4") {
			tokenEncoderMap[model] = gpt4TokenEncoder
		} else {
			tokenEncoderMap[model] = nil // 标记为未加载，使用时再按需初始化
		}
	}
	logger.SysLog("token encoders initialized")
}

// getTokenEncoder 获取指定模型的分词器，带懒加载机制
//
// 查找顺序：
//  1. 已缓存且有值 → 直接返回
//  2. 已缓存但为 nil（非 GPT 系列模型） → 尝试用 tiktoken 加载，失败则用默认分词器
//  3. 不在缓存中（新模型） → 返回默认分词器（GPT-3.5 的 cl100k_base）
func getTokenEncoder(model string) *tiktoken.Tiktoken {
	tokenEncoder, ok := tokenEncoderMap[model]
	if ok && tokenEncoder != nil {
		// 情况1：已缓存且有值，直接返回
		return tokenEncoder
	}
	if ok {
		// 情况2：已缓存但为 nil，尝试按模型名加载分词器
		// 这处理了 InitTokenEncoders 中标记为 nil 的非 GPT 模型
		tokenEncoder, err := tiktoken.EncodingForModel(model)
		if err != nil {
			// 该模型在 tiktoken 中也没有对应的分词器，回退到默认
			logger.SysError(fmt.Sprintf("failed to get token encoder for model %s: %s, using encoder for gpt-3.5-turbo", model, err.Error()))
			tokenEncoder = defaultTokenEncoder
		}
		tokenEncoderMap[model] = tokenEncoder // 缓存起来，下次直接用
		return tokenEncoder
	}
	// 情况3：模型不在缓存中（没在 ModelRatio 表里注册），用默认分词器
	return defaultTokenEncoder
}

// ============================================================
// 第二部分：Token 计数
// ============================================================

// getTokenNum 计算一段文本的 token 数
//
// 有两种模式：
//   - 精确模式（默认）：用分词器实际分词，计算真实 token 数
//   - 近似模式：用"字符数 × 0.38"估算，速度极快但不精确
//     （0.38 是经验值：英文约 4 字符 = 1 token，中文约 1.5 字符 = 1 token，综合约 0.38）
//     开启方式：设置环境变量 APPROXIMATE_TOKEN_ENABLED=true
func getTokenNum(tokenEncoder *tiktoken.Tiktoken, text string) int {
	if config.ApproximateTokenEnabled {
		// 近似模式：字符数 × 0.38，牺牲精度换速度
		return int(float64(len(text)) * 0.38)
	}
	// 精确模式：用分词器编码，统计编码后的 token 数量
	return len(tokenEncoder.Encode(text, nil, nil))
}

// CountTokenMessages 计算 Chat Messages（对话消息列表）的总 token 数
// 这是最核心的计数函数，用于请求前的预扣费
//
// OpenAI 的消息格式不仅计算内容本身的 token，还有额外的"格式开销"：
// 每条消息前后有特殊标记 <|start|>{role}\n{content}<|end|>\n
// 这些标记也会消耗 token，必须算进去
//
// 参考：https://github.com/openai/openai-cookbook/blob/main/examples/How_to_count_tokens_with_tiktoken.ipynb
func CountTokenMessages(messages []model.Message, model string) int {
	tokenEncoder := getTokenEncoder(model)

	// 每条消息的格式开销（特殊标记占的 token）
	// GPT-3.5-turbo-0301 旧版：4 tokens/消息，name 字段会替代 role 所以 -1
	// 其他所有模型：3 tokens/消息，name 额外 1 token
	var tokensPerMessage int
	var tokensPerName int
	if model == "gpt-3.5-turbo-0301" {
		tokensPerMessage = 4
		tokensPerName = -1 // If there's a name, the role is omitted
	} else {
		tokensPerMessage = 3
		tokensPerName = 1
	}

	tokenNum := 0
	for _, message := range messages {
		// 每条消息的格式开销
		tokenNum += tokensPerMessage

		// 计算 content 的 token 数
		// content 有两种格式：纯字符串 或 内容块数组（多模态）
		switch v := message.Content.(type) {
		case string:
			// 纯文本内容，直接计算 token 数
			tokenNum += getTokenNum(tokenEncoder, v)
		case []any:
			// 多模态内容（如图片+文本混合），遍历每个内容块
			for _, it := range v {
				m := it.(map[string]any)
				switch m["type"] {
				case "text":
					// 文本块：计算文本的 token 数
					if textValue, ok := m["text"]; ok {
						if textString, ok := textValue.(string); ok {
							tokenNum += getTokenNum(tokenEncoder, textString)
						}
					}
				case "image_url":
					// 图片块：按 OpenAI 的图片计费规则计算 token 数
					// 图片的 token 数跟分辨率和 detail 参数有关
					imageUrl, ok := m["image_url"].(map[string]any)
					if ok {
						url := imageUrl["url"].(string)
						detail := ""
						if imageUrl["detail"] != nil {
							detail = imageUrl["detail"].(string)
						}
						imageTokens, err := countImageTokens(url, detail, model)
						if err != nil {
							logger.SysError("error counting image tokens: " + err.Error())
						} else {
							tokenNum += imageTokens
						}
					}
				}
			}
		}

		// role 字段（"user"/"assistant"/"system"）本身也占 token
		tokenNum += getTokenNum(tokenEncoder, message.Role)

		// name 字段：如果消息指定了 name，额外占 tokensPerName 个 token
		if message.Name != nil {
			tokenNum += tokensPerName
			tokenNum += getTokenNum(tokenEncoder, *message.Name)
		}
	}

	// 每次回复都预先注入 <|start|>assistant<|message|>，占 3 个 token
	tokenNum += 3
	return tokenNum
}

// ============================================================
// 第三部分：图片 Token 计数
// ============================================================
//
// 【为什么图片也要算 token？】
// OpenAI 的多模态模型（GPT-4V、GPT-4o）处理图片时，会把图片转换成 token。
// 图片消耗的 token 数取决于分辨率和 detail 参数：
//   - low detail：固定 85 tokens（缩放到 512×512 低分辨率处理）
//   - high detail：按图片尺寸切成 512×512 的瓦片，每瓦片 170 tokens + 基础 85 tokens
//
// gpt-4o-mini 的图片 token 单价更高（约 33 倍），所以单独列了常量。

const (
	lowDetailCost         = 85  // low detail 模式固定消耗 85 tokens
	highDetailCostPerTile = 170 // high detail 模式每个 512×512 瓦片消耗 170 tokens
	additionalCost        = 85  // high detail 模式的基础开销
	// gpt-4o-mini 的图片计费更贵（约 33 倍）
	gpt4oMiniLowDetailCost  = 2833
	gpt4oMiniHighDetailCost = 5667
	gpt4oMiniAdditionalCost = 2833
)

// countImageTokens 计算一张图片消耗的 token 数
//
// OpenAI 的图片 token 计算规则（high detail 模式）：
//  1. 如果图片长边 > 2048px，等比缩放到长边 2048px
//  2. 如果图片短边 > 768px，等比缩放到短边 768px
//  3. 将缩放后的图片切成 512×512 的瓦片
//  4. token 数 = 瓦片数 × 170 + 85
//
// 参考：https://platform.openai.com/docs/guides/vision/calculating-costs
func countImageTokens(url string, detail string, model string) (_ int, err error) {
	var fetchSize = true
	var width, height int

	// detail 为空或 "auto" 时的处理
	// OpenAI 文档说 auto 会根据图片大小自动选 low/high，
	// 但实测发现即使很小的图片（125×50）也被当作 high detail 处理，
	// 所以这里统一按 high 处理
	if detail == "" || detail == "auto" {
		detail = "high"
	}

	switch detail {
	case "low":
		// low detail：固定费用，不关心图片实际大小
		if strings.HasPrefix(model, "gpt-4o-mini") {
			return gpt4oMiniLowDetailCost, nil
		}
		return lowDetailCost, nil
	case "high":
		// high detail：需要获取图片实际尺寸来计算瓦片数
		if fetchSize {
			width, height, err = image.GetImageSize(url)
			if err != nil {
				return 0, err
			}
		}
		// 步骤1：长边缩放到 2048px
		if width > 2048 || height > 2048 {
			ratio := float64(2048) / math.Max(float64(width), float64(height))
			width = int(float64(width) * ratio)
			height = int(float64(height) * ratio)
		}
		// 步骤2：短边缩放到 768px
		if width > 768 && height > 768 {
			ratio := float64(768) / math.Min(float64(width), float64(height))
			width = int(float64(width) * ratio)
			height = int(float64(height) * ratio)
		}
		// 步骤3：计算 512×512 瓦片数
		numSquares := int(math.Ceil(float64(width)/512) * math.Ceil(float64(height)/512))
		// 步骤4：token 数 = 瓦片数 × 单价 + 基础费用
		if strings.HasPrefix(model, "gpt-4o-mini") {
			return numSquares*gpt4oMiniHighDetailCost + gpt4oMiniAdditionalCost, nil
		}
		result := numSquares*highDetailCostPerTile + additionalCost
		return result, nil
	default:
		return 0, errors.New("invalid detail option")
	}
}

// ============================================================
// 第四部分：通用 Token 计数工具函数
// ============================================================

// CountTokenInput 计算任意类型输入的 token 数
// 支持 string 和 []string 两种类型（用于 Embeddings 等接口）
func CountTokenInput(input any, model string) int {
	switch v := input.(type) {
	case string:
		return CountTokenText(v, model)
	case []string:
		text := ""
		for _, s := range v {
			text += s
		}
		return CountTokenText(text, model)
	}
	return 0
}

// CountTokenText 计算纯文本的 token 数（最基础的计数函数）
func CountTokenText(text string, model string) int {
	tokenEncoder := getTokenEncoder(model)
	return getTokenNum(tokenEncoder, text)
}

// CountToken 计算纯文本的 token 数，使用默认分词器（GPT-3.5）
// 这是最简化的计数接口，不区分模型
func CountToken(text string) int {
	return CountTokenInput(text, "gpt-3.5-turbo")
}
