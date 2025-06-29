package domain

import "notification-platform/internal/pkg/retry"

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
