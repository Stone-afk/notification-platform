package dao

import (
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/sqlx"
)

// BusinessConfig 业务配置表
type BusinessConfig struct {
	ID             int64                                  `gorm:"primaryKey;type:BIGINT;comment:'业务标识'"`
	OwnerID        int64                                  `gorm:"type:BIGINT;comment:'业务方'"`
	OwnerType      string                                 `gorm:"type:ENUM('person', 'organization');comment:'业务方类型：person-个人,organization-组织'"`
	ChannelConfig  sqlx.JSONColumn[domain.ChannelConfig]  `gorm:"type:JSON;comment:'{\"channels\":[{\"channel\":\"SMS\", \"priority\":\"1\",\"enabled\":\"true\"},{\"channel\":\"EMAIL\", \"priority\":\"2\",\"enabled\":\"true\"}]}'"`
	TxnConfig      sqlx.JSONColumn[domain.TxnConfig]      `gorm:"type:JSON;comment:'事务配置'"`
	RateLimit      int                                    `gorm:"type:INT;DEFAULT:1000;comment:'每秒最大请求数'"`
	Quota          sqlx.JSONColumn[domain.QuotaConfig]    `gorm:"type:JSON;comment:'{\"monthly\":{\"SMS\":100000,\"EMAIL\":500000}}'"`
	CallbackConfig sqlx.JSONColumn[domain.CallbackConfig] `gorm:"type:JSON;comment:'回调配置，通知平台回调业务方通知异步请求结果'"`
	Ctime          int64
	Utime          int64
}

// TableName 重命名表
func (BusinessConfig) TableName() string {
	return "business_config"
}
