package controller

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/relay/constant/role"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	billingratio "github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller/validator"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func getAndValidateTextRequest(c *gin.Context, relayMode int) (*relaymodel.GeneralOpenAIRequest, error) {
	textRequest := &relaymodel.GeneralOpenAIRequest{}
	err := common.UnmarshalBodyReusable(c, textRequest)
	if err != nil {
		return nil, err
	}
	if relayMode == relaymode.Moderations && textRequest.Model == "" {
		textRequest.Model = "text-moderation-latest"
	}
	if relayMode == relaymode.Embeddings && textRequest.Model == "" {
		textRequest.Model = c.Param("model")
	}
	err = validator.ValidateTextRequest(textRequest, relayMode)
	if err != nil {
		return nil, err
	}
	return textRequest, nil
}

func getPromptTokens(textRequest *relaymodel.GeneralOpenAIRequest, relayMode int) int {
	switch relayMode {
	case relaymode.ChatCompletions:
		return openai.CountTokenMessages(textRequest.Messages, textRequest.Model)
	case relaymode.Completions:
		return openai.CountTokenInput(textRequest.Prompt, textRequest.Model)
	case relaymode.Moderations:
		return openai.CountTokenInput(textRequest.Input, textRequest.Model)
	}
	return 0
}

func getPreConsumedQuota(textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64) int64 {
	preConsumedTokens := config.PreConsumedQuota + int64(promptTokens)
	if textRequest.MaxTokens != 0 {
		preConsumedTokens += int64(textRequest.MaxTokens)
	}
	return int64(float64(preConsumedTokens) * ratio)
}

// preConsumeQuota 预扣配额：在请求发送到上游前，先扣除估算的配额
//
// 为什么要预扣？
//   - 防止用户余额不足时仍占用上游资源（上游是按量付费的，打了白打就亏了）
//   - 预扣量基于 prompt token 数估算，响应完成后 postConsumeQuota 会多退少补
//
// 信任机制：
//   - 如果用户配额非常充足（> 预扣量的 100 倍），则跳过预扣，直接信任用户
//   - 因为配额充足的用户不可能因为一次请求就耗尽，预扣反而增加无意义的数据库写入
//
// 返回值：
//   - preConsumedQuota: 实际预扣的配额量（可能为 0，如果用户被信任）
//   - ErrorWithStatusCode: 错误信息（配额不足时返回 403）
func preConsumeQuota(ctx context.Context, textRequest *relaymodel.GeneralOpenAIRequest, promptTokens int, ratio float64, meta *meta.Meta) (int64, *relaymodel.ErrorWithStatusCode) {
	// 估算预扣量：基于 prompt token 数 × 倍率，再加上一个安全余量
	// 具体逻辑见 getPreConsumedQuota，通常比实际用量稍多
	preConsumedQuota := getPreConsumedQuota(textRequest, promptTokens, ratio)

	// 查询用户当前剩余配额（优先从缓存读取，减少数据库压力）
	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}

	// 配额不足检查：如果扣除预扣量后为负数，说明余额不够，直接拒绝
	if userQuota-preConsumedQuota < 0 {
		return preConsumedQuota, openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}

	// 先在缓存中扣减用户配额（注意：这里扣了，但如果下面信任判断通过，并不会真正写入数据库）
	err = model.CacheDecreaseUserQuota(meta.UserId, preConsumedQuota)
	if err != nil {
		return preConsumedQuota, openai.ErrorWrapper(err, "decrease_user_quota_failed", http.StatusInternalServerError)
	}

	// 信任判断：如果用户配额是预扣量的 100 倍以上，说明非常充足
	// 此时不需要真正预扣，因为风险极低，而预扣的数据库写入反而浪费性能
	if userQuota > 100*preConsumedQuota {
		preConsumedQuota = 0 // 置零，后续 postConsumeQuota 会直接扣实际用量
		logger.Info(ctx, fmt.Sprintf("user %d has enough quota %d, trusted and no need to pre-consume", meta.UserId, userQuota))
	}

	// 如果需要预扣（非信任用户），在令牌维度也扣减配额
	// 令牌（Token）也有配额限制，和用户配额是两个维度的控制
	if preConsumedQuota > 0 {
		err := model.PreConsumeTokenQuota(meta.TokenId, preConsumedQuota)
		if err != nil {
			// 令牌配额不足，返回 403（注意：用户配额已在缓存中扣减，但令牌配额扣减失败）
			return preConsumedQuota, openai.ErrorWrapper(err, "pre_consume_token_quota_failed", http.StatusForbidden)
		}
	}
	return preConsumedQuota, nil
}

