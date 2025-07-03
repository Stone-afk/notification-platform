package channel

import (
	"context"
	"notification-platform/internal/domain"
)

//go:generate mockgen -source=./channel.go -destination=./mocks/channel.mock.go -package=channelmocks -typed Channel
type Channel interface {
	// Send 发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
}
