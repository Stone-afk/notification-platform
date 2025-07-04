package dao

import (
	"context"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"time"
)

type callbackLogDAO struct {
	db *egorm.Component
}

func (dao *callbackLogDAO) Find(ctx context.Context, startTime, batchSize, startID int64) (logs []CallbackLog, nextStartID int64, err error) {
	nextStartID = 0

	result := dao.db.WithContext(ctx).Model(&CallbackLog{}).
		Where("next_retry_time <= ?", startTime).
		Where("status = ?", domain.CallbackLogStatusPending).
		Where("id > ?", startID).
		Order("id ASC").
		Limit(int(batchSize)).
		Find(&logs)

	if result.Error != nil {
		return logs, nextStartID, result.Error
	}

	if len(logs) > 0 {
		nextStartID = logs[len(logs)-1].ID
	}

	return logs, nextStartID, nil
}

func (dao *callbackLogDAO) FindByNotificationIDs(ctx context.Context, notificationIDs []uint64) ([]CallbackLog, error) {
	var logs []CallbackLog
	err := dao.db.WithContext(ctx).Where("notification_id IN (?)", notificationIDs).Find(&logs).Error
	return logs, err
}

func (dao *callbackLogDAO) Update(ctx context.Context, logs []CallbackLog) error {
	if len(logs) == 0 {
		return nil
	}
	utime := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, log := range logs {
			result := tx.Model(&CallbackLog{ID: log.ID}).
				Updates(map[string]any{
					"retry_count":     log.RetryCount,
					"next_retry_time": log.NextRetryTime,
					"status":          log.Status,
					"utime":           utime,
				})
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

func NewCallbackLogDAO(db *egorm.Component) CallbackLogDAO {
	return &callbackLogDAO{db: db}
}

// CallbackLog 只有同步立刻发送会缺乏这条记录
type CallbackLog struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;comment:'回调记录ID'"`
	NotificationID uint64 `gorm:"column:notification_id;NOT NULL;uniqueIndex:idx_notification_id;comment:'待回调通知ID'"`
	RetryCount     int32  `gorm:"type:TINYINT;NOT NULL;DEFAULT:0;comment:'重试次数'"`
	NextRetryTime  int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:0;comment:'下一次重试的时间戳'"`
	Status         string `gorm:"type:ENUM('INIT','PENDING','SUCCEEDED','FAILED');NOT NULL;DEFAULT:'INIT';index:idx_status;comment:'回调状态'"`
	Ctime          int64
	Utime          int64
}

// TableName 重命名表
func (CallbackLog) TableName() string {
	return "callback_log"
}
