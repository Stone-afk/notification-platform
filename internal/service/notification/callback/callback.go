package callback

import (
	"fmt"
	"github.com/ecodeclub/ekit/syncx"
	"github.com/gotomicro/ego/client/egrpc"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/net/context"
	clientv1 "notification-platform/api/proto/gen/client/v1"
	notificationv1 "notification-platform/api/proto/gen/notification/v1"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/pkg/grpcx"
	"notification-platform/internal/pkg/retry"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/config"
	"time"
)

var _ CallbackService = (*callbackService)(nil)

type callbackService struct {
	configSvc    config.BusinessConfigService
	bizID2Config syncx.Map[int64, *domain.CallbackConfig]
	clients      *grpcx.Clients[clientv1.CallbackServiceClient]
	repo         repository.CallbackLogRepository
	logger       *elog.Component
}

func (s *callbackService) SendCallback(ctx context.Context, startTime, batchSize int64) error {
	var nextStartID int64
	for {
		// 查询需要回调的通知
		logs, newNextStartID, err := s.repo.Find(ctx, startTime, batchSize, nextStartID)
		if err != nil {
			s.logger.Error("查询回调日志失败",
				elog.FieldKey("startTime"),
				elog.FieldValueAny(startTime),
				elog.FieldKey("batchSize"),
				elog.FieldValueAny(batchSize),
				elog.FieldKey("nextStartID"),
				elog.FieldValueAny(nextStartID),
				elog.FieldErr(err))
			return err
		}

		if len(logs) == 0 {
			break
		}

		// 处理当前批次通知
		err = s.sendCallbackAndUpdateCallbackLogs(ctx, logs)
		if err != nil {
			return err
		}

		nextStartID = newNextStartID
	}
	return nil
}

func (s *callbackService) sendCallbackAndUpdateCallbackLogs(ctx context.Context, logs []domain.CallbackLog) error {
	needUpdate := make([]domain.CallbackLog, 0, len(logs))
	for i := range logs {
		changed, err := s.sendCallbackAndSetChangedFields(ctx, &logs[i])
		if err != nil {
			s.logger.Warn("业务方回调失败",
				elog.FieldKey("Callback.ID"),
				elog.FieldValueAny(logs[i].ID),
				elog.FieldErr(err))
			continue
		}
		if changed {
			needUpdate = append(needUpdate, logs[i])
		}
	}
	return s.repo.Update(ctx, needUpdate)
}

func (s *callbackService) sendCallbackAndSetChangedFields(ctx context.Context, log *domain.CallbackLog) (bool, error) {
	resp, err := s.sendCallback(ctx, log.Notification)
	if err != nil {
		return false, err
	}

	// 拿到业务方对回调的处理结果
	if resp.Success {
		log.Status = domain.CallbackLogStatusSuccess
		return true, nil
	}

	// 业务方对回调的处理失败，需要重试，此时业务方必定有配置
	cfg, _ := s.getConfig(ctx, log.Notification.BizID)
	retryStrategy, _ := retry.NewRetry(*cfg.RetryPolicy)
	interval, ok := retryStrategy.NextWithRetries(log.RetryCount)
	if ok {
		// 未达到最大重试次数，状态不变但要更新下次重试时间和重试次数
		log.NextRetryTime = time.Now().Add(interval).UnixMilli()
		log.RetryCount++
	} else {
		// 达到最大重试次数限制，不再重试，更新状态为失败
		log.Status = domain.CallbackLogStatusFailed
	}
	return true, nil
}

func (s *callbackService) sendCallback(ctx context.Context, notification domain.Notification) (*clientv1.HandleNotificationResultResponse, error) {
	cfg, err := s.getConfig(ctx, notification.BizID)
	if err != nil {
		s.logger.Warn("获取业务配置失败",
			elog.FieldKey("BizID"),
			elog.FieldValueAny(notification.BizID),
			elog.FieldErr(err))
		return nil, err
	}
	if cfg == nil {
		// 业务方未提供配置
		return nil, fmt.Errorf("%w", errs.ErrConfigNotFound)
	}
	return s.clients.Get(cfg.ServiceName).HandleNotificationResult(ctx, s.buildRequest(notification))
}

func (s *callbackService) buildRequest(notification domain.Notification) *clientv1.HandleNotificationResultRequest {
	templateParams := make(map[string]string)
	if notification.Template.Params != nil {
		templateParams = notification.Template.Params
	}
	return &clientv1.HandleNotificationResultRequest{
		NotificationId: notification.ID,
		OriginalRequest: &notificationv1.SendNotificationRequest{
			Notification: &notificationv1.Notification{
				Key:            notification.Key,
				Receivers:      notification.Receivers,
				Channel:        s.getChannel(notification),
				TemplateId:     fmt.Sprintf("%d", notification.Template.ID),
				TemplateParams: templateParams,
			},
		},
		Result: &notificationv1.SendNotificationResponse{
			NotificationId: notification.ID,
			Status:         s.getStatus(notification),
		},
	}
}

