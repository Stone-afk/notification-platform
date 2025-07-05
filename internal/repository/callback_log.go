package repository

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
)

type callbackLogRepository struct {
	notificationRepo NotificationRepository
	dao              dao.CallbackLogDAO
}

func (repo *callbackLogRepository) Find(ctx context.Context, startTime, batchSize, startID int64) (logs []domain.CallbackLog, nextStartID int64, err error) {
	entities, nextStartID, err := repo.dao.Find(ctx, startTime, batchSize, startID)
	if err != nil {
		return nil, 0, err
	}

	if int64(len(entities)) < batchSize {
		nextStartID = 0
	}

	return slice.Map(entities, func(_ int, src dao.CallbackLog) domain.CallbackLog {
		n, _ := repo.notificationRepo.GetByID(ctx, src.NotificationID)
		return repo.toDomain(src, n)
	}), nextStartID, nil
}

func (repo *callbackLogRepository) Update(ctx context.Context, logs []domain.CallbackLog) error {
	return repo.dao.Update(ctx, slice.Map(logs, func(_ int, src domain.CallbackLog) dao.CallbackLog {
		return repo.toEntity(src)
	}))
}

func (repo *callbackLogRepository) FindByNotificationIDs(ctx context.Context, notificationIDs []uint64) ([]domain.CallbackLog, error) {
	logs, err := repo.dao.FindByNotificationIDs(ctx, notificationIDs)
	if err != nil {
		return nil, err
	}
	ns, err := repo.notificationRepo.BatchGetByIDs(ctx, notificationIDs)
	if err != nil {
		return nil, err
	}
	return slice.Map(logs, func(_ int, src dao.CallbackLog) domain.CallbackLog {
		return repo.toDomain(src, ns[src.NotificationID])
	}), nil
}

func (repo *callbackLogRepository) toDomain(log dao.CallbackLog, notification domain.Notification) domain.CallbackLog {
	return domain.CallbackLog{
		ID:            log.ID,
		Notification:  notification,
		RetryCount:    log.RetryCount,
		NextRetryTime: log.NextRetryTime,
		Status:        domain.CallbackLogStatus(log.Status),
	}
}

func (repo *callbackLogRepository) toEntity(log domain.CallbackLog) dao.CallbackLog {
	return dao.CallbackLog{
		ID:             log.ID,
		NotificationID: log.Notification.ID,
		RetryCount:     log.RetryCount,
		NextRetryTime:  log.NextRetryTime,
		Status:         log.Status.String(),
	}
}

func NewCallbackLogRepository(
	notificationRepo NotificationRepository,
	dao dao.CallbackLogDAO,
) CallbackLogRepository {
	return &callbackLogRepository{
		notificationRepo: notificationRepo,
		dao:              dao,
	}
}
