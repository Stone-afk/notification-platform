package notification

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
)

// notificationService 通知服务实现
type notificationService struct {
	repo repository.NotificationRepository
}

// FindReadyNotifications 准备好调度发送的通知
func (s *notificationService) FindReadyNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error) {
	return s.repo.FindReadyNotifications(ctx, offset, limit)
}

// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
func (s *notificationService) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: 业务内唯一标识列表空", errs.ErrInvalidParameter)
	}

	notifications, err := s.repo.GetByKeys(ctx, bizID, keys...)
	if err != nil {
		return nil, fmt.Errorf("获取通知列表失败: %w", err)
	}
	return notifications, nil
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(repo repository.NotificationRepository) Service {
	return &notificationService{
		repo: repo,
	}
}
