package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/middleware"
	dbmodel "github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

// https://platform.openai.com/docs/api-reference/chat

func relayHelper(c *gin.Context, relayMode int) *model.ErrorWithStatusCode {
	var err *model.ErrorWithStatusCode
	switch relayMode {
	case relaymode.ImagesGenerations:
		err = controller.RelayImageHelper(c, relayMode)
	case relaymode.AudioSpeech:
		fallthrough
	case relaymode.AudioTranslation:
		fallthrough
	case relaymode.AudioTranscription:
		err = controller.RelayAudioHelper(c, relayMode)
	case relaymode.Proxy:
		err = controller.RelayProxyHelper(c, relayMode)
	default:
		err = controller.RelayTextHelper(c)
	}
	return err
}

// Relay 是所有 AI 模型转发请求的统一入口
//
// 核心职责：
//  1. 根据请求路径判断转发模式（文本/图像/音频/代理）
//  2. 调用 relayHelper 将请求转发到上游渠道
//  3. 如果失败，自动在其他渠道上重试
//  4. 记录渠道健康指标，必要时自动禁用故障渠道
//
// 请求流转：
//
//	TokenAuth 鉴权 → Distribute 渠道分发 → Relay（本函数）→ relayHelper → 上游供应商
func Relay(c *gin.Context) {
	ctx := c.Request.Context()

	// 根据请求路径判断转发模式（chat/embeddings/images/audio 等）
	relayMode := relaymode.GetByPath(c.Request.URL.Path)

	// 调试模式下打印请求体，方便排查问题
	if config.DebugEnabled {
		requestBody, _ := common.GetRequestBody(c)
		logger.Debugf(ctx, "request body: %s", string(requestBody))
	}

	// 从上下文中获取 Distribute 中间件已选好的渠道信息
	channelId := c.GetInt(ctxkey.ChannelId) // 当前使用的渠道 ID
	userId := c.GetInt(ctxkey.Id)           // 当前用户 ID

	// ===== 第一次转发尝试 =====
	bizErr := relayHelper(c, relayMode)
	if bizErr == nil {
		// 转发成功 → 记录渠道健康指标（成功），直接返回
		monitor.Emit(channelId, true)
		return
	}

	// ===== 转发失败，进入重试逻辑 =====

	// 记录失败的渠道 ID，重试时要跳过它，避免再次打到同一故障渠道
	lastFailedChannelId := channelId
	channelName := c.GetString(ctxkey.ChannelName)
	group := c.GetString(ctxkey.Group)                 // 用户分组（如 "default"、"vip"）
	originalModel := c.GetString(ctxkey.OriginalModel) // 用户请求的原始模型名

	// 异步处理渠道错误：记录日志 + 判断是否需要自动禁用该渠道
	go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)

	requestId := c.GetString(helper.RequestIdKey)
	retryTimes := config.RetryTimes // 最大重试次数，由环境变量 RETRY_TIMES 配置

	// 判断是否值得重试（4xx 客户端错误通常不需要重试，5xx 服务端错误才重试）
	if !shouldRetry(c, bizErr.StatusCode) {
		logger.Errorf(ctx, "relay error happen, status code is %d, won't retry in this case", bizErr.StatusCode)
		retryTimes = 0
	}

	// ===== 重试循环 =====
	// 每次重试：随机选一个同组同模型的不同渠道，重新转发
	for i := retryTimes; i > 0; i-- {
		// 从缓存中随机获取一个满足条件的渠道（同 group + 支持 originalModel）
		// 第二个参数 i != retryTimes：如果不是第一次重试，允许选择之前用过的渠道
		// （因为可能所有渠道都试过一轮了，需要回退）
		channel, err := dbmodel.CacheGetRandomSatisfiedChannel(group, originalModel, i != retryTimes)
		if err != nil {
			logger.Errorf(ctx, "CacheGetRandomSatisfiedChannel failed: %+v", err)
			break // 没有可用渠道了，停止重试
		}
		logger.Infof(ctx, "using channel #%d to retry (remain times %d)", channel.Id, i)

		// 跳过刚才失败的渠道，避免无意义重试
		if channel.Id == lastFailedChannelId {
			continue
		}

		// 将新选中的渠道信息写入上下文（替换之前 Distribute 设置的渠道）
		middleware.SetupContextForSelectedChannel(c, channel, originalModel)

		// 重放请求体：因为第一次转发时 Body 已被读取，需要重新设置
		// 否则上游供应商收到空请求体
		requestBody, err := common.GetRequestBody(c)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

		// 用新渠道再次转发
		bizErr = relayHelper(c, relayMode)
		if bizErr == nil {
			return // 重试成功，直接返回
		}

		// 重试也失败了，更新失败渠道 ID，异步记录错误
		channelId := c.GetInt(ctxkey.ChannelId)
		lastFailedChannelId = channelId
		channelName := c.GetString(ctxkey.ChannelName)
		go processChannelRelayError(ctx, userId, channelId, channelName, *bizErr)
	}

	// ===== 所有重试都失败，返回错误给客户端 =====
	if bizErr != nil {
		// 429 Too Many Requests 特殊处理：替换为用户友好的中文提示
		if bizErr.StatusCode == http.StatusTooManyRequests {
			bizErr.Error.Message = "当前分组上游负载已饱和，请稍后再试"
		}

		// BUG: bizErr 存在竞态条件——上面的 goroutine 也在读取 bizErr
		// 将请求 ID 追加到错误信息中，方便用户反馈问题时定位日志
		bizErr.Error.Message = helper.MessageWithRequestId(bizErr.Error.Message, requestId)
		c.JSON(bizErr.StatusCode, gin.H{
			"error": bizErr.Error,
		})
	}
}

func shouldRetry(c *gin.Context, statusCode int) bool {
	if _, ok := c.Get(ctxkey.SpecificChannelId); ok {
		return false
	}
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	if statusCode/100 == 5 {
		return true
	}
	if statusCode == http.StatusBadRequest {
		return false
	}
	if statusCode/100 == 2 {
		return false
	}
	return true
}

func processChannelRelayError(ctx context.Context, userId int, channelId int, channelName string, err model.ErrorWithStatusCode) {
	logger.Errorf(ctx, "relay error (channel id %d, user id: %d): %s", channelId, userId, err.Message)
	// https://platform.openai.com/docs/guides/error-codes/api-errors
	if monitor.ShouldDisableChannel(&err.Error, err.StatusCode) {
		monitor.DisableChannel(channelId, channelName, err.Message)
	} else {
		monitor.Emit(channelId, false)
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := model.Error{
		Message: "API not implemented",
		Type:    "one_api_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := model.Error{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
