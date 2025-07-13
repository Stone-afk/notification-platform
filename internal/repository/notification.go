package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/cache"
	"notification-platform/internal/repository/dao"
	"time"
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

func (repo *notificationRepository) getItems(notifications []domain.Notification) []cache.IncrItem {
	notiMap := make(map[string]cache.IncrItem)
	for idx := range notifications {
		d := notifications[idx]
		key := fmt.Sprintf("%d-%s", d.BizID, d.Channel.String())
		item, ok := notiMap[key]
		if !ok {
			item = cache.IncrItem{
				BizID:   d.BizID,
				Channel: d.Channel,
			}
		}
		item.Val++
		notiMap[key] = item
	}
	items := make([]cache.IncrItem, 0, len(notiMap))
	for key := range notiMap {
		items = append(items, notiMap[key])
	}
	return items
}

func (repo *notificationRepository) mutiDecr(ctx context.Context, notifications []domain.Notification) error {
	return repo.quotaCache.MutiDecr(ctx, repo.getItems(notifications))
}

func (repo *notificationRepository) mutiIncr(ctx context.Context, notifications []domain.Notification) error {
	return repo.quotaCache.MutiIncr(ctx, repo.getItems(notifications))
}

// Create 创建单条通知记录，但不创建对应的回调记录
func (repo *notificationRepository) Create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	// 扣减额度
	err := repo.quotaCache.Decr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
	if err != nil {
		return domain.Notification{}, err
	}
	ds, err := repo.dao.Create(ctx, repo.toEntity(notification))
	if err != nil {
		// 创建没成功把额度还回去
		qerr := repo.quotaCache.Incr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
		if qerr != nil {
			repo.logger.Error("额度归还失败", elog.FieldErr(err),
				elog.Int64("biz_id", notification.BizID),
				elog.String("channel", notification.Channel.String()),
			)
		}
		return domain.Notification{}, err
	}
	return repo.toDomain(ds), nil
}

// CreateWithCallbackLog 创建单条通知记录，同时创建对应的回调记录
func (repo *notificationRepository) CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	// 扣减额度
	err := repo.quotaCache.Decr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
	if err != nil {
		return domain.Notification{}, err
	}
	ds, err := repo.dao.CreateWithCallbackLog(ctx, repo.toEntity(notification))
	if err != nil {
		qerr := repo.quotaCache.Incr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
		if qerr != nil {
			repo.logger.Error("额度归还失败", elog.FieldErr(err),
				elog.Int64("biz_id", notification.BizID),
				elog.String("channel", notification.Channel.String()),
			)
		}
		return domain.Notification{}, err
	}
	return repo.toDomain(ds), nil
}

// BatchCreate 批量创建通知记录，但不创建对应的回调记录
func (repo *notificationRepository) BatchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	return repo.batchCreate(ctx, notifications, false)
}

func (repo *notificationRepository) batchCreate(ctx context.Context, notifications []domain.Notification, createCallbackLog bool) ([]domain.Notification, error) {
	if len(notifications) == 0 {
		return nil, nil
	}

	daoNotifications := slice.Map(notifications, func(_ int, src domain.Notification) dao.Notification {
		return repo.toEntity(src)
	})

	var createdNotifications []dao.Notification
	var err error
	// 扣减库存
	err = repo.mutiDecr(ctx, notifications)
	if err != nil {
		return nil, err
	}
	if createCallbackLog {
		createdNotifications, err = repo.dao.BatchCreateWithCallbackLog(ctx, daoNotifications)
		if err != nil {
			eerr := repo.mutiIncr(ctx, notifications)
			if eerr != nil {
				elog.Error("发送失败，归还额度失败", elog.FieldErr(eerr))
			}
			return nil, err
		}
	} else {
		createdNotifications, err = repo.dao.BatchCreate(ctx, daoNotifications)
		if err != nil {
			eerr := repo.mutiIncr(ctx, notifications)
			if eerr != nil {
				elog.Error("发送失败，归还额度失败", elog.FieldErr(eerr))
			}
			return nil, err
		}
	}

	return slice.Map(createdNotifications, func(_ int, src dao.Notification) domain.Notification {
		return repo.toDomain(src)
	}), nil
}

// BatchCreateWithCallbackLog 批量创建通知记录，同时创建对应的回调记录
func (repo *notificationRepository) BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	return repo.batchCreate(ctx, notifications, true)
}

