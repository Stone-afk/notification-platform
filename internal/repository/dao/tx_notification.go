package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"strings"
	"time"
)

var (
	ErrDuplicatedTx       = errors.New("duplicated tx")
	ErrUpdateStatusFailed = errors.New("没有更新")
)

type txNotificationDAO struct {
	db *egorm.Component
}

func (dao *txNotificationDAO) FindCheckBack(ctx context.Context, offset, limit int) ([]TxNotification, error) {
	var notifications []TxNotification
	currentTime := time.Now().UnixMilli()

	err := dao.db.WithContext(ctx).
		Where("status = ? AND next_check_time <= ? AND next_check_time > 0", domain.TxNotificationStatusPrepare, currentTime).
		Offset(offset).
		Limit(limit).
		Order("next_check_time").
		Find(&notifications).Error

	return notifications, err
}

func (dao *txNotificationDAO) UpdateCheckStatus(ctx context.Context, txNotifications []TxNotification, status domain.SendStatus) error {
	sqls := make([]string, 0, len(txNotifications))
	now := time.Now().UnixMilli()
	notificationIDs := make([]uint64, 0, len(txNotifications))
	for _, txNotification := range txNotifications {
		updateSQL := fmt.Sprintf("UPDATE `tx_notifications` set `status` = '%s',`utime` = %d ,`next_check_time` = %d,`check_count` = %d WHERE `key` = %s AND `biz_id` = %d AND `status` = 'PREPARE'", txNotification.Status, now, txNotification.NextCheckTime, txNotification.CheckCount, txNotification.Key, txNotification.BizID)
		sqls = append(sqls, updateSQL)
		notificationIDs = append(notificationIDs, txNotification.NotificationID)
	}
	// 拼接所有SQL并执行
	// UPDATE xxx; UPDATE xxx;UPDATE xxx;
	if len(sqls) > 0 {
		return dao.db.Transaction(func(tx *gorm.DB) error {
			combinedSQL := strings.Join(sqls, "; ")
			err := tx.WithContext(ctx).Exec(combinedSQL).Error
			if err != nil {
				return err
			}
			if status != domain.SendStatusPrepare {
				return tx.WithContext(ctx).Model(&Notification{}).Where("id in ?", notificationIDs).
					Update("status", status).Error
			}
			return nil
		})
	}
	return nil
}

func (dao *txNotificationDAO) First(ctx context.Context, txID int64) (TxNotification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *txNotificationDAO) BatchGetTxNotification(ctx context.Context, txIDs []int64) (map[int64]TxNotification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *txNotificationDAO) GetByBizIDKey(ctx context.Context, bizID int64, key string) (TxNotification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *txNotificationDAO) UpdateNotificationID(ctx context.Context, bizID int64, key string, notificationID uint64) error {
	//TODO implement me
	panic("implement me")
}

func (dao *txNotificationDAO) Prepare(ctx context.Context, txNotification TxNotification, notification Notification) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *txNotificationDAO) UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error {
	//TODO implement me
	panic("implement me")
}

// NewTxNotificationDAO creates a new instance of TxNotificationDAO
func NewTxNotificationDAO(db *egorm.Component) TxNotificationDAO {
	return &txNotificationDAO{
		db: db,
	}
}

type TxNotification struct {
	// 事务id
	TxID int64  `gorm:"column:tx_id;autoIncrement;primaryKey"`
	Key  string `gorm:"type:VARCHAR(256);NOT NULL;uniqueIndex:idx_biz_id_key,priority:2;comment:'业务内唯一标识，区分同一个业务内的不同通知'"`
	// 创建的通知id
	NotificationID uint64 `gorm:"column:notification_id"`
	// 业务方唯一标识
	BizID int64 `gorm:"column:biz_id;type:bigint;not null;uniqueIndex:idx_biz_id_key"`
	// 通知状态
	Status string `gorm:"column:status;type:varchar(20);not null;default:'PREPARE';index:idx_next_check_time_status"`
	// 第几次检查从1开始
	CheckCount int `gorm:"column:check_count;type:int;not null;default:1"`
	// 下一次的回查时间戳
	NextCheckTime int64 `gorm:"column:next_check_time;type:bigint;not null;default:0;index:idx_next_check_time_status"`
	// 创建时间
	Ctime int64 `gorm:"column:ctime;type:bigint;not null"`
	// 更新时间
	Utime int64 `gorm:"column:utime;type:bigint;not null"`
}

// TableName specifies the table name for the TxNotification model
func (t *TxNotification) TableName() string {
	return "tx_notification"
}
