package repository

import (
	"context"
	"notification-platform/internal/domain"
)

type QuotaRepository interface {
	CreateOrUpdate(ctx context.Context, quota ...domain.Quota) error
	Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error)
}

type TxNotificationRepository interface {
	Create(ctx context.Context, notification domain.TxNotification) (uint64, error)
	FindCheckBack(ctx context.Context, offset, limit int) ([]domain.TxNotification, error)
	UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error
	UpdateCheckStatus(ctx context.Context, txNotifications []domain.TxNotification, notificationStatus domain.SendStatus) error
}

// NotificationRepository 通知仓储接口
type NotificationRepository interface {
	// Create 创建单条通知记录，但不创建对应的回调记录
	Create(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	// CreateWithCallbackLog 创建单条通知记录，同时创建对应的回调记录
	CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	// BatchCreate 批量创建通知记录，但不创建对应的回调记录
	BatchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error)
	// BatchCreateWithCallbackLog 批量创建通知记录，同时创建对应的回调记录
	BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error)

	// GetByID 根据ID获取通知
	GetByID(ctx context.Context, id uint64) (domain.Notification, error)
	// BatchGetByIDs 根据ID列表获取通知列表
	BatchGetByIDs(ctx context.Context, ids []uint64) (map[uint64]domain.Notification, error)

	GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error)
	// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
	GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error)

	// CASStatus 更新通知状态
	CASStatus(ctx context.Context, notification domain.Notification) error
	UpdateStatus(ctx context.Context, notification domain.Notification) error

	// BatchUpdateStatusSucceededOrFailed 批量更新通知状态为成功或失败
	BatchUpdateStatusSucceededOrFailed(ctx context.Context, succeededNotifications, failedNotifications []domain.Notification) error

	FindReadyNotifications(ctx context.Context, offset int, limit int) ([]domain.Notification, error)
	MarkSuccess(ctx context.Context, entity domain.Notification) error
	MarkFailed(ctx context.Context, notification domain.Notification) error
	// MarkTimeoutSendingAsFailed 将超时的 SENDING 状态的通知都标记为失败
	MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error)
}

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
