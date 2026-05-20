package model

import (
	"encoding/json"
	"fmt"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
)

const (
	ChannelStatusUnknown          = 0
	ChannelStatusEnabled          = 1 // don't use 0, 0 is the default value!
	ChannelStatusManuallyDisabled = 2 // also don't use 0
	ChannelStatusAutoDisabled     = 3
)

// Channel 渠道表模型，存储上游 AI 服务商的渠道配置
// 每个渠道代表一个上游 API 接入点（如 OpenAI 官方、Azure OpenAI、Claude 等），
// 系统通过渠道将请求转发到对应的上游服务商
type Channel struct {
	Id                 int     `json:"id"`                                                 // 渠道ID，自增主键
	Type               int     `json:"type" gorm:"default:0"`                              // 渠道类型，对应 common/constants.go 中的 ChannelType 常量（如 1=OpenAI, 3=Azure, 14=Anthropic 等）
	Key                string  `json:"key" gorm:"type:text"`                               // 上游 API Key，text 类型支持存储多个 key（用换行符分隔，系统会轮询/随机选择）
	Status             int     `json:"status" gorm:"default:1"`                            // 渠道状态：1=启用，2=禁用，3=自动禁用（上游报错过多时系统自动禁用）
	Name               string  `json:"name" gorm:"index"`                                  // 渠道名称，有索引，用于后台管理展示
	Weight             *uint   `json:"weight" gorm:"default:0"`                            // 权重，*uint 表示可为 nil；同优先级渠道按权重比例分配流量，0=均分
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`                         // 创建时间，Unix 时间戳
	TestTime           int64   `json:"test_time" gorm:"bigint"`                            // 最近一次测试时间，Unix 时间戳
	ResponseTime       int     `json:"response_time"`                                      // 最近一次测试的响应时间（毫秒），用于后台展示渠道延迟
	BaseURL            *string `json:"base_url" gorm:"column:base_url;default:''"`         // 上游 API 基础地址，*string 可为 nil；为空时使用适配器默认地址（如 OpenAI 官方地址）
	Other              *string `json:"other"`                                              // 已废弃：旧版额外配置，请使用 Config 字段替代
	Balance            float64 `json:"balance"`                                            // 上游账户余额（美元），由定时任务查询更新
	BalanceUpdatedTime int64   `json:"balance_updated_time" gorm:"bigint"`                 // 上游余额最近更新时间，Unix 时间戳
	Models             string  `json:"models"`                                             // 该渠道支持的模型列表，逗号分隔（如 "gpt-4,gpt-3.5-turbo,text-davinci-003"），用于路由匹配
	Group              string  `json:"group" gorm:"type:varchar(32);default:'default'"`    // 渠道所属分组，决定哪些用户组可以使用此渠道；需与用户 Group 匹配才能路由到该渠道
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;default:0"`                 // 渠道累计已用额度，用于后台统计各渠道消耗
	ModelMapping       *string `json:"model_mapping" gorm:"type:varchar(1024);default:''"` // 模型映射，JSON 格式，将用户请求的模型名映射为上游实际模型名（如 {"gpt-4":"gpt-4-0613"}）
	Priority           *int64  `json:"priority" gorm:"bigint;default:0"`                   // 优先级，*int64 可为 nil；数值越大优先级越高，系统优先选择高优先级渠道
	Config             string  `json:"config"`                                             // 渠道扩展配置，JSON 格式，存储各适配器特有的配置项（如 Azure 的 API Version、AWS 的 Region 等）
	SystemPrompt       *string `json:"system_prompt" gorm:"type:text"`                     // 系统提示词注入，*string 可为 nil；设置后该渠道的所有请求都会自动 prepend 此系统提示词
}

type ChannelConfig struct {
	Region            string `json:"region,omitempty"`
	SK                string `json:"sk,omitempty"`
	AK                string `json:"ak,omitempty"`
	UserID            string `json:"user_id,omitempty"`
	APIVersion        string `json:"api_version,omitempty"`
	LibraryID         string `json:"library_id,omitempty"`
	Plugin            string `json:"plugin,omitempty"`
	VertexAIProjectID string `json:"vertex_ai_project_id,omitempty"`
	VertexAIADC       string `json:"vertex_ai_adc,omitempty"`
}

