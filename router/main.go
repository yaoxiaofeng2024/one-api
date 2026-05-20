package router

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

func SetRouter(router *gin.Engine, buildFS embed.FS) {
	// 后台管理 API（/api）：用户、渠道、Token、兑换码、日志、系统选项等 CRUD
	SetApiRouter(router)
	// 计费查询接口（/dashboard、/v1/dashboard）：兼容 OpenAI 官方的 subscription/usage 查询
	SetDashboardRouter(router)
	// AI 模型转发（/v1）：核心网关，将请求转发到上游 AI 供应商（OpenAI、Claude 等）
	SetRelayRouter(router)

	// 前端地址
	frontendBaseUrl := os.Getenv("FRONTEND_BASE_URL")
	if config.IsMasterNode && frontendBaseUrl != "" {
		frontendBaseUrl = ""
		logger.SysLog("FRONTEND_BASE_URL is ignored on master node")
	}

	// 内置前端
	if frontendBaseUrl == "" {
		SetWebRouter(router, buildFS)
	} else {
		// 外部前端
		frontendBaseUrl = strings.TrimSuffix(frontendBaseUrl, "/")
		router.NoRoute(func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("%s%s", frontendBaseUrl, c.Request.RequestURI))
		})
	}
}
