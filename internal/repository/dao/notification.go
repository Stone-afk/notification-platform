package dao

import (
	"context"
	"fmt"
	"github.com/ego-component/egorm"
	"strings"
)

type notificationDAO struct {
	db *egorm.Component

	coreDB     *egorm.Component
	noneCoreDB *egorm.Component
}

func (dao *notificationDAO) Create(ctx context.Context, data Notification) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) CreateWithCallbackLog(ctx context.Context, data Notification) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) BatchCreate(ctx context.Context, dataList []Notification) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) BatchCreateWithCallbackLog(ctx context.Context, datas []Notification) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) GetByID(ctx context.Context, id uint64) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) BatchGetByIDs(ctx context.Context, ids []uint64) (map[uint64]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) GetByKey(ctx context.Context, bizID int64, key string) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) CASStatus(ctx context.Context, notification Notification) error {
	//TODO implement me
	panic("implement me")
}

func (dao *notificationDAO) UpdateStatus(ctx context.Context, notification Notification) error {
	//TODO implement me
	panic("implement me")
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