func (s *callbackService) getStatus(notification domain.Notification) notificationv1.SendStatus {
	var status notificationv1.SendStatus
	switch notification.Status {
	case domain.SendStatusSucceeded:
		status = notificationv1.SendStatus_SUCCEEDED
	case domain.SendStatusFailed:
		status = notificationv1.SendStatus_FAILED
	case domain.SendStatusPrepare:
		status = notificationv1.SendStatus_PREPARE
	case domain.SendStatusCanceled:
		status = notificationv1.SendStatus_CANCELED
	case domain.SendStatusPending:
		status = notificationv1.SendStatus_PENDING
	default:
		status = notificationv1.SendStatus_SEND_STATUS_UNSPECIFIED
	}
	return status
}

func (s *callbackService) getChannel(notification domain.Notification) notificationv1.Channel {
	var channel notificationv1.Channel
	switch notification.Channel {
	case domain.ChannelSMS:
		channel = notificationv1.Channel_SMS
	case domain.ChannelEmail:
		channel = notificationv1.Channel_EMAIL
	case domain.ChannelInApp:
		channel = notificationv1.Channel_IN_APP
	default:
		channel = notificationv1.Channel_CHANNEL_UNSPECIFIED
	}
	return channel
}

func (s *callbackService) getConfig(ctx context.Context, bizID int64) (*domain.CallbackConfig, error) {
	cfg, ok := s.bizID2Config.Load(bizID)
	if ok {
		return cfg, nil
	}
	bizConfig, err := s.configSvc.GetByID(ctx, bizID)
	if err != nil {
		return nil, err
	}
	if bizConfig.CallbackConfig != nil {
		s.bizID2Config.Store(bizID, bizConfig.CallbackConfig)
	}
	return bizConfig.CallbackConfig, nil
}

func (s *callbackService) SendCallbackByNotification(ctx context.Context, notification domain.Notification) error {
	logs, err := s.repo.FindByNotificationIDs(ctx, []uint64{notification.ID})
	if err != nil {
		return err
	}
	return s.sendCallbackAndUpdateCallbackLogs(ctx, logs)
}

func (s *callbackService) SendCallbackByNotifications(ctx context.Context, notifications []domain.Notification) error {
	notificationIDs := make([]uint64, 0, len(notifications))
	mp := make(map[uint64]domain.Notification, len(notifications))
	for i := range notifications {
		notificationIDs = append(notificationIDs, notifications[i].ID)
		mp[notifications[i].ID] = notifications[i]
	}

	logs, err := s.repo.FindByNotificationIDs(ctx, notificationIDs)
	if err != nil {
		// 一个失败整批次失败
		return err
	}
	if len(logs) == len(notifications) {
		// 全部有通知回调记录，非立即发送
		return s.sendCallbackAndUpdateCallbackLogs(ctx, logs)
	}

	for i := range logs {
		// 删除有回调记录的通知
		delete(mp, logs[i].Notification.ID)
	}

	var err1 error
	if len(logs) != 0 {
		// 部分有回调记录（调度器调度发送成功后触发）
		err1 = s.sendCallbackAndUpdateCallbackLogs(ctx, logs)
	}
	/*
		这通常发生在以下几种场景：
			该通知是同步立即发送，未落库；
			该通知是同步非立即发送，但回调未配置；
			该通知是调度器调度刚成功，尚未创建回调记录（即：发送完成，尚未执行回调创建）；
	*/
	for k := range mp {
		// 全部没有回调记录（同步立刻批量发送，或者同步非立刻发送同时没有回调配置）
		// 部分没有回调记录（调度器调度发送成功后触发）
		// Client上没有批量接口，这里可以考虑开协程
		_, err1 = s.sendCallback(ctx, mp[k])
	}
	return err1
}

func NewCallbackService(
	configSvc config.BusinessConfigService,
	repo repository.CallbackLogRepository,
) CallbackService {
	return &callbackService{
		configSvc:    configSvc,
		bizID2Config: syncx.Map[int64, *domain.CallbackConfig]{},
		repo:         repo,
		clients: grpcx.NewClients(func(conn *egrpc.Component) clientv1.CallbackServiceClient {
			return clientv1.NewCallbackServiceClient(conn)
		}),
		logger: elog.DefaultLogger.With(elog.FieldComponent("callback")),
	}
}
