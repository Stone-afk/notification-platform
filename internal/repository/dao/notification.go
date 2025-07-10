package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/ego-component/egorm"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"strings"
	"time"
)

type notificationDAO struct {
	db *egorm.Component

	coreDB     *egorm.Component
	noneCoreDB *egorm.Component
}

// isUniqueConstraintError 检查是否是唯一索引冲突错误
func (dao *notificationDAO) isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	me := new(mysql.MySQLError)
	if ok := errors.As(err, &me); ok {
		const uniqueIndexErrNo uint16 = 1062
		return me.Number == uniqueIndexErrNo
	}
	return false
}

//nolint:unused // 演示使用本地事务完成额度扣减
func (dao *notificationDAO) createV1(ctx context.Context, db *gorm.DB, data Notification, createCallbackLog bool) (Notification, error) {
	now := time.Now().UnixMilli()
	data.Ctime, data.Utime = now, now
	data.Version = 1

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&data).Error; err != nil {
			if dao.isUniqueConstraintError(err) {
				return fmt.Errorf("%w", errs.ErrNotificationDuplicate)
			}
			return err
		}
		// 直接数据库操作，直接扣减， 扣减1
		res := tx.Model(&Quota{}).
			Where("quota >= 1 AND biz_id = ? AND channel = ?", data.BizID, data.Channel).
			Updates(map[string]any{
				"quota": gorm.Expr("quota - 1"),
				"utime": now,
			})
		if res.Error != nil && res.RowsAffected > 0 {
			return fmt.Errorf("%w， 原因：%w", errs.ErrNoQuota, res.Error)
		}

		if createCallbackLog {
			if err := tx.Create(&CallbackLog{
				NotificationID: data.ID,
				Status:         domain.CallbackLogStatusInit.String(),
				NextRetryTime:  now,
			}).Error; err != nil {
				return fmt.Errorf("%w", errs.ErrCreateCallbackLogFailed)
			}
		}
		return nil
	})

	return data, err
}

func (dao *notificationDAO) create(ctx context.Context, db *gorm.DB, data Notification, createCallbackLog bool) (Notification, error) {
	now := time.Now().UnixMilli()
	data.Ctime, data.Utime = now, now
	data.Version = 1
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&data).Error; err != nil {
			if dao.isUniqueConstraintError(err) {
				return fmt.Errorf("%w", errs.ErrNotificationDuplicate)
			}
			return err
		}
		if createCallbackLog {
			if err := tx.Create(&CallbackLog{
				NotificationID: data.ID,
				Status:         domain.CallbackLogStatusInit.String(),
				NextRetryTime:  now,
			}).Error; err != nil {
				return fmt.Errorf("%w", errs.ErrCreateCallbackLogFailed)
			}
		}
		return nil
	})
	return data, err
}

// Create 创建单条通知记录，但不创建对应的回调记录
func (dao *notificationDAO) Create(ctx context.Context, data Notification) (Notification, error) {
	return dao.create(ctx, dao.db, data, false)
}

// CreateWithCallbackLog 创建单条通知记录，同时创建对应的回调记录
func (dao *notificationDAO) CreateWithCallbackLog(ctx context.Context, data Notification) (Notification, error) {
	return dao.create(ctx, dao.db, data, true)
}

// batchCreate 批量创建通知记录，以及可能的对应回调记录
func (dao *notificationDAO) batchCreate(ctx context.Context, datas []Notification, createCallbackLog bool) ([]Notification, error) {
	if len(datas) == 0 {
		return []Notification{}, nil
	}
	const batchSize = 100
	now := time.Now().UnixMilli()
	for i := range datas {
		datas[i].Ctime, datas[i].Utime = now, now
		datas[i].Version = 1
	}
	// 使用事务执行批量插入
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建通知记录 - 真正的批量插入
		if err := tx.CreateInBatches(datas, batchSize).Error; err != nil {
			if dao.isUniqueConstraintError(err) {
				return fmt.Errorf("%w", errs.ErrNotificationDuplicate)
			}
			return err
		}
		if createCallbackLog {
			// 创建回调记录
			var callbackLogs []CallbackLog
			for i := range datas {
				callbackLogs = append(callbackLogs, CallbackLog{
					NotificationID: datas[i].ID,
					NextRetryTime:  now,
					Ctime:          now,
					Utime:          now,
				})
			}
			if err := tx.CreateInBatches(callbackLogs, batchSize).Error; err != nil {
				return fmt.Errorf("%w", errs.ErrCreateCallbackLogFailed)
			}
		}
		return nil
	})
	return datas, err
}

// BatchCreate 批量创建通知记录，但不创建对应的回调记录
func (dao *notificationDAO) BatchCreate(ctx context.Context, notifications []Notification) ([]Notification, error) {
	return dao.batchCreate(ctx, notifications, false)
}

// BatchCreateWithCallbackLog 批量创建通知记录，同时创建对应的回调记录
func (dao *notificationDAO) BatchCreateWithCallbackLog(ctx context.Context, notifications []Notification) ([]Notification, error) {
	return dao.batchCreate(ctx, notifications, true)
}

// GetByID 根据ID查询通知
func (dao *notificationDAO) GetByID(ctx context.Context, id uint64) (Notification, error) {
	var notification Notification
	err := dao.db.WithContext(ctx).First(&notification, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Notification{}, fmt.Errorf("%w: id=%d", errs.ErrNotificationNotFound, id)
		}
		return Notification{}, err
	}
	return notification, nil
}

