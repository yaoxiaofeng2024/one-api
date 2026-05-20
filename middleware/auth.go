package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/blacklist"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/network"
	"github.com/songquanpeng/one-api/model"
)

func authHelper(c *gin.Context, minRole int) {
	session := sessions.Default(c)
	username := session.Get("username")
	role := session.Get("role")
	id := session.Get("id")
	status := session.Get("status")
	if username == nil {
		// Check access token
		accessToken := c.Request.Header.Get("Authorization")
		if accessToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "无权进行此操作，未登录且未提供 access token",
			})
			c.Abort()
			return
		}
		user := model.ValidateAccessToken(accessToken)
		if user != nil && user.Username != "" {
			// Token is valid
			username = user.Username
			role = user.Role
			id = user.Id
			status = user.Status
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无权进行此操作，access token 无效",
			})
			c.Abort()
			return
		}
	}
	if status.(int) == model.UserStatusDisabled || blacklist.IsUserBanned(id.(int)) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户已被封禁",
		})
		session := sessions.Default(c)
		session.Clear()
		_ = session.Save()
		c.Abort()
		return
	}
	if role.(int) < minRole {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权进行此操作，权限不足",
		})
		c.Abort()
		return
	}
	c.Set("username", username)
	c.Set("role", role)
	c.Set("id", id)
	c.Next()
}

func UserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleCommonUser)
	}
}

func AdminAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleAdminUser)
	}
}

func RootAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, model.RoleRootUser)
	}
}

// TokenAuth 是 AI 模型转发（/v1）的鉴权中间件
//
// 与 UserAuth/AdminAuth/RootAuth（基于 Session）不同，TokenAuth 基于 API Key 鉴权，
// 适用于程序化调用场景（如 curl、SDK 调用 OpenAI 兼容接口）。
//
// API Key 格式：sk-{tokenKey}-{channelId}
//   - sk-          : 前缀，兼容 OpenAI 的 key 格式习惯
//   - tokenKey     : 令牌的唯一标识（数据库中 token 表的 key 字段）
//   - channelId    : 可选，管理员通过此字段指定请求走哪个渠道，跳过自动分发
//
// 鉴权流程：
//
//	提取 Key → 验证令牌 → 子网校验 → 用户状态校验 → 模型权限校验 → 写入上下文
func TokenAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// ===== 第一步：从 Authorization 头提取 API Key =====
		// 标准格式为 "Bearer sk-xxxx"，需逐层剥离前缀
		key := c.Request.Header.Get("Authorization")
		key = strings.TrimPrefix(key, "Bearer ") // 去掉 "Bearer " 前缀（OAuth 标准格式）
		key = strings.TrimPrefix(key, "sk-")     // 去掉 "sk-" 前缀，得到 tokenKey[-channelId]

		// 按 "-" 分割：parts[0] = tokenKey，parts[1:] = 可选的 channelId 等附加信息
		// 只取第一段作为令牌标识去数据库查询
		parts := strings.Split(key, "-")
		key = parts[0]

		// ===== 第二步：验证令牌有效性 =====
		// 从数据库/缓存中查询 token，校验是否过期、是否已禁用等
		token, err := model.ValidateUserToken(key)
		if err != nil {
			abortWithMessage(c, http.StatusUnauthorized, err.Error())
			return
		}

		// ===== 第三步：子网访问控制（可选） =====
		// 令牌可配置 IP 白名单（如 "192.168.1.0/24"），限制只能从指定网段调用
		// 防止 API Key 泄露后被异地滥用
		if token.Subnet != nil && *token.Subnet != "" {
			if !network.IsIpInSubnets(ctx, c.ClientIP(), *token.Subnet) {
				abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该令牌只能在指定网段使用：%s，当前 ip：%s", *token.Subnet, c.ClientIP()))
				return
			}
		}

		// ===== 第四步：用户状态校验 =====
		// 即使令牌有效，如果所属用户被禁用或被封禁，请求也要拒绝
		userEnabled, err := model.CacheIsUserEnabled(token.UserId)
		if err != nil {
			abortWithMessage(c, http.StatusInternalServerError, err.Error())
			return
		}
		if !userEnabled || blacklist.IsUserBanned(token.UserId) {
			abortWithMessage(c, http.StatusForbidden, "用户已被封禁")
			return
		}

		// ===== 第五步：提取请求中的模型名，校验模型权限 =====
		// 从请求体中解析出用户要调用的模型名称（如 "gpt-4"、"claude-3-opus"）
		requestModel, err := getRequestModel(c)
		if err != nil && shouldCheckModel(c) {
			// 只对需要模型名的接口（chat/completions/images/audio）强制要求解析成功
			// 其他接口（如 /v1/models 列表）不需要模型名，解析失败也不报错
			abortWithMessage(c, http.StatusBadRequest, err.Error())
			return
		}
		c.Set(ctxkey.RequestModel, requestModel)

		// 令牌可配置可用模型范围（如只允许用 gpt-3.5-turbo）
		// 如果令牌配置了模型白名单，则检查请求的模型是否在白名单内
		if token.Models != nil && *token.Models != "" {
			c.Set(ctxkey.AvailableModels, *token.Models)
			if requestModel != "" && !isModelInList(requestModel, *token.Models) {
				abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("该令牌无权使用模型：%s", requestModel))
				return
			}
		}

		// ===== 第六步：将鉴权信息写入 Gin 上下文，供后续中间件和控制器使用 =====
		c.Set(ctxkey.Id, token.UserId)      // 用户 ID（计费、限流等使用）
		c.Set(ctxkey.TokenId, token.Id)     // 令牌 ID（日志记录使用）
		c.Set(ctxkey.TokenName, token.Name) // 令牌名称（日志记录使用）

		// ===== 第七步：处理管理员指定渠道的特殊逻辑 =====
		// API Key 格式 "sk-tokenKey-channelId" 中的 channelId 部分
		// 管理员可以在 key 中追加渠道 ID，让请求强制走指定渠道（用于调试/测试）
		// 普通用户不允许指定渠道，防止绕过渠道分发策略
		if len(parts) > 1 {
			if model.IsAdmin(token.UserId) {
				c.Set(ctxkey.SpecificChannelId, parts[1])
			} else {
				abortWithMessage(c, http.StatusForbidden, "普通用户不支持指定渠道")
				return
			}
		}

		// 另一种指定渠道的方式：通过 URL 路径参数 /oneapi/proxy/:channelid/*
		// 这是 relay.go 中的自定义代理路由，直接在 URL 中指定渠道
		if channelId := c.Param("channelid"); channelId != "" {
			c.Set(ctxkey.SpecificChannelId, channelId)
		}

		c.Next()
	}
}

func shouldCheckModel(c *gin.Context) bool {
	if strings.HasPrefix(c.Request.URL.Path, "/v1/completions") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/chat/completions") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/images") {
		return true
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/audio") {
		return true
	}
	return false
}