func GetAllChannels(startIdx int, num int, scope string) ([]*Channel, error) {
	var channels []*Channel
	var err error
	switch scope {
	case "all":
		err = DB.Order("id desc").Find(&channels).Error
	case "disabled":
		err = DB.Order("id desc").Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Find(&channels).Error
	default:
		err = DB.Order("id desc").Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	}
	return channels, err
}

func SearchChannels(keyword string) (channels []*Channel, err error) {
	err = DB.Omit("key").Where("id = ? or name LIKE ?", helper.String2Int(keyword), keyword+"%").Find(&channels).Error
	return channels, err
}

func GetChannelById(id int, selectAll bool) (*Channel, error) {
	channel := Channel{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&channel, "id = ?", id).Error
	} else {
		err = DB.Omit("key").First(&channel, "id = ?", id).Error
	}
	return &channel, err
}

func BatchInsertChannels(channels []Channel) error {
	var err error
	err = DB.Create(&channels).Error
	if err != nil {
		return err
	}
	for _, channel_ := range channels {
		err = channel_.AddAbilities()
		if err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) GetPriority() int64 {
	if channel.Priority == nil {
		return 0
	}
	return *channel.Priority
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	return *channel.BaseURL
}

func (channel *Channel) GetModelMapping() map[string]string {
	if channel.ModelMapping == nil || *channel.ModelMapping == "" || *channel.ModelMapping == "{}" {
		return nil
	}
	modelMapping := make(map[string]string)
	err := json.Unmarshal([]byte(*channel.ModelMapping), &modelMapping)
	if err != nil {
		logger.SysError(fmt.Sprintf("failed to unmarshal model mapping for channel %d, error: %s", channel.Id, err.Error()))
		return nil
	}
	return modelMapping
}

func (channel *Channel) Insert() error {
	var err error
	err = DB.Create(channel).Error
	if err != nil {
		return err
	}
	err = channel.AddAbilities()
	return err
}

func (channel *Channel) Update() error {
	var err error
	err = DB.Model(channel).Updates(channel).Error
	if err != nil {
		return err
	}
	DB.Model(channel).First(channel, "id = ?", channel.Id)
	err = channel.UpdateAbilities()
	return err
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("response_time", "test_time").Updates(Channel{
		TestTime:     helper.GetTimestamp(),
		ResponseTime: int(responseTime),
	}).Error
	if err != nil {
		logger.SysError("failed to update response time: " + err.Error())
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_time", "balance").Updates(Channel{
		BalanceUpdatedTime: helper.GetTimestamp(),
		Balance:            balance,
	}).Error
	if err != nil {
		logger.SysError("failed to update balance: " + err.Error())
	}
}

func (channel *Channel) Delete() error {
	var err error
	err = DB.Delete(channel).Error
	if err != nil {
		return err
	}
	err = channel.DeleteAbilities()
	return err
}

func (channel *Channel) LoadConfig() (ChannelConfig, error) {
	var cfg ChannelConfig
	if channel.Config == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(channel.Config), &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func UpdateChannelStatusById(id int, status int) {
	err := UpdateAbilityStatus(id, status == ChannelStatusEnabled)
	if err != nil {
		logger.SysError("failed to update ability status: " + err.Error())
	}
	err = DB.Model(&Channel{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		logger.SysError("failed to update channel status: " + err.Error())
	}
}

func UpdateChannelUsedQuota(id int, quota int64) {
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeChannelUsedQuota, id, quota)
		return
	}
	updateChannelUsedQuota(id, quota)
}

func updateChannelUsedQuota(id int, quota int64) {
	err := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error
	if err != nil {
		logger.SysError("failed to update channel used quota: " + err.Error())
	}
}

func DeleteChannelByStatus(status int64) (int64, error) {
	result := DB.Where("status = ?", status).Delete(&Channel{})
	return result.RowsAffected, result.Error
}

func DeleteDisabledChannel() (int64, error) {
	result := DB.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Delete(&Channel{})
	return result.RowsAffected, result.Error
}
