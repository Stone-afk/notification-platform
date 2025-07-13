package notification

import (
	"context"
	"notification-platform/internal/domain"
)

type ServiceDispatcher struct {
	v1 NotificationService
	v2 NotificationService
}

func (s *ServiceDispatcher) FindReadyNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error) {
	if ctx.Value("version") == "v2" {
		return s.v2.FindReadyNotifications(ctx, offset, limit)
	}
	return s.v1.FindReadyNotifications(ctx, offset, limit)
}

func (s *ServiceDispatcher) GetByKeys(_ context.Context, _ int64, _ ...string) ([]domain.Notification, error) {
	// TODO implement me
	panic("implement me")
}
