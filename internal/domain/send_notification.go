package domain

import "time"

// SendStrategyType 发送策略类型
type SendStrategyType string

const (
	SendStrategyImmediate  SendStrategyType = "IMMEDIATE"   // 立即发送
	SendStrategyDelayed    SendStrategyType = "DELAYED"     // 延迟发送
	SendStrategyScheduled  SendStrategyType = "SCHEDULED"   // 定时发送
	SendStrategyTimeWindow SendStrategyType = "TIME_WINDOW" // 时间窗口发送
	SendStrategyDeadline   SendStrategyType = "DEADLINE"    // 截止日期发送
)

// SendStrategyConfig 发送策略配置
type SendStrategyConfig struct {
	Type          SendStrategyType `json:"type"`          // 发送策略类型
	Delay         time.Duration    `json:"delay"`         // 延迟发送策略使用
	ScheduledTime time.Time        `json:"scheduledTime"` // 定时发送策略使用，计划发送时间
	StartTime     time.Time        `json:"startTime"`     // 窗口发送策略使用，开始时间（毫秒）
	EndTime       time.Time        `json:"endTime"`       // 窗口发送策略使用，结束时间（毫秒）
	DeadlineTime  time.Time        `json:"deadlineTime"`  // 截止日期策略使用，截止日期
}

// SendResponse 发送响应
type SendResponse struct {
	NotificationID uint64     // 通知ID
	Status         SendStatus // 发送状态
}

// BatchSendResponse 批量发送响应
type BatchSendResponse struct {
	Results []SendResponse // 所有结果
}

// BatchSendAsyncResponse 批量异步发送响应
type BatchSendAsyncResponse struct {
	NotificationIDs []uint64 // 生成的通知ID列表
}
