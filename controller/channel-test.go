package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/message"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/monitor"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func buildTestRequest(model string) *relaymodel.GeneralOpenAIRequest {
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	testRequest := &relaymodel.GeneralOpenAIRequest{
		Model: model,
	}
	testMessage := relaymodel.Message{
		Role:    "user",
		Content: config.TestPrompt,
	}
	testRequest.Messages = append(testRequest.Messages, testMessage)
	return testRequest
}

func parseTestResponse(resp string) (*openai.TextResponse, string, error) {
	var response openai.TextResponse
	err := json.Unmarshal([]byte(resp), &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Choices) == 0 {
		return nil, "", errors.New("response has no choices")
	}
	stringContent, ok := response.Choices[0].Content.(string)
	if !ok {
		return nil, "", errors.New("response content is not string")
	}
	return &response, stringContent, nil
}

// testChannel 测试单个渠道的可用性
// 模拟一次 ChatCompletions 请求的完整流程：构建请求 → 发送 → 解析响应
// 返回值：
//   - responseMessage: 测试响应内容摘要
//   - err: Go 标准错误（网络错误、HTTP 非200等）
//   - openaiErr: OpenAI 格式的业务错误（如额度不足、模型不可用等）
func testChannel(ctx context.Context, channel *model.Channel, request *relaymodel.GeneralOpenAIRequest) (responseMessage string, err error, openaiErr *relaymodel.Error) {
	startTime := time.Now()

	// ===== 第一步：构建模拟的 Gin 上下文 =====
	// 使用 httptest.NewRecorder 作为响应写入器，无需真正启动 HTTP 服务
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/v1/chat/completions"},
		Body:   nil,
		Header: make(http.Header),
	}
	c.Request.Header.Set("Authorization", "Bearer "+channel.Key)
	c.Request.Header.Set("Content-Type", "application/json")

	// 将渠道信息注入上下文，后续 adaptor 会从上下文中读取
	c.Set(ctxkey.Channel, channel.Type)         // 渠道类型（如 OpenAI、Azure 等）
	c.Set(ctxkey.BaseURL, channel.GetBaseURL()) // 渠道基础 URL
	cfg, _ := channel.LoadConfig()
	c.Set(ctxkey.Config, cfg)                                 // 渠道自定义配置
	middleware.SetupContextForSelectedChannel(c, channel, "") // 设置上下文中的渠道相关字段

	// ===== 第二步：获取对应渠道类型的适配器 =====
	meta := meta.GetByContext(c)
	apiType := channeltype.ToAPIType(channel.Type)
	adaptor := relay.GetAdaptor(apiType) // 不同渠道类型有不同的请求/响应格式适配器
	if adaptor == nil {
		return "", fmt.Errorf("invalid api type: %d, adaptor is nil", apiType), nil
	}
	adaptor.Init(meta)

	// ===== 第三步：确定测试使用的模型名称 =====
	modelName := request.Model
	modelMap := channel.GetModelMapping() // 模型名映射，如 "gpt-4" -> "gpt-4-32k"
	// 如果请求未指定模型，或指定模型不在渠道支持列表中，则取渠道支持的第一个模型
	if modelName == "" || !strings.Contains(channel.Models, modelName) {
		modelNames := strings.Split(channel.Models, ",")
		if len(modelNames) > 0 {
			modelName = modelNames[0]
		}
	}
	// 如果有模型映射，将原始模型名转换为上游实际模型名
	if modelMap != nil && modelMap[modelName] != "" {
		modelName = modelMap[modelName]
	}
	meta.OriginModelName, meta.ActualModelName = request.Model, modelName
	request.Model = modelName

	// ===== 第四步：通过适配器转换请求格式并序列化 =====
	// 不同渠道（如 Anthropic、Azure）的请求格式不同，adaptor 负责统一转换
	convertedRequest, err := adaptor.ConvertRequest(c, relaymode.ChatCompletions, request)
	if err != nil {
		return "", err, nil
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		return "", err, nil
	}

	// ===== defer：函数返回后记录测试日志 =====
	defer func() {
		logContent := fmt.Sprintf("渠道 %s 测试成功，响应：%s", channel.Name, responseMessage)
		if err != nil || openaiErr != nil {
			errorMessage := ""
			if err != nil {
				errorMessage = err.Error()
			} else {
				errorMessage = openaiErr.Message
			}
			logContent = fmt.Sprintf("渠道 %s 测试失败，错误：%s", channel.Name, errorMessage)
		}
		// 异步写入测试日志到数据库，不影响主流程
		go model.RecordTestLog(ctx, &model.Log{
			ChannelId:   channel.Id,
			ModelName:   modelName,
			Content:     logContent,
			ElapsedTime: helper.CalcElapsedTime(startTime),
		})
	}()

	// ===== 第五步：发送请求 =====
	logger.SysLog(string(jsonData))
	requestBody := bytes.NewBuffer(jsonData)
	c.Request.Body = io.NopCloser(requestBody)
	resp, err := adaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		// 网络层面的错误（DNS 解析失败、连接超时等）
		return "", err, nil
	}

	// ===== 第六步：检查 HTTP 响应状态码 =====
	if resp != nil && resp.StatusCode != http.StatusOK {
		// 非 200 响应，解析 OpenAI 格式的错误信息
		err := controller.RelayErrorHandler(resp)
		errorMessage := err.Error.Message
		if errorMessage != "" {
			errorMessage = ", error message: " + errorMessage
		}
		return "", fmt.Errorf("http status code: %d%s", resp.StatusCode, errorMessage), &err.Error
	}

	// ===== 第七步：解析响应体 =====
	// DoResponse 将上游响应解析为统一的 usage 格式，同时将响应写入 w（httptest.Recorder）
	usage, respErr := adaptor.DoResponse(c, resp, meta)
	if respErr != nil {
		return "", fmt.Errorf("%s", respErr.Error.Message), &respErr.Error
	}
	if usage == nil {
		// 正常情况下 usage 不应为 nil，若为 nil 说明响应格式异常
		return "", errors.New("usage is nil"), nil
	}

	// ===== 第八步：提取并解析响应文本 =====
	rawResponse := w.Body.String()
	_, responseMessage, err = parseTestResponse(rawResponse)
	if err != nil {
		logger.SysError(fmt.Sprintf("failed to parse error: %s, \nresponse: %s", err.Error(), rawResponse))
		return "", err, nil
	}

	// 记录完整响应到日志，用于排查问题
	result := w.Result()
	respBody, err := io.ReadAll(result.Body)
	if err != nil {
		return "", err, nil
	}
	logger.SysLog(fmt.Sprintf("testing channel #%d, response: \n%s", channel.Id, string(respBody)))

	return responseMessage, nil, nil
}

