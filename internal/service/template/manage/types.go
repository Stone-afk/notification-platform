package manage

import (
	"context"
	"notification-platform/internal/domain"
)

// ChannelTemplateService 提供模板管理的服务接口
//
//go:generate mockgen -source=./manage.go -destination=../mocks/manage.mock.go -package=templatemocks -typed ChannelTemplateService
type ChannelTemplateService interface {
	// 模版相关方法

	// GetTemplatesByOwner 获取指定所有者的模板列表
	GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error)

	// GetTemplateByIDAndProviderInfo 根据模板ID和供应商信息获取模板
	GetTemplateByIDAndProviderInfo(ctx context.Context, templateID int64, providerName string, channel domain.Channel) (domain.ChannelTemplate, error)

	// GetTemplateByID 根据ID获取模板
	GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error)

	// CreateTemplate 创建模板
	CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error)

	// UpdateTemplate 更新模板
	UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error

	// PublishTemplate 发布模板
	PublishTemplate(ctx context.Context, templateID, versionID int64) error

	// 模版版本相关方法

	// ForkVersion 基于已有版本创建模版版本
	ForkVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error)

	// UpdateVersion 更新模板版本
	UpdateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error

	// SubmitForInternalReview 提交内部审核
	SubmitForInternalReview(ctx context.Context, versionID int64) error

	// BatchUpdateVersionAuditStatus 批量更新版本审核状态
	BatchUpdateVersionAuditStatus(ctx context.Context, versions []domain.ChannelTemplateVersion) error

	// 供应商相关方法

	// BatchSubmitForProviderReview 批量提交供应商审核
	BatchSubmitForProviderReview(ctx context.Context, versionID []int64) error

	// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error)

	// BatchQueryAndUpdateProviderAuditInfo 批量查询并更新供应商审核信息
	BatchQueryAndUpdateProviderAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error
}
