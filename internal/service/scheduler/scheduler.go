package scheduler

import (
	"context"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/pkg/loopjob"
	notificationsvc "notification-platform/internal/service/notification"
	"notification-platform/internal/service/sender"
	"time"
)

// staticScheduler 通知调度服务实现
type staticScheduler struct {
	notificationSvc notificationsvc.NotificationService
	sender          sender.NotificationSender
	dclient         dlock.Client

	batchSize int
}

func (s *staticScheduler) Start(ctx context.Context) {
	const key = "notification_platform_async_scheduler"
	lj := loopjob.NewInfiniteLoop(s.dclient, s.processPendingNotifications, key)
	lj.Run(ctx)
}

// processPendingNotifications 处理待发送的通知
func (s *staticScheduler) processPendingNotifications(ctx context.Context) error {
	const defaultTimeout = 3 * time.Second
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	const offset = 0
	notifications, err := s.notificationSvc.FindReadyNotifications(ctx, offset, s.batchSize)
	if err != nil {
		return err
	}
	if len(notifications) == 0 {
		time.Sleep(time.Second)
		return nil
	}
	_, err = s.sender.BatchSend(ctx, notifications)
	return err
}

// NewScheduler 创建通知调度服务
func NewScheduler(
	notificationSvc notificationsvc.NotificationService,
	dispatcher sender.NotificationSender,
	dclient dlock.Client,
) NotificationScheduler {
	const defaultBatchSize = 10
	return &staticScheduler{
		notificationSvc: notificationSvc,
		sender:          dispatcher,
		batchSize:       defaultBatchSize,
		dclient:         dclient,
	}
}
