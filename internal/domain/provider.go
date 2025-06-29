package domain

import (
	"fmt"
	"notification-platform/internal/errs"
)

// Channel 通知渠道
type Channel string

const (
	ChannelSMS   Channel = "SMS"    // 短信
	ChannelEmail Channel = "EMAIL"  // 邮件
	ChannelInApp Channel = "IN_APP" // 站内信
)

func (c Channel) String() string {
	return string(c)
}

func (c Channel) IsValid() bool {
	return c == ChannelSMS || c == ChannelEmail || c == ChannelInApp
}

func (c Channel) IsSMS() bool {
	return c == ChannelSMS
}

func (c Channel) IsEmail() bool {
	return c == ChannelEmail
}

func (c Channel) IsInApp() bool {
	return c == ChannelInApp
}

// ProviderStatus 供应商状态
type ProviderStatus string

const (
	ProviderStatusActive   ProviderStatus = "ACTIVE"   // 激活
	ProviderStatusInactive ProviderStatus = "INACTIVE" // 未激活
)

func (p ProviderStatus) String() string {
	return string(p)
}

// Provider 供应商领域模型
type Provider struct {
	ID int64 // 供应商ID
	// ali
	Name    string  // 供应商名称
	Channel Channel // 支持的渠道

	// 基本信息
	Endpoint  string // API入口地址
	RegionID  string
	APIKey    string // API密钥
	APISecret string // API密钥
	APPID     string

	Weight     int // 权重
	QPSLimit   int // 每秒请求数限制
	DailyLimit int // 每日请求数限制

	AuditCallbackURL string // 审核请求回调地址
	Status           ProviderStatus
}

func (p *Provider) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("%w: 供应商名称不能为空", errs.ErrInvalidParameter)
	}

	if !p.Channel.IsValid() {
		return fmt.Errorf("%w: 不支持的渠道类型", errs.ErrInvalidParameter)
	}

	if p.Endpoint == "" {
		return fmt.Errorf("%w: API入口地址不能为空", errs.ErrInvalidParameter)
	}

	if p.APIKey == "" {
		return fmt.Errorf("%w: API Key不能为空", errs.ErrInvalidParameter)
	}

	if p.APISecret == "" {
		return fmt.Errorf("%w: API Secret不能为空", errs.ErrInvalidParameter)
	}

	if p.Weight <= 0 {
		return fmt.Errorf("%w: 权重不能小于等于0", errs.ErrInvalidParameter)
	}

	if p.QPSLimit <= 0 {
		return fmt.Errorf("%w: 每秒请求数限制不能小于等于0", errs.ErrInvalidParameter)
	}

	if p.DailyLimit <= 0 {
		return fmt.Errorf("%w: 每日请求数限制不能小于等于0", errs.ErrInvalidParameter)
	}

	return nil
}
