package openai

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
)

// ImageHandler 处理来自 OpenAI 兼容渠道的图片生成响应。
// 它读取响应，解析它，并将其转发给客户端。
// 与文本响应不同，图片响应不包含使用信息。
//
// 参数：
//   - c: HTTP 请求的 Gin 上下文
//   - resp: 上游渠道的 HTTP 响应
//
// 返回值：
//   - 如果发生错误，返回带状态码的错误
//   - 使用信息（图片响应始终为 nil）
func ImageHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.Usage) {
	var imageResponse ImageResponse
	
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
	err = json.Unmarshal(responseBody, &imageResponse)
	if err != nil {
		return ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
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
	
	// 图片响应不包含使用信息
	return nil, nil
}