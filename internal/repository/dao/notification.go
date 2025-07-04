package dao

import (
	"fmt"
	"strings"
)

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

// CheckErrIsIDDuplicate 判断是否是主键冲突
func CheckErrIsIDDuplicate(id uint64, err error) bool {
	return strings.Contains(err.Error(), fmt.Sprintf("%d", id))
}
