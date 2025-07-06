package sender

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit/pool"
	"github.com/gotomicro/ego/core/elog"
	"log"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/channel"
	configsvc "notification-platform/internal/service/config"
	"notification-platform/internal/service/notification/callback"
	"sync"
)

// sender 通知发送器实现
type sender struct {
	repo        repository.NotificationRepository
	configSvc   configsvc.BusinessConfigService
	callbackSvc callback.CallbackService
	channel     channel.Channel
	taskPool    pool.TaskPool

	logger *elog.Component
}

func (s *sender) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	resp := domain.SendResponse{
		NotificationID: notification.ID,
	}
	_, err := s.channel.Send(ctx, notification)
	if err != nil {
		s.logger.Error("发送失败 %w", elog.FieldErr(err))
		resp.Status = domain.SendStatusFailed
		notification.Status = domain.SendStatusFailed
		// 如果是 FAILED，你需要把 quota 加回去
		err = s.repo.MarkFailed(ctx, notification)
	} else {
		resp.Status = domain.SendStatusSucceeded
		notification.Status = domain.SendStatusSucceeded
		err = s.repo.MarkSuccess(ctx, notification)
	}

	// 更新发送状态
	if err != nil {
		return domain.SendResponse{}, err
	}

	// 得到准确的发送结果，发起回调，发送成功和失败都应该回调
	_ = s.callbackSvc.SendCallbackByNotification(ctx, notification)

	return resp, nil
}

func (s *sender) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, nil
	}

	// 并发发送通知
	var succeedMu, failedMu sync.Mutex
	var succeeds, faileds []domain.SendResponse

	var wg sync.WaitGroup
	wg.Add(len(notifications))
	for i := range notifications {
		n := notifications[i]
		err := s.taskPool.Submit(ctx, pool.TaskFunc(func(ctx context.Context) error {
			defer wg.Done()
			_, err := s.channel.Send(ctx, n)
			if err != nil {
				resp := domain.SendResponse{
					NotificationID: n.ID,
					Status:         domain.SendStatusFailed,
				}
				failedMu.Lock()
				faileds = append(faileds, resp)
				failedMu.Unlock()
			} else {
				resp := domain.SendResponse{
					NotificationID: n.ID,
					Status:         domain.SendStatusSucceeded,
				}
				succeedMu.Lock()
				succeeds = append(succeeds, resp)
				succeedMu.Unlock()
			}
			log.Printf("submit notification[%d] = %#v\n", i, n)
			return nil
		}))
		if err != nil {
			s.logger.Warn("提交任务到任务池失败",
				elog.FieldErr(err),
				elog.Any("notification", n),
			)
			return nil, fmt.Errorf("提交任务到任务池失败: %w", err)
		}
	}
	wg.Wait()

	// 获取通知信息，以便获取版本号
	allNotificationIDs := make([]uint64, 0, len(succeeds)+len(faileds))
	for _, succeed := range succeeds {
		allNotificationIDs = append(allNotificationIDs, succeed.NotificationID)
	}
	for _, failed := range faileds {
		allNotificationIDs = append(allNotificationIDs, failed.NotificationID)
	}

	// 获取所有通知的详细信息，包括版本号
	notificationsMap, err := s.repo.BatchGetByIDs(ctx, allNotificationIDs)
	if err != nil {
		s.logger.Warn("批量获取通知详情失败",
			elog.FieldErr(err),
			elog.Any("allNotificationIDs", allNotificationIDs),
		)
		return nil, fmt.Errorf("批量获取通知详情失败: %w", err)
	}

	succeedNotifications := s.getUpdatedNotifications(succeeds, notificationsMap)
	failedNotifications := s.getUpdatedNotifications(faileds, notificationsMap)

	// 更新发送状态
	err = s.batchUpdateStatus(ctx, succeedNotifications, failedNotifications)
	if err != nil {
		return nil, err
	}

	// 得到准确的发送结果，发起回调，发送成功和失败都应该回调
	_ = s.callbackSvc.SendCallbackByNotifications(ctx, append(succeedNotifications, failedNotifications...))

	// 合并结果并返回
	return append(succeeds, faileds...), nil

}

// getUpdatedNotifications 获取更新字段后的实体
func (s *sender) getUpdatedNotifications(responses []domain.SendResponse, notificationsMap map[uint64]domain.Notification) []domain.Notification {
	notifications := make([]domain.Notification, 0, len(responses))
	for i := range responses {
		if n, ok := notificationsMap[responses[i].NotificationID]; ok {
			n.Status = responses[i].Status
			notifications = append(notifications, n)
		}
	}
	return notifications
}

// batchUpdateStatus 更新发送状态
func (s *sender) batchUpdateStatus(ctx context.Context, succeedNotifications, failedNotifications []domain.Notification) error {
	if len(succeedNotifications) > 0 || len(failedNotifications) > 0 {
		err := s.repo.BatchUpdateStatusSucceededOrFailed(ctx, succeedNotifications, failedNotifications)
		if err != nil {
			s.logger.Warn("批量更新通知状态失败",
				elog.Any("Error", err),
				elog.Any("succeedNotifications", succeedNotifications),
				elog.Any("failedNotifications", failedNotifications),
			)
			return fmt.Errorf("批量更新通知状态失败: %w", err)
		}
	}
	return nil
}

// NewSender 创建通知发送器
func NewSender(
	repo repository.NotificationRepository,
	configSvc configsvc.BusinessConfigService,
	callbackSvc callback.CallbackService,
	channel channel.Channel,
	taskPool pool.TaskPool,
) NotificationSender {
	return &sender{
		repo:        repo,
		configSvc:   configSvc,
		callbackSvc: callbackSvc,
		channel:     channel,
		taskPool:    taskPool,
		logger:      elog.DefaultLogger,
	}
}
