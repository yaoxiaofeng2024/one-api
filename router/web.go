package router

import (
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
)

// SetWebRouter 配置前端静态文件路由（内嵌模式）
//
// 当 FRONTEND_BASE_URL 为空时由 main.go 调用，将编译时嵌入的 React 产物作为静态文件提供服务。
// 支持多主题：通过 config.Theme 选择 web/build/{theme}/ 目录下的前端构建产物。
func SetWebRouter(router *gin.Engine, buildFS embed.FS) {
	// 预读 index.html 到内存，后续 NoRoute 处理时直接返回，避免每次请求都从 embed.FS 读取
	// 忽略错误——如果文件不存在，说明前端未构建或主题配置错误，启动时就该暴露问题
	indexPageData, _ := buildFS.ReadFile(fmt.Sprintf("web/build/%s/index.html", config.Theme))

	// gzip 压缩：对静态资源（JS/CSS/HTML）压缩传输，减少带宽消耗
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// 全局限流：保护 Web 页面请求不被恶意刷爆
	router.Use(middleware.GlobalWebRateLimit())

	// 浏览器缓存中间件：为静态资源设置 Cache-Control 头，减少重复请求
	router.Use(middleware.Cache())

	// 静态文件服务：将 embed.FS 中 web/build/{theme}/ 目录映射到 /
	// 例如请求 /static/js/main.js → 返回 web/build/{theme}/static/js/main.js 文件内容
	router.Use(static.Serve("/", common.EmbedFolder(buildFS, fmt.Sprintf("web/build/%s", config.Theme))))

	// NoRoute 处理：所有未匹配到上述路由的请求走这里
	// 这是 React SPA 的关键——前端路由（如 /dashboard、/channel）在服务端并不存在，
	// 需要统一返回 index.html，由前端 JS 路由接管渲染
	router.NoRoute(func(c *gin.Context) {
		// 区分两类未匹配请求：
		// 1. /v1 或 /api 开头 → 说明是 API 调用但路径写错了，返回 API 风格的 404
		//    不能返回 index.html，否则客户端会收到一坨 HTML 而非 JSON 错误信息
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") {
			controller.RelayNotFound(c)
			return
		}
		// 2. 其他路径（如 /dashboard、/setting）→ 返回 index.html
		//    设置 no-cache 是因为 index.html 是 SPA 入口，必须每次都获取最新版本
		//    否则浏览器缓存了旧 index.html，可能导致 JS/CSS 引用更新后 404
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPageData)
	})
}
