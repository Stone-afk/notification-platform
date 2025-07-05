package dao

import (
	"context"
	"errors"
	"github.com/ego-component/egorm"
	"notification-platform/internal/domain"
)

var (
	ErrDuplicatedTx       = errors.New("duplicated tx")
	ErrUpdateStatusFailed = errors.New("没有更新")
)

type txNotificationDAO struct {
	db *egorm.Component
}

func (dao *txNotificationDAO) FindCheckBack(ctx context.Context, offset, limit int) ([]TxNotification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *txNotificationDAO) UpdateCheckStatus(ctx context.Context, txNotifications []TxNotification, status domain.SendStatus) error {
	//TODO implement me
	panic("implement me")
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
