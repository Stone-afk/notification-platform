package domain

import (
	"fmt"
	"notification-platform/internal/errs"
	"time"
)

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

// SendTimeWindow 计算最早发送时间和最晚发送时间
func (e SendStrategyConfig) SendTimeWindow() (stime, etime time.Time) {
	switch e.Type {
	case SendStrategyImmediate:
		now := time.Now()
		const defaultEndDuration = 30 * time.Minute
		return now, now.Add(defaultEndDuration)
	case SendStrategyDelayed:
		now := time.Now()
		return now, now.Add(e.Delay)
	case SendStrategyDeadline:
		now := time.Now()
		return now, e.DeadlineTime
	case SendStrategyTimeWindow:
		return e.StartTime, e.EndTime
	case SendStrategyScheduled:
		// 无法精确控制，所以允许一些误差
		const scheduledTimeTolerance = 3 * time.Second
		return e.ScheduledTime.Add(-scheduledTimeTolerance), e.ScheduledTime
	default:
		// 假定一定检测过了，所以这里随便返回一个就可以
		now := time.Now()
		return now, now
	}
}

func (e SendStrategyConfig) Validate() error {
	// 校验策略相关字段
	switch e.Type {
	case SendStrategyImmediate:
		return nil
	case SendStrategyDelayed:
		if e.Delay <= 0 {
			return fmt.Errorf("%w: 延迟发送策略需要指定正数的延迟秒数", errs.ErrInvalidParameter)
		}
	case SendStrategyScheduled:
		if e.ScheduledTime.IsZero() || e.ScheduledTime.Before(time.Now()) {
			return fmt.Errorf("%w: 定时发送策略需要指定未来的发送时间", errs.ErrInvalidParameter)
		}
	case SendStrategyTimeWindow:
		if e.StartTime.IsZero() || e.StartTime.After(e.EndTime) {
			return fmt.Errorf("%w: 时间窗口发送策略需要指定有效的开始和结束时间", errs.ErrInvalidParameter)
		}
	case SendStrategyDeadline:
		if e.DeadlineTime.IsZero() || e.DeadlineTime.Before(time.Now()) {
			return fmt.Errorf("%w: 截止日期发送策略需要指定未来的发送时间", errs.ErrInvalidParameter)
		}
	}
	return nil
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
