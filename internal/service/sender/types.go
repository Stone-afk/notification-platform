package sender

import (
	"context"
	"notification-platform/internal/domain"
)

// NotificationSender 通知发送接口
//
//go:generate mockgen -source=./sender.go -destination=./mocks/sender.mock.go -package=sendermocks -typed NotificationSender
type NotificationSender interface {
	// Send 单条发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
	// BatchSend 发送批量通知
	BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error)
}
