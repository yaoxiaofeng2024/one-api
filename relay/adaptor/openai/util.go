package openai

import (
	"context"
	"fmt"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
)

// ErrorWrapper 将普通错误包装成带状态码的错误对象。
// 它记录错误日志并返回统一格式的错误响应。
//
// 参数：
//   - err: 原始错误对象
//   - code: 错误代码（用于标识错误类型）
//   - statusCode: HTTP 状态码
//
// 返回值：
//   - 带状态码的错误对象，包含错误消息、类型和代码
func ErrorWrapper(err error, code string, statusCode int) *model.ErrorWithStatusCode {
	logger.Error(context.TODO(), fmt.Sprintf("[%s]%+v", code, err))

	Error := model.Error{
		Message: err.Error(),
		Type:    "one_api_error",
		Code:    code,
	}
	return &model.ErrorWithStatusCode{
		Error:      Error,
		StatusCode: statusCode,
	}
}
