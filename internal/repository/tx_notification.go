package repository

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
)

type txNotificationRepo struct {
	txdao dao.TxNotificationDAO
}

func (repo *txNotificationRepo) Create(ctx context.Context, notification domain.TxNotification) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *txNotificationRepo) FindCheckBack(ctx context.Context, offset, limit int) ([]domain.TxNotification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *txNotificationRepo) UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error {
	//TODO implement me
	panic("implement me")
}

func (repo *txNotificationRepo) UpdateCheckStatus(ctx context.Context, txNotifications []domain.TxNotification, notificationStatus domain.SendStatus) error {
	//TODO implement me
	panic("implement me")
}

// NewTxNotificationRepository creates a new TxNotificationRepository instance
func NewTxNotificationRepository(txdao dao.TxNotificationDAO) TxNotificationRepository {
	return &txNotificationRepo{
		txdao: txdao,
	}
}
