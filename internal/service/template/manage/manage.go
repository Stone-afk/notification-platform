package manage

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/audit"
	providersvc "notification-platform/internal/service/provider/manage"
	"notification-platform/internal/service/provider/sms/client"
)

// templateService 实现了ChannelTemplateService接口，提供模板管理的具体实现
type templateService struct {
	repo        repository.ChannelTemplateRepository
	providerSvc providersvc.Service
	auditSvc    audit.Service
	smsClients  map[string]client.Client
}

func (s *templateService) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error) {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) GetTemplateByIDAndProviderInfo(ctx context.Context, templateID int64, providerName string, channel domain.Channel) (domain.ChannelTemplate, error) {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error) {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error) {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) PublishTemplate(ctx context.Context, templateID, versionID int64) error {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) ForkVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) UpdateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) SubmitForInternalReview(ctx context.Context, versionID int64) error {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) BatchUpdateVersionAuditStatus(ctx context.Context, versions []domain.ChannelTemplateVersion) error {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) BatchSubmitForProviderReview(ctx context.Context, versionID []int64) error {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *templateService) BatchQueryAndUpdateProviderAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	//TODO implement me
	panic("implement me")
}

// NewChannelTemplateService 创建模板服务实例
func NewChannelTemplateService(
	repo repository.ChannelTemplateRepository,
	providerSvc providersvc.Service,
	auditSvc audit.Service,
	smsClients map[string]client.Client,
) ChannelTemplateService {
	return &templateService{
		repo:        repo,
		providerSvc: providerSvc,
		auditSvc:    auditSvc,
		smsClients:  smsClients,
	}
}
