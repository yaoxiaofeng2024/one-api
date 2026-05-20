// controller/model.go — 模型列表接口
//
// 本文件实现了 OpenAI 兼容的模型查询接口，让客户端能像调用 OpenAI 官方 API 一样
// 查询当前系统支持的模型列表。同时提供管理后台用的模型查询功能。
//
// 核心数据结构：
//   - models:        全量模型列表，启动时从各渠道适配器收集，格式兼容 OpenAI /v1/models 响应
//   - modelsMap:     模型名 → 模型详情的映射，用于按名称快速查找（RetrieveModel 使用）
//   - channelId2Models: 渠道类型 → 该渠道支持的模型名列表，用于管理后台展示
package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/model"
	relay "github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/apitype"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

// https://platform.openai.com/docs/api-reference/models/list

// OpenAIModelPermission 模型权限信息
// 完全兼容 OpenAI 官方 API 的响应格式，客户端 SDK 可直接解析
// 大部分字段是固定值，因为 one-api 不需要细粒度的模型权限控制
type OpenAIModelPermission struct {
	Id                 string  `json:"id"`                   // 权限 ID（固定值，仅做格式兼容）
	Object             string  `json:"object"`               // 对象类型，固定为 "model_permission"
	Created            int     `json:"created"`              // 创建时间
	AllowCreateEngine  bool    `json:"allow_create_engine"`  // 是否允许创建引擎
	AllowSampling      bool    `json:"allow_sampling"`       // 是否允许采样调用
	AllowLogprobs      bool    `json:"allow_logprobs"`       // 是否允许获取 logprobs
	AllowSearchIndices bool    `json:"allow_search_indices"` // 是否允许搜索索引
	AllowView          bool    `json:"allow_view"`           // 是否允许查看
	AllowFineTuning    bool    `json:"allow_fine_tuning"`    // 是否允许微调
	Organization       string  `json:"organization"`         // 所属组织（"*" 表示所有）
	Group              *string `json:"group"`                // 所属分组
	IsBlocking         bool    `json:"is_blocking"`          // 是否被阻止
}

// OpenAIModels 单个模型信息
// 兼容 OpenAI /v1/models 和 /v1/models/:model 的响应格式
type OpenAIModels struct {
	Id         string                  `json:"id"`         // 模型 ID（如 "gpt-4"、"claude-3-opus"）
	Object     string                  `json:"object"`     // 对象类型，固定为 "model"
	Created    int                     `json:"created"`    // 创建时间（Unix 时间戳）
	OwnedBy    string                  `json:"owned_by"`   // 模型提供方名称（如 "openai"、"anthropic"）
	Permission []OpenAIModelPermission `json:"permission"` // 权限列表
	Root       string                  `json:"root"`       // 根模型名（通常与 Id 相同）
	Parent     *string                 `json:"parent"`     // 父模型（微调模型才有）
}

// models 全量模型列表，启动时从各渠道适配器收集
var models []OpenAIModels

// modelsMap 模型名 → 模型详情映射，用于 RetrieveModel 按名称 O(1) 查找
var modelsMap map[string]OpenAIModels

// channelId2Models 渠道类型 → 该渠道支持的模型名列表
// key 是 channeltype 包中定义的渠道类型 ID（如 1=OpenAI, 3=Azure, 14=Anthropic）
// value 是该渠道类型支持的所有模型名称
var channelId2Models map[int][]string

func init() {
	// 构造一个通用的权限对象，所有模型共用
	// one-api 不做模型级别的细粒度权限控制，这些字段只是为了兼容 OpenAI 响应格式
	var permission []OpenAIModelPermission
	permission = append(permission, OpenAIModelPermission{
		Id:                 "modelperm-LwHkVFn8AcMItP432fKKDIKJ",
		Object:             "model_permission",
		Created:            1626777600,
		AllowCreateEngine:  true,
		AllowSampling:      true,
		AllowLogprobs:      true,
		AllowSearchIndices: false,
		AllowView:          true,
		AllowFineTuning:    false,
		Organization:       "*",
		Group:              nil,
		IsBlocking:         false,
	})

	// ===== 第一部分：从各 API 类型的适配器收集模型 =====
	// 遍历所有 API 类型（OpenAI、Claude、Gemini、百度、阿里等），
	// 每个适配器知道自己支持哪些模型，通过 GetModelList() 返回
	for i := 0; i < apitype.Dummy; i++ {
		if i == apitype.AIProxyLibrary {
			continue // 跳过 AIProxyLibrary 类型，它不是具体的 AI 供应商
		}
		adaptor := relay.GetAdaptor(i)
		channelName := adaptor.GetChannelName() // 如 "openai"、"anthropic"
		modelNames := adaptor.GetModelList()    // 如 ["gpt-4", "gpt-3.5-turbo", ...]
		for _, modelName := range modelNames {
			models = append(models, OpenAIModels{
				Id:         modelName,
				Object:     "model",
				Created:    1626777600,
				OwnedBy:    channelName,
				Permission: permission,
				Root:       modelName,
				Parent:     nil,
			})
		}
	}

	// ===== 第二部分：收集 OpenAI 兼容渠道的模型 =====
	// 某些渠道（如各种中转站）使用 OpenAI 协议，但模型名不同
	// 这些渠道的模型不通过适配器注册，而是通过 CompatibleChannels 机制收集
	for _, channelType := range openai.CompatibleChannels {
		if channelType == channeltype.Azure {
			continue // Azure 走独立的适配器，不在这里处理
		}
		channelName, channelModelList := openai.GetCompatibleChannelMeta(channelType)
		for _, modelName := range channelModelList {
			models = append(models, OpenAIModels{
				Id:         modelName,
				Object:     "model",
				Created:    1626777600,
				OwnedBy:    channelName,
				Permission: permission,
				Root:       modelName,
				Parent:     nil,
			})
		}
	}

	// ===== 构建模型名 → 模型对象的映射表 =====
	// 用于 RetrieveModel 接口按名称快速查找，避免遍历
	modelsMap = make(map[string]OpenAIModels)
	for _, model := range models {
		fmt.Println(model.Id)
		modelsMap[model.Id] = model
	}

	// ===== 构建渠道类型 → 模型名列表的映射表 =====
	// 用于 DashboardListModels 接口，管理后台展示"每个渠道类型支持哪些模型"
	channelId2Models = make(map[int][]string)
	for i := 1; i < channeltype.Dummy; i++ {
		adaptor := relay.GetAdaptor(channeltype.ToAPIType(i))
		meta := &meta.Meta{
			ChannelType: i,
		}
		adaptor.Init(meta)
		channelId2Models[i] = adaptor.GetModelList()
	}
}

