package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/billing"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

// RelayTextHelper 文本类请求的转发处理器
//
// 处理 ChatCompletions、Completions、Embeddings、Moderations、Edits 等文本类接口。
// 是 relayHelper 中 default 分支的调用目标，也是系统最核心的转发逻辑。
//
// 完整流程：
//
//	解析请求 → 模型名映射 → 预扣配额 → 获取适配器 → 构建请求体 → 发送请求 → 处理响应 → 结算配额
func RelayTextHelper(c *gin.Context) *model.ErrorWithStatusCode {
	ctx := c.Request.Context()

	// 从 Gin 上下文中构建 Meta 对象
	// Meta 包含了渠道、用户、令牌、模型等所有转发所需的元信息
	meta := meta.GetByContext(c)

	// ===== 第一步：解析并校验请求体 =====
	textRequest, err := getAndValidateTextRequest(c, meta.Mode)
	if err != nil {
		logger.Errorf(ctx, "getAndValidateTextRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_text_request", http.StatusBadRequest)
	}
	meta.IsStream = textRequest.Stream // 是否流式响应（SSE），影响后续响应处理方式

	// ===== 第二步：模型名映射 =====
	// 管理员可以配置模型名映射（如 "gpt-4" → "gpt-4-0613"）
	// 用户请求的是 OriginModelName，实际发给上游的是映射后的 ActualModelName
	meta.OriginModelName = textRequest.Model                                        // 用户请求的原始模型名
	textRequest.Model, _ = getMappedModelName(textRequest.Model, meta.ModelMapping) // 映射后的模型名
	meta.ActualModelName = textRequest.Model                                        // 实际使用的模型名

	// ===== 第三步：注入系统提示词（可选） =====
	// 管理员可配置强制系统提示词，会在用户消息前自动插入
	// 返回值表示是否注入了提示词，用于后续计费时扣除额外 token 费用
	systemPromptReset := setSystemPrompt(ctx, textRequest, meta.ForcedSystemPrompt)

	// ===== 第四步：计算计费倍率 =====
	// 最终倍率 = 模型倍率 × 用户组倍率
	// 例如：gpt-4 模型倍率 15.0 × VIP 组倍率 0.8 = 实际倍率 12.0
	modelRatio := billingratio.GetModelRatio(textRequest.Model, meta.ChannelType) // 模型倍率（不同模型单价不同）
	groupRatio := billingratio.GetGroupRatio(meta.Group)                          // 用户组倍率（VIP 折扣等）
	ratio := modelRatio * groupRatio

	// ===== 第五步：预扣配额 =====
	// 在请求发送前先扣除一部分配额，防止用户余额不足时仍占用上游资源
	// 预扣量基于 prompt token 数量估算，响应完成后会多退少补
	promptTokens := getPromptTokens(textRequest, meta.Mode) // 计算 prompt 消耗的 token 数
	meta.PromptTokens = promptTokens
	preConsumedQuota, bizErr := preConsumeQuota(ctx, textRequest, promptTokens, ratio, meta)
	if bizErr != nil {
		logger.Warnf(ctx, "preConsumeQuota failed: %+v", *bizErr)
		return bizErr // 配额不足，直接拒绝
	}

	// ===== 第六步：获取渠道适配器并初始化 =====
	// 适配器模式：不同供应商（OpenAI、Claude、Gemini 等）有不同的请求/响应格式
	// 适配器负责统一转换，使上层代码无需关心供应商差异
	adaptor := relay.GetAdaptor(meta.APIType)
	if adaptor == nil {
		return openai.ErrorWrapper(fmt.Errorf("invalid api type: %d", meta.APIType), "invalid_api_type", http.StatusBadRequest)
	}
	adaptor.Init(meta)

	// ===== 第七步：构建请求体 =====
	// 如果是纯 OpenAI 请求且无特殊配置，可以直接透传原始请求体（零拷贝，性能最优）
	// 否则需要通过适配器转换请求格式（如将 OpenAI 格式转为 Claude 格式）
	requestBody, err := getRequestBody(c, meta, textRequest, adaptor)
	if err != nil {
		return openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	// ===== 第八步：发送请求到上游供应商 =====
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	// 检查上游是否返回了错误（如 401 认证失败、429 限流等）
	if isErrorHappened(meta, resp) {
		// 上游报错 → 退回预扣的配额（请求没成功，不该扣费）
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return RelayErrorHandler(resp)
	}

	// ===== 第九步：处理响应 =====
	// 非流式：一次性读取完整响应体，解析 usage（token 用量）
	// 流式（SSE）：逐行读取 data: {...}，累加 usage，实时转发给客户端
	usage, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		// 响应处理失败 → 退回预扣配额
		billing.ReturnPreConsumedQuota(ctx, preConsumedQuota, meta.TokenId)
		return respErr
	}

	// ===== 第十步：结算配额（异步） =====
	// 根据实际 token 用量（prompt + completion）计算最终费用
	// 多扣的退回，少扣的补扣
	// 异步执行是为了不阻塞响应返回（用户已经拿到结果了，结算可以后台慢慢算）
	go postConsumeQuota(ctx, usage, meta, textRequest, ratio, preConsumedQuota, modelRatio, groupRatio, systemPromptReset)
	return nil
}

func getRequestBody(c *gin.Context, meta *meta.Meta, textRequest *model.GeneralOpenAIRequest, adaptor adaptor.Adaptor) (io.Reader, error) {
	if !config.EnforceIncludeUsage &&
		meta.APIType == apitype.OpenAI &&
		meta.OriginModelName == meta.ActualModelName &&
		meta.ChannelType != channeltype.Baichuan &&
		meta.ForcedSystemPrompt == "" {
		// no need to convert request for openai
		return c.Request.Body, nil
	}

	// get request body
	var requestBody io.Reader
	convertedRequest, err := adaptor.ConvertRequest(c, meta.Mode, textRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request failed: %s\n", err.Error())
		return nil, err
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request json_marshal_failed: %s\n", err.Error())
		return nil, err
	}
	logger.Debugf(c.Request.Context(), "converted request: \n%s", string(jsonData))
	requestBody = bytes.NewBuffer(jsonData)
	return requestBody, nil
}
