package callback

import (
	"context"
	"notification-platform/internal/domain"
)

type CallbackService interface {
	SendCallback(ctx context.Context, startTime, batchSize int64) error
	SendCallbackByNotification(ctx context.Context, notification domain.Notification) error
	SendCallbackByNotifications(ctx context.Context, notifications []domain.Notification) error
}
