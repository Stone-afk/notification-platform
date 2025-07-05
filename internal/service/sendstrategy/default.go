package sendstrategy

import (
	"context"
	"fmt"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
	configsvc "notification-platform/internal/service/config"
)

var _ SendStrategy = (*DefaultSendStrategy)(nil)

// DefaultSendStrategy 延迟发送策略
type DefaultSendStrategy struct {
	repo      repository.NotificationRepository
	configSvc configsvc.BusinessConfigService
	logger    *elog.Component
}

func (s *DefaultSendStrategy) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	notification.SetSendTime()
	// 创建通知记录
	created, err := s.create(ctx, notification)
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("创建延迟通知失败: %w", err)
	}

	return domain.SendResponse{
		NotificationID: created.ID,
		Status:         created.Status,
	}, nil
}

func (s *DefaultSendStrategy) create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	if s.needCreateCallbackLog(ctx, notification) {
		return s.repo.CreateWithCallbackLog(ctx, notification)
	}
	return s.repo.Create(ctx, notification)
}

func (s *DefaultSendStrategy) needCreateCallbackLog(ctx context.Context, notification domain.Notification) bool {
	bizConfig, err := s.configSvc.GetByID(ctx, notification.BizID)
	if err != nil {
		s.logger.Error("查找 biz config 失败", elog.FieldErr(err))
		return false
	}
	return bizConfig.CallbackConfig != nil
}

func (s *DefaultSendStrategy) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}

	for i := range notifications {
		notifications[i].SetSendTime()
	}

	// 创建通知记录
	createdNotifications, err := s.batchCreate(ctx, notifications)
	if err != nil {
		return nil, fmt.Errorf("创建延迟通知失败: %w", err)
	}

	// 仅创建通知记录，等待定时任务扫描发送
	responses := make([]domain.SendResponse, len(createdNotifications))
	for i := range createdNotifications {
		responses[i] = domain.SendResponse{
			NotificationID: createdNotifications[i].ID,
			Status:         createdNotifications[i].Status,
		}
	}
	return responses, nil
}

func (s *DefaultSendStrategy) batchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	const first = 0
	if s.needCreateCallbackLog(ctx, notifications[first]) {
		return s.repo.BatchCreateWithCallbackLog(ctx, notifications)
	}
	return s.repo.BatchCreate(ctx, notifications)
}

// NewDefaultStrategy 创建延迟发送策略
func NewDefaultStrategy(repo repository.NotificationRepository, configSvc configsvc.BusinessConfigService) *DefaultSendStrategy {
	return &DefaultSendStrategy{
		repo:      repo,
		configSvc: configSvc,
		logger:    elog.DefaultLogger,
	}
}