func (dao *notificationDAO) BatchGetByIDs(ctx context.Context, ids []uint64) (map[uint64]Notification, error) {
	var notifications []Notification
	err := dao.db.WithContext(ctx).
		Where("id in (?)", ids).
		Find(&notifications).Error
	notificationMap := make(map[uint64]Notification, len(ids))
	for idx := range notifications {
		notification := notifications[idx]
		notificationMap[notification.ID] = notification
	}
	return notificationMap, err
}

// GetByKey 根据业务ID和业务内唯一标识获取通知
func (dao *notificationDAO) GetByKey(ctx context.Context, bizID int64, key string) (Notification, error) {
	var not Notification
	err := dao.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", bizID, key).First(&not).Error
	if err != nil {
		return Notification{}, fmt.Errorf("查询通知失败:bizID: %d, key %s %w", bizID, key, err)
	}
	return not, nil
}

// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
func (dao *notificationDAO) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]Notification, error) {
	var notifications []Notification
	err := dao.db.WithContext(ctx).Where("biz_id = ? AND `key` IN ?", bizID, keys).Find(&notifications).Error
	if err != nil {
		return nil, fmt.Errorf("查询通知列表失败: %w", err)
	}
	return notifications, nil
}

// CASStatus 更新通知状态
func (dao *notificationDAO) CASStatus(ctx context.Context, notification Notification) error {
	updates := map[string]any{
		"status":  notification.Status,
		"version": gorm.Expr("version + 1"),
		"utime":   time.Now().Unix(),
	}
	result := dao.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND version = ?", notification.ID, notification.Version).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected < 1 {
		return fmt.Errorf("并发竞争失败 %w, id %d", errs.ErrNotificationVersionMismatch, notification.ID)
	}
	return nil
}

func (dao *notificationDAO) UpdateStatus(ctx context.Context, notification Notification) error {
	return dao.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ?", notification.ID).
		Updates(map[string]any{
			"status":  notification.Status,
			"version": gorm.Expr("version + 1"),
			"utime":   time.Now().Unix(),
		}).Error
}

func (dao *notificationDAO) BatchUpdateStatusSucceededOrFailed(ctx context.Context, successNotifications, failedNotifications []Notification) error {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) FindReadyNotifications(ctx context.Context, offset, limit int) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) MarkSuccess(ctx context.Context, entity Notification) error {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) MarkFailed(ctx context.Context, entity Notification) error {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	//TODO implement me
	panic("implement me")
}

//nolint:unused // 这是我的演示代码
func (dao *notificationDAO) selectDB(ctx context.Context) *egorm.Component {
	if ctx.Value("Priority") == "high" {
		return dao.coreDB
	}
	return dao.noneCoreDB
}

func NewNotificationDAOV1(coreDB *egorm.Component,
	noneCoreDB *egorm.Component,
) NotificationDAO {
	return &notificationDAO{
		coreDB:     coreDB,
		noneCoreDB: noneCoreDB,
	}
}

// NewNotificationDAO 创建通知DAO实例
func NewNotificationDAO(db *egorm.Component) NotificationDAO {
	return &notificationDAO{
		db: db,
	}
}

// Notification 通知记录表
type Notification struct {
	ID                uint64 `gorm:"primaryKey;comment:'雪花算法ID'"`
	BizID             int64  `gorm:"type:BIGINT;NOT NULL;index:idx_biz_id_status,priority:1;uniqueIndex:idx_biz_id_key,priority:1;comment:'业务配表ID，业务方可能有多个业务每个业务配置不同'"`
	Key               string `gorm:"type:VARCHAR(256);NOT NULL;uniqueIndex:idx_biz_id_key,priority:2;comment:'业务内唯一标识，区分同一个业务内的不同通知'"`
	Receivers         string `gorm:"type:TEXT;NOT NULL;comment:'接收者(手机/邮箱/用户ID)，JSON数组'"`
	Channel           string `gorm:"type:ENUM('SMS','EMAIL','IN_APP');NOT NULL;comment:'发送渠道'"`
	TemplateID        int64  `gorm:"type:BIGINT;NOT NULL;comment:'模板ID'"`
	TemplateVersionID int64  `gorm:"type:BIGINT;NOT NULL;comment:'模板版本ID'"`
	TemplateParams    string `gorm:"NOT NULL;comment:'模版参数'"`
	Status            string `gorm:"type:ENUM('PREPARE','CANCELED','PENDING','SENDING','SUCCEEDED','FAILED');DEFAULT:'PENDING';index:idx_biz_id_status,priority:2;index:idx_scheduled,priority:3;comment:'发送状态'"`
	ScheduledSTime    int64  `gorm:"column:scheduled_stime;index:idx_scheduled,priority:1;comment:'计划发送开始时间'"`
	ScheduledETime    int64  `gorm:"column:scheduled_etime;index:idx_scheduled,priority:2;comment:'计划发送结束时间'"`
	Version           int    `gorm:"type:INT;NOT NULL;DEFAULT:1;comment:'版本号，用于CAS操作'"`
	Ctime             int64
	Utime             int64
}

// TableName specifies the table name for the TxNotification model
func (t *Notification) TableName() string {
	return "notification"
}

// CheckErrIsIDDuplicate 判断是否是主键冲突
func CheckErrIsIDDuplicate(id uint64, err error) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("%d", id))
}
