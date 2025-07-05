package sendstrategy

import (
	"context"
	"notification-platform/internal/domain"
)

// SendStrategy 发送策略接口
//
//go:generate mockgen -source=./types.go -destination=./mocks/send_strategy.mock.go -package=sendstrategymocks -typed SendStrategy
type SendStrategy interface {
	// Send 单条发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
	// BatchSend 批量发送通知，其中每个通知的发送策略必须相同
	BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error)
}