// DashboardListModels 管理后台接口：返回所有渠道类型及其支持的模型列表
//
// 路由: GET /api/models
// 响应格式: { "success": true, "data": { 1: ["gpt-4", ...], 14: ["claude-3-opus", ...] } }
// key 是渠道类型 ID，value 是该渠道支持的模型名数组
func DashboardListModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channelId2Models,
	})
}

// ListAllModels 管理后台接口：返回全量模型列表（OpenAI 格式）
//
// 路由: GET /api/channel/models
// 返回所有注册的模型，不区分用户权限，用于管理员添加渠道时选择模型
func ListAllModels(c *gin.Context) {
	c.JSON(200, gin.H{
		"object": "list",
		"data":   models,
	})
}

// ListModels OpenAI 兼容接口：返回当前用户可用的模型列表
//
// 路由: GET /v1/models
// 这是 OpenAI 官方 API 的标准接口，客户端 SDK 调用此接口获取可用模型
//
// 过滤逻辑：
//  1. 如果令牌配置了模型白名单（token.Models），则只返回白名单中的模型
//  2. 否则根据用户所属用户组（group）获取该组允许的模型列表
//  3. 在系统注册模型中能匹配到的，返回完整模型信息
//  4. 在系统注册模型中匹配不到的（自定义模型名），构造一个 owned_by="custom" 的占位对象返回
func ListModels(c *gin.Context) {
	ctx := c.Request.Context()
	var availableModels []string

	// 确定当前用户允许使用的模型列表
	if c.GetString(ctxkey.AvailableModels) != "" {
		// 令牌配置了模型白名单 → 使用白名单
		availableModels = strings.Split(c.GetString(ctxkey.AvailableModels), ",")
	} else {
		// 令牌未配置白名单 → 根据用户所属组的权限确定可用模型
		userId := c.GetInt(ctxkey.Id)
		userGroup, _ := model.CacheGetUserGroup(userId)
		availableModels, _ = model.CacheGetGroupModels(ctx, userGroup)
	}

	// 用 map 标记哪些模型是当前用户可用的
	modelSet := make(map[string]bool)
	for _, availableModel := range availableModels {
		modelSet[availableModel] = true
	}

	// 遍历系统注册的全量模型，筛选出用户可用的
	availableOpenAIModels := make([]OpenAIModels, 0)
	for _, model := range models {
		if _, ok := modelSet[model.Id]; ok {
			modelSet[model.Id] = false // 标记为已匹配，避免下面重复添加
			availableOpenAIModels = append(availableOpenAIModels, model)
		}
	}

	// 处理用户可用但系统未注册的模型（管理员自定义的模型名）
	// 这些模型没有在适配器中注册，所以没有完整的模型信息
	// 构造一个简单的占位对象返回，owned_by 标记为 "custom"
	for modelName, ok := range modelSet {
		if ok { // 仍然为 true 说明上面的遍历没匹配到
			availableOpenAIModels = append(availableOpenAIModels, OpenAIModels{
				Id:      modelName,
				Object:  "model",
				Created: 1626777600,
				OwnedBy: "custom",
				Root:    modelName,
				Parent:  nil,
			})
		}
	}

	c.JSON(200, gin.H{
		"object": "list",
		"data":   availableOpenAIModels,
	})
}

// RetrieveModel OpenAI 兼容接口：查询单个模型详情
//
// 路由: GET /v1/models/:model
// 按模型名从 modelsMap 中查找，找不到则返回 OpenAI 风格的错误
func RetrieveModel(c *gin.Context) {
	modelId := c.Param("model")
	if model, ok := modelsMap[modelId]; ok {
		c.JSON(200, model)
	} else {
		Error := relaymodel.Error{
			Message: fmt.Sprintf("The model '%s' does not exist", modelId),
			Type:    "invalid_request_error",
			Param:   "model",
			Code:    "model_not_found",
		}
		c.JSON(200, gin.H{
			"error": Error,
		})
	}
}

// GetUserAvailableModels 用户个人接口：返回当前用户所在组可用的模型名列表
//
// 路由: GET /api/user/available_models
// 前端页面用此接口决定"模型选择下拉框"中展示哪些模型
// 与 ListModels 不同，这个接口只返回模型名数组，不返回 OpenAI 格式的完整信息
func GetUserAvailableModels(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.GetInt(ctxkey.Id)
	userGroup, err := model.CacheGetUserGroup(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	models, err := model.CacheGetGroupModels(ctx, userGroup)
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
		"data":    models,
	})
	return
}
