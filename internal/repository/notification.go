package repository

import (
	"context"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/cache"
	"notification-platform/internal/repository/dao"
)

const (
	defaultQuotaNumber int32 = 1
)

// notificationRepository 通知仓储实现
type notificationRepository struct {
	dao        dao.NotificationDAO
	quotaCache cache.QuotaCache
	logger     *elog.Component
}

func (repo *notificationRepository) Create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) BatchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) GetByID(ctx context.Context, id uint64) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) BatchGetByIDs(ctx context.Context, ids []uint64) (map[uint64]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) CASStatus(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) UpdateStatus(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) BatchUpdateStatusSucceededOrFailed(ctx context.Context, succeededNotifications, failedNotifications []domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) FindReadyNotifications(ctx context.Context, offset int, limit int) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) MarkSuccess(ctx context.Context, entity domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) MarkFailed(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (repo *notificationRepository) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	//TODO implement me
	panic("implement me")
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(d dao.NotificationDAO, quotaCache cache.QuotaCache) NotificationRepository {
	return &notificationRepository{
		dao:        d,
		quotaCache: quotaCache,
		logger:     elog.DefaultLogger,
	}
}
