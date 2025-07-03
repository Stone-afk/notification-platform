package domain

import "notification-platform/internal/pkg/retry"

// BusinessConfig 业务配置领域对象
type BusinessConfig struct {
	ID             int64           // 业务标识
	OwnerID        int64           // 业务方ID
	OwnerType      string          // 业务方类型：person-个人,organization-组织
	ChannelConfig  *ChannelConfig  // 渠道配置，JSON格式
	TxnConfig      *TxnConfig      // 事务配置，JSON格式
	RateLimit      int             // 每秒最大请求数
	Quota          *QuotaConfig    // 配额设置，JSON格式
	CallbackConfig *CallbackConfig // 回调配置
	Ctime          int64           // 创建时间
	Utime          int64           // 更新时间
}

type QuotaConfig struct {
	Monthly MonthlyConfig `json:"monthly"`
}

type MonthlyConfig struct {
	SMS   int `json:"sms"`
	EMAIL int `json:"email"`
}

type ChannelConfig struct {
	Channels    []ChannelItem `json:"channels"`
	RetryPolicy *retry.Config `json:"retryPolicy"`
}

type ChannelItem struct {
	Channel  string `json:"channel"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

type TxnConfig struct {
	// 回查方法名
	ServiceName string `json:"serviceName"`
	// 期望事务在 initialDelay秒后完成
	InitialDelay int `json:"initialDelay"`
	// 回查的重试策略
	RetryPolicy *retry.Config `json:"retryPolicy"`
}

type CallbackConfig struct {
	ServiceName string        `json:"serviceName"`
	RetryPolicy *retry.Config `json:"retryPolicy"`
}
