package model

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/message"
)

const (
	TokenStatusEnabled   = 1 // 令牌可用（不用 0 是因为 0 是 int 默认值，无法区分"未设置"和"禁用"）
	TokenStatusDisabled  = 2 // 令牌已禁用（手动禁用）
	TokenStatusExpired   = 3 // 令牌已过期（超过 expired_time）
	TokenStatusExhausted = 4 // 令牌额度已耗尽（remain_quota 降为 0）
)

type Token struct {
	Id             int     `json:"id"`                                    // 令牌 ID（主键）
	UserId         int     `json:"user_id"`                               // 所属用户 ID（关联 users 表）
	Key            string  `json:"key" gorm:"type:char(48);uniqueIndex"`  // 令牌密钥（API Key 中 sk- 后面的部分，唯一索引）
	Status         int     `json:"status" gorm:"default:1"`               // 令牌状态：1=可用 2=禁用 3=过期 4=额度耗尽
	Name           string  `json:"name" gorm:"index"`                     // 令牌名称（用户自定义，方便识别用途，加索引便于搜索）
	CreatedTime    int64   `json:"created_time" gorm:"bigint"`            // 创建时间（Unix 时间戳）
	AccessedTime   int64   `json:"accessed_time" gorm:"bigint"`           // 最后访问时间（每次调用时更新）
	ExpiredTime    int64   `json:"expired_time" gorm:"bigint;default:-1"` // 过期时间（Unix 时间戳，-1 表示永不过期）
	RemainQuota    int64   `json:"remain_quota" gorm:"bigint;default:0"`  // 剩余配额（额度单位，非美元，由系统换算）
	UnlimitedQuota bool    `json:"unlimited_quota" gorm:"default:false"`  // 是否无限配额（为 true 时忽略 RemainQuota）
	UsedQuota      int64   `json:"used_quota" gorm:"bigint;default:0"`    // 已用配额（累计消耗，用于统计和计费）
	Models         *string `json:"models" gorm:"type:text"`               // 允许使用的模型列表（逗号分隔，为空表示不限制）
	Subnet         *string `json:"subnet" gorm:"default:''"`              // 允许访问的 IP 子网（CIDR 格式，如 "192.168.1.0/24"，为空表示不限制）
}

func GetAllUserTokens(userId int, startIdx int, num int, order string) ([]*Token, error) {
	var tokens []*Token
	var err error
	query := DB.Where("user_id = ?", userId)

	switch order {
	case "remain_quota":
		query = query.Order("unlimited_quota desc, remain_quota desc")
	case "used_quota":
		query = query.Order("used_quota desc")
	default:
		query = query.Order("id desc")
	}

	err = query.Limit(num).Offset(startIdx).Find(&tokens).Error
	return tokens, err
}

func SearchUserTokens(userId int, keyword string) (tokens []*Token, err error) {
	err = DB.Where("user_id = ?", userId).Where("name LIKE ?", keyword+"%").Find(&tokens).Error
	return tokens, err
}

// 验证用户令牌
func ValidateUserToken(key string) (token *Token, err error) {
	if key == "" {
		return nil, errors.New("未提供令牌")
	}

	// 获取令牌
	token, err = CacheGetTokenByKey(key)
	if err != nil {
		logger.SysError("CacheGetTokenByKey failed: " + err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的令牌")
		}
		return nil, errors.New("令牌验证失败")
	}

	// 状态验证
	if token.Status == TokenStatusExhausted {
		return nil, fmt.Errorf("令牌 %s（#%d）额度已用尽", token.Name, token.Id)
	} else if token.Status == TokenStatusExpired {
		return nil, errors.New("该令牌已过期")
	}
	if token.Status != TokenStatusEnabled {
		return nil, errors.New("该令牌状态不可用")
	}

	// 过期时间验证
	if token.ExpiredTime != -1 && token.ExpiredTime < helper.GetTimestamp() {
		if !common.RedisEnabled {
			token.Status = TokenStatusExpired
			err := token.SelectUpdate()
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, errors.New("该令牌已过期")
	}

	// 额度验证
	if !token.UnlimitedQuota && token.RemainQuota <= 0 {
		if !common.RedisEnabled {
			// in this case, we can make sure the token is exhausted
			token.Status = TokenStatusExhausted
			err := token.SelectUpdate()
			if err != nil {
				logger.SysError("failed to update token status" + err.Error())
			}
		}
		return nil, errors.New("该令牌额度已用尽")
	}
	return token, nil
}

func GetTokenByIds(id int, userId int) (*Token, error) {
	if id == 0 || userId == 0 {
		return nil, errors.New("id 或 userId 为空！")
	}
	token := Token{Id: id, UserId: userId}
	var err error = nil
	err = DB.First(&token, "id = ? and user_id = ?", id, userId).Error
	return &token, err
}

func GetTokenById(id int) (*Token, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	token := Token{Id: id}
	var err error = nil
	err = DB.First(&token, "id = ?", id).Error
	return &token, err
}

func (t *Token) Insert() error {
	var err error
	err = DB.Create(t).Error
	return err
}

// Update Make sure your token's fields is completed, because this will update non-zero values
func (t *Token) Update() error {
	var err error
	err = DB.Model(t).Select("name", "status", "expired_time", "remain_quota", "unlimited_quota", "models", "subnet").Updates(t).Error
	return err
}

func (t *Token) SelectUpdate() error {
	// This can update zero values
	return DB.Model(t).Select("accessed_time", "status").Updates(t).Error
}

func (t *Token) Delete() error {
	var err error
	err = DB.Delete(t).Error
	return err
}

func (t *Token) GetModels() string {
	if t == nil {
		return ""
	}
	if t.Models == nil {
		return ""
	}
	return *t.Models
}

func DeleteTokenById(id int, userId int) (err error) {
	// Why we need userId here? In case user want to delete other's token.
	if id == 0 || userId == 0 {
		return errors.New("id 或 userId 为空！")
	}
	token := Token{Id: id, UserId: userId}
	err = DB.Where(token).First(&token).Error
	if err != nil {
		return err
	}
	return token.Delete()
}

func IncreaseTokenQuota(id int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeTokenQuota, id, quota)
		return nil
	}
	return increaseTokenQuota(id, quota)
}

