package dao

import "context"

type BusinessConfigDAO interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (BusinessConfig, error)
	Delete(ctx context.Context, id int64) error
	SaveConfig(ctx context.Context, config BusinessConfig) (BusinessConfig, error)
	Find(ctx context.Context, offset int, limit int) ([]BusinessConfig, error)
}

type CallbackLogDAO interface {
	Find(ctx context.Context, startTime, batchSize, startID int64) (logs []CallbackLog, nextStartID int64, err error)
	FindByNotificationIDs(ctx context.Context, notificationIDs []uint64) ([]CallbackLog, error)
	Update(ctx context.Context, logs []CallbackLog) error
}

type ProviderDAO interface {
	// Create 创建供应商
	Create(ctx context.Context, provider Provider) (Provider, error)
	// Update 更新供应商
	Update(ctx context.Context, provider Provider) error
	// FindByID 根据ID查找供应商
	FindByID(ctx context.Context, id int64) (Provider, error)
	// FindByChannel 查找指定渠道的所有供应商
	FindByChannel(ctx context.Context, channel string) ([]Provider, error)
}

// ChannelTemplateDAO 提供模板数据访问对象接口
type ChannelTemplateDAO interface {
	// 模版相关方法

	// GetTemplatesByOwner 获取指定所有者的模板列表
	GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType string) ([]ChannelTemplate, error)

	// GetTemplateByID 根据ID获取模板
	GetTemplateByID(ctx context.Context, id int64) (ChannelTemplate, error)

	// CreateTemplate 创建模板
	CreateTemplate(ctx context.Context, template ChannelTemplate) (ChannelTemplate, error)

	// UpdateTemplate 更新模板
	UpdateTemplate(ctx context.Context, template ChannelTemplate) error

	// SetTemplateActiveVersion 设置模板的活跃版本
	SetTemplateActiveVersion(ctx context.Context, templateID, versionID int64) error

	// 模版版本相关方法

	// GetTemplateVersionsByTemplateIDs 根据模板ID列表获取对应的版本列表
	GetTemplateVersionsByTemplateIDs(ctx context.Context, templateIDs []int64) ([]ChannelTemplateVersion, error)

	// GetTemplateVersionByID 根据ID获取模板版本
	GetTemplateVersionByID(ctx context.Context, versionID int64) (ChannelTemplateVersion, error)

	// CreateTemplateVersion 创建模板版本
	CreateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) (ChannelTemplateVersion, error)

	// ForkTemplateVersion 基于已有版本创建新版本
	ForkTemplateVersion(ctx context.Context, versionID int64) (ChannelTemplateVersion, error)

	// 供应商关联相关方法

	// GetProvidersByVersionIDs 根据版本ID列表获取供应商列表
	GetProvidersByVersionIDs(ctx context.Context, versionIDs []int64) ([]ChannelTemplateProvider, error)

	// GetProviderByNameAndChannel 根据名称和渠道获取供应商
	GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName string, channel string) ([]ChannelTemplateProvider, error)

	// BatchCreateTemplateProviders 批量创建模板供应商关联
	BatchCreateTemplateProviders(ctx context.Context, providers []ChannelTemplateProvider) ([]ChannelTemplateProvider, error)

	// GetApprovedProvidersByTemplateIDAndVersionID 获取已审核通过的供应商列表
	GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]ChannelTemplateProvider, error)

	// GetProvidersByTemplateIDAndVersionID 获取模板和版本关联的所有供应商
	GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]ChannelTemplateProvider, error)

	// UpdateTemplateVersion 更新模板版本信息
	UpdateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) error

	// BatchUpdateTemplateVersionAuditInfo 批量更新模板版本审核信息
	BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []ChannelTemplateVersion) error

	// UpdateTemplateProviderAuditInfo 更新模板供应商审核信息
	UpdateTemplateProviderAuditInfo(ctx context.Context, provider ChannelTemplateProvider) error

	// BatchUpdateTemplateProvidersAuditInfo 批量更新模板供应商审核信息
	BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []ChannelTemplateProvider) error

	// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) ([]ChannelTemplateProvider, error)

	// TotalPendingOrInReviewProviders 统计未审核或审核中的供应商关联总数
	TotalPendingOrInReviewProviders(ctx context.Context, utime int64) (int64, error)
}
