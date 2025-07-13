package notification

import (
	"context"
	"github.com/gotomicro/ego/core/elog"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/config"
	"notification-platform/internal/service/sender"
	"time"
)

const defaultBatchSize = 10

type txNotificationService struct {
	repo      repository.TxNotificationRepository
	notiRepo  repository.NotificationRepository
	configSvc config.BusinessConfigService
	logger    *elog.Component
	lock      dlock.Client
	sender    sender.NotificationSender
}

func (s *txNotificationService) Prepare(ctx context.Context, notification domain.Notification) (uint64, error) {
	// TODO
	notification.Status = domain.SendStatusPrepare
	notification.SetSendTime()
	txn := domain.TxNotification{
		Notification: notification,
		Key:          notification.Key,
		BizID:        notification.BizID,
		Status:       domain.TxNotificationStatusPrepare,
	}

	cfg, err := s.configSvc.GetByID(ctx, notification.BizID)
	if err == nil {
		now := time.Now().UnixMilli()
		const second = 1000
		if cfg.TxnConfig != nil {
			txn.NextCheckTime = now + int64(cfg.TxnConfig.InitialDelay*second)
		}
	}
	return s.repo.Create(ctx, txn)
}

func (s *txNotificationService) Commit(ctx context.Context, bizID int64, key string) error {
	err := s.repo.UpdateStatus(ctx, bizID, key, domain.TxNotificationStatusCommit, domain.SendStatusPending)
	if err != nil {
		return err
	}
	notification, err := s.notiRepo.GetByKey(ctx, bizID, key)
	if err != nil {
		return err
	}
	if notification.IsImmediate() {
		_, err = s.sender.Send(ctx, notification)
	}
	return err
}

func (s *txNotificationService) Cancel(ctx context.Context, bizID int64, key string) error {
	return s.repo.UpdateStatus(ctx, bizID, key, domain.TxNotificationStatusCancel, domain.SendStatusCanceled)
}

func NewTxNotificationService(
	repo repository.TxNotificationRepository,
	configSvc config.BusinessConfigService,
	notiRepo repository.NotificationRepository,
	lock dlock.Client,
	sender sender.NotificationSender,
) TxNotificationService {
	return &txNotificationService{
		repo:      repo,
		configSvc: configSvc,
		logger:    elog.DefaultLogger,
		notiRepo:  notiRepo,
		lock:      lock,
		sender:    sender,
	}
}