// GetByID 根据ID获取通知
func (repo *notificationRepository) GetByID(ctx context.Context, id uint64) (domain.Notification, error) {
	n, err := repo.dao.GetByID(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	return repo.toDomain(n), nil
}

func (repo *notificationRepository) BatchGetByIDs(ctx context.Context, ids []uint64) (map[uint64]domain.Notification, error) {
	notificationMap, err := repo.dao.BatchGetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	domainNotificationMap := make(map[uint64]domain.Notification, len(notificationMap))
	for id := range notificationMap {
		notification := notificationMap[id]
		domainNotificationMap[id] = repo.toDomain(notification)
	}
	return domainNotificationMap, nil
}

func (repo *notificationRepository) GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error) {
	not, err := repo.dao.GetByKey(ctx, bizID, key)
	return repo.toDomain(not), err
}

// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
func (repo *notificationRepository) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error) {
	notifications, err := repo.dao.GetByKeys(ctx, bizID, keys...)
	if err != nil {
		return nil, fmt.Errorf("查询通知列表失败: %w", err)
	}
	result := make([]domain.Notification, len(notifications))
	for i := range notifications {
		result[i] = repo.toDomain(notifications[i])
	}
	return result, nil
}

// CASStatus 更新通知状态
func (repo *notificationRepository) CASStatus(ctx context.Context, notification domain.Notification) error {
	return repo.dao.CASStatus(ctx, repo.toEntity(notification))
}

func (repo *notificationRepository) UpdateStatus(ctx context.Context, notification domain.Notification) error {
	return repo.dao.UpdateStatus(ctx, repo.toEntity(notification))
}

// BatchUpdateStatusSucceededOrFailed 批量更新通知状态为成功或失败
func (repo *notificationRepository) BatchUpdateStatusSucceededOrFailed(ctx context.Context, succeededNotifications, failedNotifications []domain.Notification) error {
	// 转换成功的通知为DAO层的实体
	successItems := slice.Map(succeededNotifications, func(_ int, src domain.Notification) dao.Notification {
		return repo.toEntity(src)
	})

	// 转换失败的通知为DAO层的实体
	failedItems := slice.Map(failedNotifications, func(_ int, src domain.Notification) dao.Notification {
		return repo.toEntity(src)
	})

	err := repo.dao.BatchUpdateStatusSucceededOrFailed(ctx, successItems, failedItems)
	if err != nil {
		return err
	}

	items := repo.getItems(failedNotifications)
	eerr := repo.quotaCache.MutiIncr(ctx, items)
	if eerr != nil {
		elog.Error("发送失败，归还额度失败", elog.FieldErr(eerr))
	}
	return nil
}

func (repo *notificationRepository) FindReadyNotifications(ctx context.Context, offset int, limit int) ([]domain.Notification, error) {
	nos, err := repo.dao.FindReadyNotifications(ctx, offset, limit)
	return slice.Map(nos, func(_ int, src dao.Notification) domain.Notification {
		return repo.toDomain(src)
	}), err
}

func (repo *notificationRepository) MarkSuccess(ctx context.Context, notification domain.Notification) error {
	return repo.dao.MarkSuccess(ctx, repo.toEntity(notification))
}

func (repo *notificationRepository) MarkFailed(ctx context.Context, notification domain.Notification) error {
	err := repo.dao.MarkFailed(ctx, repo.toEntity(notification))
	if err != nil {
		return err
	}
	return repo.quotaCache.Incr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
}

func (repo *notificationRepository) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	// TODO 不用 MutiIncr 归还额度？？？
	return repo.dao.MarkTimeoutSendingAsFailed(ctx, batchSize)
}

// toEntity 将领域对象转换为DAO实体
func (repo *notificationRepository) toEntity(notification domain.Notification) dao.Notification {
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

// toDomain 将DAO实体转换为领域对象
func (repo *notificationRepository) toDomain(n dao.Notification) domain.Notification {
	var templateParams map[string]string
	_ = json.Unmarshal([]byte(n.TemplateParams), &templateParams)

	var receivers []string
	_ = json.Unmarshal([]byte(n.Receivers), &receivers)

	return domain.Notification{
		ID:        n.ID,
		BizID:     n.BizID,
		Key:       n.Key,
		Receivers: receivers,
		Channel:   domain.Channel(n.Channel),
		Template: domain.Template{
			ID:        n.TemplateID,
			VersionID: n.TemplateVersionID,
			Params:    templateParams,
		},
		Status:         domain.SendStatus(n.Status),
		ScheduledSTime: time.UnixMilli(n.ScheduledSTime),
		ScheduledETime: time.UnixMilli(n.ScheduledETime),
		Version:        n.Version,
	}
}

// NewNotificationRepository 创建通知仓储实例
func NewNotificationRepository(d dao.NotificationDAO, quotaCache cache.QuotaCache) NotificationRepository {
	return &notificationRepository{
		dao:        d,
		quotaCache: quotaCache,
		logger:     elog.DefaultLogger,
	}
}