func TestChannel(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel, err := model.GetChannelById(id, true)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	modelName := c.Query("model")
	testRequest := buildTestRequest(modelName)
	tik := time.Now()
	responseMessage, err, _ := testChannel(ctx, channel, testRequest)
	tok := time.Now()
	milliseconds := tok.Sub(tik).Milliseconds()
	if err != nil {
		milliseconds = 0
	}
	go channel.UpdateResponseTime(milliseconds)
	consumedTime := float64(milliseconds) / 1000.0
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":   false,
			"message":   err.Error(),
			"time":      consumedTime,
			"modelName": modelName,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   responseMessage,
		"time":      consumedTime,
		"modelName": modelName,
	})
	return
}

var testAllChannelsLock sync.Mutex
var testAllChannelsRunning bool = false

// testChannels 批量测试所有渠道的可用性
// ctx: 上下文，用于控制请求生命周期
// notify: 测试完成后是否发送通知
// scope: 渠道筛选范围
//
// 核心逻辑：
//  1. 通过互斥锁防止并发重复执行
//  2. 在后台 goroutine 中逐个测试渠道
//  3. 根据测试结果自动禁用/启用渠道，并更新响应时间
func testChannels(ctx context.Context, notify bool, scope string) error {
	// 确保根用户邮箱已加载，后续发送通知时需要
	if config.RootUserEmail == "" {
		config.RootUserEmail = model.GetRootUserEmail()
	}

	// ===== 互斥锁：防止多轮测试并发执行 =====
	testAllChannelsLock.Lock()
	if testAllChannelsRunning {
		testAllChannelsLock.Unlock()
		return errors.New("测试已在运行中")
	}
	testAllChannelsRunning = true
	testAllChannelsLock.Unlock()

	// 查询所有渠道（0, 0 表示不分页，获取全部）
	channels, err := model.GetAllChannels(0, 0, scope)
	if err != nil {
		return err
	}

	// 计算响应时间禁用阈值（秒转毫秒）
	var disableThreshold = int64(config.ChannelDisableThreshold * 1000)
	if disableThreshold == 0 {
		disableThreshold = 10000000 // 未配置阈值时设为极大值，相当于不禁用
	}

	// 在后台 goroutine 中逐个测试，避免阻塞调用方
	go func() {
		for _, channel := range channels {
			isChannelEnabled := channel.Status == model.ChannelStatusEnabled

			// 计时开始
			tik := time.Now()
			testRequest := buildTestRequest("")
			_, err, openaiErr := testChannel(ctx, channel, testRequest)
			// 计时结束，计算响应耗时（毫秒）
			tok := time.Now()
			milliseconds := tok.Sub(tik).Milliseconds()

			// ===== 场景一：渠道启用 + 响应超时 =====
			if isChannelEnabled && milliseconds > disableThreshold {
				err = fmt.Errorf("响应时间 %.2fs 超过阈值 %.2fs", float64(milliseconds)/1000.0, float64(disableThreshold)/1000.0)
				if config.AutomaticDisableChannelEnabled {
					// 开启了自动禁用：直接禁用该渠道
					monitor.DisableChannel(channel.Id, channel.Name, err.Error())
				} else {
					// 未开启自动禁用：仅发送通知提醒管理员
					_ = message.Notify(message.ByAll, fmt.Sprintf("渠道 %s （%d）测试超时", channel.Name, channel.Id), "", err.Error())
				}
			}

			// ===== 场景二：渠道启用 + 返回错误需禁用 =====
			// ShouldDisableChannel 根据 OpenAI 错误码判断是否应该禁用
			if isChannelEnabled && monitor.ShouldDisableChannel(openaiErr, -1) {
				monitor.DisableChannel(channel.Id, channel.Name, err.Error())
			}

			// ===== 场景三：渠道已禁用 + 测试通过可恢复 =====
			// ShouldEnableChannel 判断错误是否为暂时性的，渠道可以重新启用
			if !isChannelEnabled && monitor.ShouldEnableChannel(err, openaiErr) {
				monitor.EnableChannel(channel.Id, channel.Name)
			}

			// 无论成功失败，都更新该渠道的响应时间记录
			channel.UpdateResponseTime(milliseconds)

			// 每个渠道测试之间间隔一段时间，避免对上游 API 造成压力
			time.Sleep(config.RequestInterval)
		}

		// ===== 测试结束：释放锁 + 发送通知 =====
		testAllChannelsLock.Lock()
		testAllChannelsRunning = false
		testAllChannelsLock.Unlock()
		if notify {
			err := message.Notify(message.ByAll, "渠道测试完成", "", "渠道测试完成，如果没有收到禁用通知，说明所有渠道都正常")
			if err != nil {
				logger.SysError(fmt.Sprintf("failed to send email: %s", err.Error()))
			}
		}
	}()
	return nil
}

func TestChannels(c *gin.Context) {
	ctx := c.Request.Context()
	scope := c.Query("scope")
	if scope == "" {
		scope = "all"
	}
	err := testChannels(ctx, true, scope)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func AutomaticallyTestChannels(frequency int) {
	ctx := context.Background()
	for {
		time.Sleep(time.Duration(frequency) * time.Minute)
		logger.SysLog("testing all channels")
		_ = testChannels(ctx, false, "all")
		logger.SysLog("channel test finished")
	}
}
