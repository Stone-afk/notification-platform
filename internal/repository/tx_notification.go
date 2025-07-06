package repository

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
)

type txNotificationRepo struct {
	txdao dao.TxNotificationDAO
}

func (repo *txNotificationRepo) First(ctx context.Context, txID int64) (domain.TxNotification, error) {
	noti, err := repo.txdao.First(ctx, txID)
	if err != nil {
		return domain.TxNotification{}, err
	}
	return repo.toDomain(noti), nil
}

func (repo *txNotificationRepo) Create(ctx context.Context, txn domain.TxNotification) (uint64, error) {
	// 转换领域模型到DAO对象
	txnEntity := repo.toEntity(txn)
	notificationEntity := repo.toNotificationEntity(txn.Notification)
	// 调用DAO层创建记录
	return repo.txdao.Prepare(ctx, txnEntity, notificationEntity)
}

func (repo *txNotificationRepo) FindCheckBack(ctx context.Context, offset, limit int) ([]domain.TxNotification, error) {
	// 调用DAO层查询记录
	daoNotifications, err := repo.txdao.FindCheckBack(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	// 将DAO对象列表转换为领域模型列表
	result := make([]domain.TxNotification, 0, len(daoNotifications))
	for _, daoNotification := range daoNotifications {
		result = append(result, repo.toDomain(daoNotification))
	}
	return result, nil
}

func (repo *txNotificationRepo) UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error {
	// 直接调用DAO层更新状态
	return repo.txdao.UpdateStatus(ctx, bizID, key, status, notificationStatus)
}

func (repo *txNotificationRepo) UpdateCheckStatus(ctx context.Context, txNotifications []domain.TxNotification, notificationStatus domain.SendStatus) error {
	// 将领域模型列表转换为DAO对象列表
	daoNotifications := make([]dao.TxNotification, 0, len(txNotifications))
	for idx := range txNotifications {
		txNotification := txNotifications[idx]
		daoNotifications = append(daoNotifications, repo.toEntity(txNotification))
	}

	// 调用DAO层更新检查状态
	return repo.txdao.UpdateCheckStatus(ctx, daoNotifications, notificationStatus)
}

func (repo *txNotificationRepo) toNotificationEntity(notification domain.Notification) dao.Notification {
	templateParams, _ := notification.MarshalTemplateParams()
	receivers, _ := notification.MarshalReceivers()
	return dao.Notification{
		ID:                notification.ID,
		BizID:             notification.BizID,
		Key:               notification.Key,
		Receivers:         receivers,
		Channel:           notification.Channel.String(),
		TemplateID:        notification.Template.ID,
		TemplateVersionID: notification.Template.VersionID,
		TemplateParams:    templateParams,
		Status:            notification.Status.String(),
		ScheduledSTime:    notification.ScheduledSTime.UnixMilli(),
		ScheduledETime:    notification.ScheduledETime.UnixMilli(),
		Version:           notification.Version,
	}
}

// toDomain 将DAO对象转换为领域模型
func (repo *txNotificationRepo) toDomain(daoNotification dao.TxNotification) domain.TxNotification {
	return domain.TxNotification{
		TxID: daoNotification.TxID,
		Notification: domain.Notification{
			ID: daoNotification.NotificationID,
		},
		Key:           daoNotification.Key,
		BizID:         daoNotification.BizID,
		Status:        domain.TxNotificationStatus(daoNotification.Status),
		CheckCount:    daoNotification.CheckCount,
		NextCheckTime: daoNotification.NextCheckTime,
		Ctime:         daoNotification.Ctime,
		Utime:         daoNotification.Utime,
	}
}

// toDao 将领域模型转换为DAO对象
func (repo *txNotificationRepo) toEntity(domainNotification domain.TxNotification) dao.TxNotification {
	return dao.TxNotification{
		TxID:           domainNotification.TxID,
		Key:            domainNotification.Key,
		NotificationID: domainNotification.Notification.ID,
		BizID:          domainNotification.BizID,
		Status:         domainNotification.Status.String(),
		CheckCount:     domainNotification.CheckCount,
		NextCheckTime:  domainNotification.NextCheckTime,
		Ctime:          domainNotification.Ctime,
		Utime:          domainNotification.Utime,
	}
}

// NewTxNotificationRepository creates a new TxNotificationRepository instance
func NewTxNotificationRepository(txdao dao.TxNotificationDAO) TxNotificationRepository {
	return &txNotificationRepo{
		txdao: txdao,
	}
}
