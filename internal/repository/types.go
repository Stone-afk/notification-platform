package repository

import (
	"context"
	"notification-platform/internal/domain"
)

// CallbackLogRepository 回调记录仓储接口
type CallbackLogRepository interface {
	Find(ctx context.Context, startTime, batchSize, startID int64) (logs []domain.CallbackLog, nextStartID int64, err error)
	Update(ctx context.Context, logs []domain.CallbackLog) error
	FindByNotificationIDs(ctx context.Context, notificationIDs []uint64) ([]domain.CallbackLog, error)
}

type BusinessConfigRepository interface {
	LoadCache(ctx context.Context) error
	GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error)
	Delete(ctx context.Context, id int64) error
	SaveConfig(ctx context.Context, config domain.BusinessConfig) error
	Find(ctx context.Context, offset, limit int) ([]domain.BusinessConfig, error)
}

// ProviderRepository 供应商仓储接口
type ProviderRepository interface {
	// Create 创建供应商
	Create(ctx context.Context, provider domain.Provider) (domain.Provider, error)
	// Update 更新供应商
	Update(ctx context.Context, provider domain.Provider) error
	// FindByID 根据ID查找供应商
	FindByID(ctx context.Context, id int64) (domain.Provider, error)
	// FindByChannel 查找指定渠道的所有供应商
	FindByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error)
}

// ChannelTemplateRepository 提供模板数据存储的仓库接口
type ChannelTemplateRepository interface {
	// 模版相关方法

	// GetTemplatesByOwner 获取指定所有者的模板列表
	GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error)

	// GetTemplateByID 根据ID获取模板
	GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error)

	// CreateTemplate 创建模板
	CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error)

	// UpdateTemplate 更新模板
	UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error

	// SetTemplateActiveVersion 设置模板的活跃版本
	SetTemplateActiveVersion(ctx context.Context, templateID, versionID int64) error

	// 模版版本相关方法

	// GetTemplateVersionByID 根据ID获取模板版本
	GetTemplateVersionByID(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error)

	// CreateTemplateVersion 创建模板版本
	CreateTemplateVersion(ctx context.Context, version domain.ChannelTemplateVersion) (domain.ChannelTemplateVersion, error)

	// ForkTemplateVersion 基于已有版本创建新版本
	ForkTemplateVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error)

	// 供应商相关方法

	// GetProviderByNameAndChannel 根据名称和渠道获取供应商
	GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName string, channel domain.Channel) ([]domain.ChannelTemplateProvider, error)

	// BatchCreateTemplateProviders 批量创建模板供应商关联
	BatchCreateTemplateProviders(ctx context.Context, providers []domain.ChannelTemplateProvider) ([]domain.ChannelTemplateProvider, error)

	// GetApprovedProvidersByTemplateIDAndVersionID 获取已审核通过的供应商列表
	GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error)

	// GetProvidersByTemplateIDAndVersionID 获取模板和版本关联的所有供应商
	GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error)

	// UpdateTemplateVersion 更新模板版本
	UpdateTemplateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error

	// BatchUpdateTemplateVersionAuditInfo 批量更新模板版本审核信息
	BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []domain.ChannelTemplateVersion) error

	// UpdateTemplateProviderAuditInfo 更新模板供应商审核信息
	UpdateTemplateProviderAuditInfo(ctx context.Context, provider domain.ChannelTemplateProvider) error

	// BatchUpdateTemplateProvidersAuditInfo 批量更新模板供应商审核信息
	BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error

	// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error)
}