// postConsumeQuota 后消费额度（结算实际用量）
// 与 preConsumeQuota 配对使用：preConsumeQuota 在请求前预扣预估额度，本方法在请求后根据实际用量结算差额。
// quotaDelta = 实际用量 - 预扣额度：
//   - quotaDelta > 0：实际用量更大，继续扣减差额
//   - quotaDelta < 0：预扣多了，退还差额
//   - quotaDelta = 0：预扣精准，无需调整
//
// 参数说明：
//   - usage: 上游返回的实际 token 用量
//   - ratio: 最终计费倍率 = modelRatio × groupRatio
//   - preConsumedQuota: 之前预扣的额度值
//   - modelRatio: 模型倍率
//   - groupRatio: 分组倍率
//   - completionRatio: 补全倍率（输出 token 相对输入 token 的价格倍数）
//   - systemPromptReset: 是否注入了系统提示词（用于日志记录）
func postConsumeQuota(ctx context.Context, usage *relaymodel.Usage, meta *meta.Meta, textRequest *relaymodel.GeneralOpenAIRequest, ratio float64, preConsumedQuota int64, modelRatio float64, groupRatio float64, systemPromptReset bool) {
	// usage 为 nil 说明上游未返回用量信息，属于异常情况，无法计费
	if usage == nil {
		logger.Error(ctx, "usage is nil, which is unexpected")
		return
	}

	// 第一步：根据实际 token 用量计算应扣额度
	var quota int64
	// 获取补全倍率：输出 token 通常比输入 token 贵（如 gpt-4 输出价格是输入的 2 倍）
	completionRatio := billingratio.GetCompletionRatio(textRequest.Model, meta.ChannelType)
	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens

	// 计费公式：额度 = (输入token数 + 输出token数 × 补全倍率) × 计费倍率
	// math.Ceil 向上取整，确保至少收取完整单位
	quota = int64(math.Ceil((float64(promptTokens) + float64(completionTokens)*completionRatio) * ratio))
	// 保底：有 token 消耗但计算结果为 0 时，至少扣 1 单位（防止免费使用）
	if ratio != 0 && quota <= 0 {
		quota = 1
	}
	totalTokens := promptTokens + completionTokens
	if totalTokens == 0 {
		// 总 token 为 0 说明请求出错（上游未正常处理），不扣费
		// 但不能直接 return，因为可能之前已经预扣了额度，需要退还
		quota = 0
	}

	// 第二步：计算差额并结算（多退少补）
	quotaDelta := quota - preConsumedQuota
	err := model.PostConsumeTokenQuota(meta.TokenId, quotaDelta)
	if err != nil {
		logger.Error(ctx, "error consuming token remain quota: "+err.Error())
	}

	// 第三步：刷新用户额度的 Redis 缓存，确保缓存与 DB 一致
	err = model.CacheUpdateUserQuota(ctx, meta.UserId)
	if err != nil {
		logger.Error(ctx, "error update user quota cache: "+err.Error())
	}

	// 第四步：记录消费日志，包含完整的倍率信息，便于排查计费问题
	logContent := fmt.Sprintf("倍率：%.2f × %.2f × %.2f", modelRatio, groupRatio, completionRatio)
	model.RecordConsumeLog(ctx, &model.Log{
		UserId:            meta.UserId,
		ChannelId:         meta.ChannelId,
		PromptTokens:      promptTokens,
		CompletionTokens:  completionTokens,
		ModelName:         textRequest.Model,
		TokenName:         meta.TokenName,
		Quota:             int(quota),
		Content:           logContent,
		IsStream:          meta.IsStream,
		ElapsedTime:       helper.CalcElapsedTime(meta.StartTime),
		SystemPromptReset: systemPromptReset,
	})

	// 第五步：更新用户和渠道的累计用量统计（用于仪表盘展示）
	model.UpdateUserUsedQuotaAndRequestCount(meta.UserId, quota)
	model.UpdateChannelUsedQuota(meta.ChannelId, quota)
}

func getMappedModelName(modelName string, mapping map[string]string) (string, bool) {
	if mapping == nil {
		return modelName, false
	}
	mappedModelName := mapping[modelName]
	if mappedModelName != "" {
		return mappedModelName, true
	}
	return modelName, false
}

func isErrorHappened(meta *meta.Meta, resp *http.Response) bool {
	if resp == nil {
		if meta.ChannelType == channeltype.AwsClaude {
			return false
		}
		return true
	}
	if resp.StatusCode != http.StatusOK &&
		// replicate return 201 to create a task
		resp.StatusCode != http.StatusCreated {
		return true
	}
	if meta.ChannelType == channeltype.DeepL {
		// skip stream check for deepl
		return false
	}

	if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") &&
		// Even if stream mode is enabled, replicate will first return a task info in JSON format,
		// requiring the client to request the stream endpoint in the task info
		meta.ChannelType != channeltype.Replicate {
		return true
	}
	return false
}

func setSystemPrompt(ctx context.Context, request *relaymodel.GeneralOpenAIRequest, prompt string) (reset bool) {
	if prompt == "" {
		return false
	}
	if len(request.Messages) == 0 {
		return false
	}
	if request.Messages[0].Role == role.System {
		request.Messages[0].Content = prompt
		logger.Infof(ctx, "rewrite system prompt")
		return true
	}
	request.Messages = append([]relaymodel.Message{{
		Role:    role.System,
		Content: prompt,
	}}, request.Messages...)
	logger.Infof(ctx, "add system prompt")
	return true
}
