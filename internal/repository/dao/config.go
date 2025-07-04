package dao

import (
	"context"
	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/sqlx"
	"time"
)

// Implementation of the BusinessConfigDAO interface
type businessConfigDAO struct {
	db *egorm.Component
}

// GetByIDs 根据ID获取业务配置信息
func (dao *businessConfigDAO) GetByIDs(ctx context.Context, ids []int64) (map[int64]BusinessConfig, error) {
	var configs []BusinessConfig
	// 根据ID查询业务配置
	err := dao.db.WithContext(ctx).Where("id in (?)", ids).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	configMap := make(map[int64]BusinessConfig, len(ids))
	for idx := range configs {
		config := configs[idx]
		configMap[config.ID] = config
	}
	return configMap, nil
}

func (dao *businessConfigDAO) GetByID(ctx context.Context, id int64) (BusinessConfig, error) {
	var config BusinessConfig

	// 根据ID查询业务配置
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		return BusinessConfig{}, err
	}

	return config, nil
}

func (dao *businessConfigDAO) Delete(ctx context.Context, id int64) error {
	// 执行删除操作
	result := dao.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&BusinessConfig{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// SaveConfig 保存业务配置
func (dao *businessConfigDAO) SaveConfig(ctx context.Context, config BusinessConfig) (BusinessConfig, error) {
	now := time.Now().UnixMilli()
	config.Ctime = now
	config.Utime = now
	// 使用upsert语句，如果记录存在则更新，不存在则插入
	db := dao.db.WithContext(ctx)
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // 根据ID判断冲突
		DoUpdates: clause.AssignmentColumns([]string{
			"owner_id",
			"owner_type",
			"channel_config",
			"txn_config",
			"rate_limit",
			"quota",
			"callback_config",
			"utime",
		}), // 只更新指定的非空列
	}).Create(&config)
	if result.Error != nil {
		return BusinessConfig{}, result.Error
	}
	return config, nil
}

func (dao *businessConfigDAO) Find(ctx context.Context, offset int, limit int) ([]BusinessConfig, error) {
	var res []BusinessConfig
	err := dao.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&res).Error
	return res, err
}

// NewBusinessConfigDAO 创建一个新的BusinessConfigDAO实例
func NewBusinessConfigDAO(db *egorm.Component) BusinessConfigDAO {
	return &businessConfigDAO{
		db: db,
	}
}

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