func increaseTokenQuota(id int, quota int64) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota + ?", quota),
			"used_quota":    gorm.Expr("used_quota - ?", quota),
			"accessed_time": helper.GetTimestamp(),
		},
	).Error
	return err
}

func DecreaseTokenQuota(id int, quota int64) (err error) {
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeTokenQuota, id, -quota)
		return nil
	}
	return decreaseTokenQuota(id, quota)
}

func decreaseTokenQuota(id int, quota int64) (err error) {
	err = DB.Model(&Token{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"remain_quota":  gorm.Expr("remain_quota - ?", quota),
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"accessed_time": helper.GetTimestamp(),
		},
	).Error
	return err
}

// PreConsumeTokenQuota 预消费令牌额度
// 在实际调用上游 AI 接口之前，先扣除预估的额度，防止超额使用。
// 扣除是双维度的：令牌维度 + 用户维度，两者都必须有足够余额。
// 流程：校验额度 → 邮件提醒（异步）→ 扣减令牌额度 → 扣减用户额度
// 注意：预消费只是估算，实际用量在 PostConsumeTokenQuota 中结算（多退少补）
func PreConsumeTokenQuota(tokenId int, quota int64) (err error) {
	// 防御性检查：预消费额度不能为负数
	if quota < 0 {
		return errors.New("quota 不能为负数！")
	}

	// 第一步：查询令牌信息，检查令牌维度额度是否充足
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	// UnlimitedQuota=true 表示令牌额度无上限，跳过令牌维度的余额检查
	if !token.UnlimitedQuota && token.RemainQuota < quota {
		return errors.New("令牌额度不足")
	}

	// 第二步：查询用户额度，检查用户维度额度是否充足
	// 即使令牌额度充足，用户总额度也可能不够
	userQuota, err := GetUserQuota(token.UserId)
	if err != nil {
		return err
	}
	if userQuota < quota {
		return errors.New("用户额度不足")
	}

	// 第三步：判断是否需要发送额度提醒邮件
	// quotaTooLow: 扣费前高于提醒阈值，扣费后低于提醒阈值 → 额度即将用尽
	// noMoreQuota: 扣费后用户额度归零或为负 → 额度已用尽
	quotaTooLow := userQuota >= config.QuotaRemindThreshold && userQuota-quota < config.QuotaRemindThreshold
	noMoreQuota := userQuota-quota <= 0
	if quotaTooLow || noMoreQuota {
		// 异步发送邮件，不阻塞主流程（邮件发送失败不影响请求处理）
		go func() {
			email, err := GetUserEmail(token.UserId)
			if err != nil {
				logger.SysError("failed to fetch user email: " + err.Error())
			}
			prompt := "额度提醒"
			var contentText string
			if noMoreQuota {
				contentText = "您的额度已用尽"
			} else {
				contentText = "您的额度即将用尽"
			}
			if email != "" {
				// 构造充值链接和 HTML 邮件内容
				topUpLink := fmt.Sprintf("%s/topup", config.ServerAddress)
				content := message.EmailTemplate(
					prompt,
					fmt.Sprintf(`
						<p>您好！</p>
						<p>%s，当前剩余额度为 <strong>%d</strong>。</p>
						<p>为了不影响您的使用，请及时充值。</p>
						<p style="text-align: center; margin: 30px 0;">
							<a href="%s" style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">立即充值</a>
						</p>
						<p style="color: #666;">如果按钮无法点击，请复制以下链接到浏览器中打开：</p>
						<p style="background-color: #f8f8f8; padding: 10px; border-radius: 4px; word-break: break-all;">%s</p>
					`, contentText, userQuota, topUpLink, topUpLink),
				)
				err = message.SendEmail(prompt, email, content)
				if err != nil {
					logger.SysError("failed to send email: " + err.Error())
				}
			}
		}()
	}

	// 第四步：实际扣减额度（先扣令牌，再扣用户）
	// 无限额度令牌跳过令牌维度扣减，但仍需扣减用户维度额度
	if !token.UnlimitedQuota {
		err = DecreaseTokenQuota(tokenId, quota)
		if err != nil {
			return err
		}
	}
	// 用户维度额度始终需要扣减（即使用户也有"无限额度"，也会走 DecreaseUserQuota 逻辑）
	err = DecreaseUserQuota(token.UserId, quota)
	return err
}

func PostConsumeTokenQuota(tokenId int, quota int64) (err error) {
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	if quota > 0 {
		err = DecreaseUserQuota(token.UserId, quota)
	} else {
		err = IncreaseUserQuota(token.UserId, -quota)
	}
	if !token.UnlimitedQuota {
		if quota > 0 {
			err = DecreaseTokenQuota(tokenId, quota)
		} else {
			err = IncreaseTokenQuota(tokenId, -quota)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
