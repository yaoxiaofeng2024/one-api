package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/render"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/conv"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

const (
	dataPrefix       = "data: " // SSE（Server-Sent Events）数据前缀
	done             = "[DONE]" // SSE 流完成标记
	dataPrefixLength = len(dataPrefix)
)

// StreamHandler 处理来自 OpenAI 兼容渠道的流式响应。
// 它处理 SSE（Server-Sent Events）流，提取响应文本，并将数据转发给客户端。
// 它还在可用时提取使用信息。
//
// 参数：
//   - c: HTTP 请求的 Gin 上下文
//   - resp: 上游渠道的 HTTP 响应
//   - relayMode: 中继操作类型（聊天补全、文本补全等）
//
// 返回值：
//   - 如果发生错误，返回带状态码的错误
//   - 从流中累积的完整响应文本
//   - 如果可用，返回使用信息
func StreamHandler(c *gin.Context, resp *http.Response, relayMode int) (*model.ErrorWithStatusCode, string, *model.Usage) {
	responseText := ""
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	var usage *model.Usage

	// 设置 SSE 流式响应的头
	common.SetEventStreamHeaders(c)

	doneRendered := false
	for scanner.Scan() {
		data := scanner.Text()
		if len(data) < dataPrefixLength {
			// 忽略空行或格式错误的行
			continue
		}
		if data[:dataPrefixLength] != dataPrefix && data[:dataPrefixLength] != done {
			// 跳过不以 "data: " 或 "[DONE]" 开头的行
			continue
		}
		if strings.HasPrefix(data[dataPrefixLength:], done) {
			// 流完成标记
			render.StringData(c, data)
			doneRendered = true
			continue
		}
		switch relayMode {
		case relaymode.ChatCompletions:
			// 解析聊天补全流响应
			var streamResponse ChatCompletionsStreamResponse
			err := json.Unmarshal([]byte(data[dataPrefixLength:]), &streamResponse)
			if err != nil {
				logger.SysError("error unmarshalling stream response: " + err.Error())
				render.StringData(c, data)
				continue
			}
			if len(streamResponse.Choices) == 0 && streamResponse.Usage == nil {
				// 跳过空响应（Azure 常见）
				continue
			}
			// 向客户端转发数据
			render.StringData(c, data)
			// 累积响应文本
			for _, choice := range streamResponse.Choices {
				responseText += conv.AsString(choice.Delta.Content)
			}
			// 如果可用，提取使用信息
			if streamResponse.Usage != nil {
				usage = streamResponse.Usage
			}
		case relaymode.Completions:
			// 解析文本补全流响应
			render.StringData(c, data)
			var streamResponse CompletionsStreamResponse
			err := json.Unmarshal([]byte(data[dataPrefixLength:]), &streamResponse)
			if err != nil {
				logger.SysError("error unmarshalling stream response: " + err.Error())
				continue
			}
			// 累积响应文本
			for _, choice := range streamResponse.Choices {
				responseText += choice.Text
			}
		}
	}

	// 检查扫描器错误
	if err := scanner.Err(); err != nil {
		logger.SysError("error reading stream: " + err.Error())
	}

	// 确保发送完成标记
	if !doneRendered {
		render.Done(c)
	}

	// 关闭响应体
	err := resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), "", nil
	}

	return nil, responseText, usage
}

// Handler 处理来自 OpenAI 兼容渠道的非流式响应。
// 它读取响应体，解析它，检查错误，并将其转发给客户端。
// 它还提取或估算使用信息。
//
// 参数：
//   - c: HTTP 请求的 Gin 上下文
//   - resp: 上游渠道的 HTTP 响应
//   - promptTokens: 提示词中的 token 数量（用于使用估算）
//   - modelName: 使用的模型名称（用于 token 计数）
//
// 返回值：
//   - 如果发生错误，返回带状态码的错误
//   - 使用信息
func Handler(c *gin.Context, resp *http.Response, promptTokens int, modelName string) (*model.ErrorWithStatusCode, *model.Usage) {
	var textResponse SlimTextResponse
	
	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	
	// 关闭响应体
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	
	// 解析响应 JSON
	err = json.Unmarshal(responseBody, &textResponse)
	if err != nil {
		return ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}
	
	// 检查响应中的 API 错误
	if textResponse.Error.Type != "" {
		return &model.ErrorWithStatusCode{
			Error:      textResponse.Error,
			StatusCode: resp.StatusCode,
		}, nil
	}
	
	// 重置响应体以便转发
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	// 从上游响应复制头
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	
	// 设置状态码
	c.Writer.WriteHeader(resp.StatusCode)
	
	// 将响应体复制到客户端
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return ErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError), nil
	}
	
	// 再次关闭响应体
	err = resp.Body.Close()
	if err != nil {
		return ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}

	// 如果未提供或不完整，估算使用信息
	if textResponse.Usage.TotalTokens == 0 || (textResponse.Usage.PromptTokens == 0 && textResponse.Usage.CompletionTokens == 0) {
		completionTokens := 0
		for _, choice := range textResponse.Choices {
			completionTokens += CountTokenText(choice.Message.StringContent(), modelName)
		}
		textResponse.Usage = model.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}
	}
	return nil, &textResponse.Usage
}