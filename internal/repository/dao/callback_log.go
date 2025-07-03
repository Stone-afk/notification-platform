package dao

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
