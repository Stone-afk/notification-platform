package repository

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
)

// channelTemplateRepository 实现了ChannelTemplateRepository接口，提供模板数据的存储实现
type channelTemplateRepository struct {
	dao dao.ChannelTemplateDAO
}

func (repo *channelTemplateRepository) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error) {
	// 获取模板列表
	templates, err := repo.dao.GetTemplatesByOwner(ctx, ownerID, ownerType.String())
	if err != nil {
		return nil, err
	}

	if len(templates) == 0 {
		return []domain.ChannelTemplate{}, nil
	}

	return repo.getTemplates(ctx, templates)
}

func (repo *channelTemplateRepository) getTemplates(ctx context.Context, templates []dao.ChannelTemplate) ([]domain.ChannelTemplate, error) {
	// 提取模板IDs
	templateIDs := make([]int64, len(templates))
	for i := range templates {
		templateIDs[i] = templates[i].ID
	}

	// 获取所有模板关联的版本
	versions, err := repo.dao.GetTemplateVersionsByTemplateIDs(ctx, templateIDs)
	if err != nil {
		return nil, err
	}

	// 提取版本IDs
	versionIDs := make([]int64, len(versions))
	for i := range versions {
		versionIDs[i] = versions[i].ID
	}

	// 获取所有版本关联的供应商
	providers, err := repo.dao.GetProvidersByVersionIDs(ctx, versionIDs)
	if err != nil {
		return nil, err
	}

	// 构建版本ID到供应商列表的映射
	versionToProviders := make(map[int64][]domain.ChannelTemplateProvider)
	for i := range providers {
		domainProvider := repo.toProviderDomain(providers[i])
		versionToProviders[providers[i].TemplateVersionID] = append(versionToProviders[providers[i].TemplateVersionID], domainProvider)
	}

	// 构建模板ID到版本列表的映射
	templateToVersions := make(map[int64][]domain.ChannelTemplateVersion)
	for i := range versions {
		domainVersion := repo.toVersionDomain(versions[i])
		// 添加版本关联的供应商
		domainVersion.Providers = versionToProviders[versions[i].ID]
		templateToVersions[versions[i].ChannelTemplateID] = append(templateToVersions[versions[i].ChannelTemplateID], domainVersion)
	}

	// 构建最终的领域模型列表
	result := make([]domain.ChannelTemplate, len(templates))
	for i, t := range templates {
		domainTemplate := repo.toTemplateDomain(t)
		// 添加模板关联的版本
		domainTemplate.Versions = templateToVersions[t.ID]
		result[i] = domainTemplate
	}

	return result, nil
}

func (repo *channelTemplateRepository) GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error) {
	templateEntity, err := repo.dao.GetTemplateByID(ctx, templateID)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}
	templates, err := repo.getTemplates(ctx, []dao.ChannelTemplate{templateEntity})
	if err != nil {
		return domain.ChannelTemplate{}, err
	}
	const first = 0
	return templates[first], nil
}

func (repo *channelTemplateRepository) CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error) {
	// 转换为数据库模型
	daoTemplate := repo.toTemplateEntity(template)

	// 创建模板
	createdTemplate, err := repo.dao.CreateTemplate(ctx, daoTemplate)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	// 转换回领域模型
	result := repo.toTemplateDomain(createdTemplate)
	return result, nil
}

func (repo *channelTemplateRepository) UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error {
	return repo.dao.UpdateTemplate(ctx, repo.toTemplateEntity(template))
}

func (repo *channelTemplateRepository) SetTemplateActiveVersion(ctx context.Context, templateID, versionID int64) error {
	return repo.dao.SetTemplateActiveVersion(ctx, templateID, versionID)
}

func (repo *channelTemplateRepository) GetTemplateVersionByID(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	version, err := repo.dao.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}
	providers, err := repo.dao.GetProvidersByVersionIDs(ctx, []int64{versionID})
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}
	domainProviders := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		domainProviders = append(domainProviders, repo.toProviderDomain(providers[i]))
	}

	domainVersion := repo.toVersionDomain(version)
	domainVersion.Providers = domainProviders
	return domainVersion, nil
}

// CreateTemplateVersion 创建模板版本
func (repo *channelTemplateRepository) CreateTemplateVersion(ctx context.Context, version domain.ChannelTemplateVersion) (domain.ChannelTemplateVersion, error) {
	// 转换为数据库模型
	daoVersion := repo.toVersionEntity(version)

	// 创建模板版本
	createdVersion, err := repo.dao.CreateTemplateVersion(ctx, daoVersion)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}

	// 转换回领域模型
	result := repo.toVersionDomain(createdVersion)
	return result, nil
}

func (repo *channelTemplateRepository) ForkTemplateVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	v, err := repo.dao.ForkTemplateVersion(ctx, versionID)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}

	version := repo.toVersionDomain(v)

	providers, err := repo.dao.GetProvidersByTemplateIDAndVersionID(ctx, v.ChannelTemplateID, v.ID)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}

	version.Providers = slice.Map(providers, func(_ int, src dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
		return repo.toProviderDomain(src)
	})

	return version, nil
}

// 供应商相关方法

func (repo *channelTemplateRepository) GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName string, channel domain.Channel) ([]domain.ChannelTemplateProvider, error) {
	providers, err := repo.dao.GetProviderByNameAndChannel(ctx, templateID, versionID, providerName, channel.String())
	if err != nil {
		return nil, err
	}
	results := make([]domain.ChannelTemplateProvider, len(providers))
	for i := range providers {
		results[i] = repo.toProviderDomain(providers[i])
	}
	return results, nil
}

// BatchCreateTemplateProviders 批量创建模板供应商关联
func (repo *channelTemplateRepository) BatchCreateTemplateProviders(ctx context.Context, providers []domain.ChannelTemplateProvider) ([]domain.ChannelTemplateProvider, error) {
	if len(providers) == 0 {
		return []domain.ChannelTemplateProvider{}, nil
	}

	// 转换为数据库模型
	daoProviders := slice.Map(providers, func(_ int, src domain.ChannelTemplateProvider) dao.ChannelTemplateProvider {
		return repo.toProviderEntity(src)
	})

	// 批量创建
	createdProviders, err := repo.dao.BatchCreateTemplateProviders(ctx, daoProviders)
	if err != nil {
		return nil, err
	}

	// 转换回领域模型
	return slice.Map(createdProviders, func(_ int, src dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
		return repo.toProviderDomain(src)
	}), nil
}

func (repo *channelTemplateRepository) GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error) {
	providers, err := repo.dao.GetApprovedProvidersByTemplateIDAndVersionID(ctx, templateID, versionID)
	if err != nil {
		return nil, err
	}
	return slice.Map(providers, func(_ int, src dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
		return repo.toProviderDomain(src)
	}), nil
}

func (repo *channelTemplateRepository) GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error) {
	providers, err := repo.dao.GetProvidersByTemplateIDAndVersionID(ctx, templateID, versionID)
	if err != nil {
		return nil, err
	}
	return slice.Map(providers, func(_ int, src dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
		return repo.toProviderDomain(src)
	}), nil
}

func (repo *channelTemplateRepository) UpdateTemplateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error {
	return repo.dao.UpdateTemplateVersion(ctx, repo.toVersionEntity(version))
}

func (repo *channelTemplateRepository) BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []domain.ChannelTemplateVersion) error {
	return repo.dao.BatchUpdateTemplateVersionAuditInfo(ctx, slice.Map(versions, func(_ int, src domain.ChannelTemplateVersion) dao.ChannelTemplateVersion {
		return repo.toVersionEntity(src)
	}))
}

func (repo *channelTemplateRepository) UpdateTemplateProviderAuditInfo(ctx context.Context, provider domain.ChannelTemplateProvider) error {
	return repo.dao.UpdateTemplateProviderAuditInfo(ctx, repo.toProviderEntity(provider))
}

func (repo *channelTemplateRepository) BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	daoProviders := slice.Map(providers, func(_ int, src domain.ChannelTemplateProvider) dao.ChannelTemplateProvider {
		return repo.toProviderEntity(src)
	})
	return repo.dao.BatchUpdateTemplateProvidersAuditInfo(ctx, daoProviders)
}

// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
func (repo *channelTemplateRepository) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error) {
	var daoProviders []dao.ChannelTemplateProvider

	daoProviders, err = repo.dao.GetPendingOrInReviewProviders(ctx, offset, limit, utime)
	if err != nil {
		return nil, 0, err
	}

	total, err = repo.dao.TotalPendingOrInReviewProviders(ctx, utime)
	if err != nil {
		return nil, 0, err
	}

	return slice.Map(daoProviders, func(_ int, src dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
		return repo.toProviderDomain(src)
	}), total, nil
}

func (repo *channelTemplateRepository) toTemplateDomain(daoTemplate dao.ChannelTemplate) domain.ChannelTemplate {
	return domain.ChannelTemplate{
		ID:              daoTemplate.ID,
		OwnerID:         daoTemplate.OwnerID,
		OwnerType:       domain.OwnerType(daoTemplate.OwnerType),
		Name:            daoTemplate.Name,
		Description:     daoTemplate.Description,
		Channel:         domain.Channel(daoTemplate.Channel),
		BusinessType:    domain.BusinessType(daoTemplate.BusinessType),
		ActiveVersionID: daoTemplate.ActiveVersionID,
		Ctime:           daoTemplate.Ctime,
		Utime:           daoTemplate.Utime,
	}
}

func (repo *channelTemplateRepository) toVersionDomain(daoVersion dao.ChannelTemplateVersion) domain.ChannelTemplateVersion {
	return domain.ChannelTemplateVersion{
		ID:                       daoVersion.ID,
		ChannelTemplateID:        daoVersion.ChannelTemplateID,
		Name:                     daoVersion.Name,
		Signature:                daoVersion.Signature,
		Content:                  daoVersion.Content,
		Remark:                   daoVersion.Remark,
		AuditID:                  daoVersion.AuditID,
		AuditorID:                daoVersion.AuditorID,
		AuditTime:                daoVersion.AuditTime,
		AuditStatus:              domain.AuditStatus(daoVersion.AuditStatus),
		RejectReason:             daoVersion.RejectReason,
		LastReviewSubmissionTime: daoVersion.LastReviewSubmissionTime,
		Ctime:                    daoVersion.Ctime,
		Utime:                    daoVersion.Utime,
	}
}

func (repo *channelTemplateRepository) toProviderDomain(daoProvider dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
	return domain.ChannelTemplateProvider{
		ID:                       daoProvider.ID,
		TemplateID:               daoProvider.TemplateID,
		TemplateVersionID:        daoProvider.TemplateVersionID,
		ProviderID:               daoProvider.ProviderID,
		ProviderName:             daoProvider.ProviderName,
		ProviderChannel:          domain.Channel(daoProvider.ProviderChannel),
		RequestID:                daoProvider.RequestID,
		ProviderTemplateID:       daoProvider.ProviderTemplateID,
		AuditStatus:              domain.AuditStatus(daoProvider.AuditStatus),
		RejectReason:             daoProvider.RejectReason,
		LastReviewSubmissionTime: daoProvider.LastReviewSubmissionTime,
		Ctime:                    daoProvider.Ctime,
		Utime:                    daoProvider.Utime,
	}
}

func (repo *channelTemplateRepository) toTemplateEntity(domainTemplate domain.ChannelTemplate) dao.ChannelTemplate {
	return dao.ChannelTemplate{
		ID:              domainTemplate.ID,
		OwnerID:         domainTemplate.OwnerID,
		OwnerType:       domainTemplate.OwnerType.String(),
		Name:            domainTemplate.Name,
		Description:     domainTemplate.Description,
		Channel:         domainTemplate.Channel.String(),
		BusinessType:    domainTemplate.BusinessType.ToInt64(),
		ActiveVersionID: domainTemplate.ActiveVersionID,
	}
}

func (repo *channelTemplateRepository) toVersionEntity(domainVersion domain.ChannelTemplateVersion) dao.ChannelTemplateVersion {
	return dao.ChannelTemplateVersion{
		ID:                       domainVersion.ID,
		ChannelTemplateID:        domainVersion.ChannelTemplateID,
		Name:                     domainVersion.Name,
		Signature:                domainVersion.Signature,
		Content:                  domainVersion.Content,
		Remark:                   domainVersion.Remark,
		AuditID:                  domainVersion.AuditID,
		AuditorID:                domainVersion.AuditorID,
		AuditTime:                domainVersion.AuditTime,
		AuditStatus:              domainVersion.AuditStatus.String(),
		RejectReason:             domainVersion.RejectReason,
		LastReviewSubmissionTime: domainVersion.LastReviewSubmissionTime,
	}
}

func (repo *channelTemplateRepository) toProviderEntity(domainProvider domain.ChannelTemplateProvider) dao.ChannelTemplateProvider {
	return dao.ChannelTemplateProvider{
		ID:                       domainProvider.ID,
		TemplateID:               domainProvider.TemplateID,
		TemplateVersionID:        domainProvider.TemplateVersionID,
		ProviderID:               domainProvider.ProviderID,
		ProviderName:             domainProvider.ProviderName,
		ProviderChannel:          domainProvider.ProviderChannel.String(),
		RequestID:                domainProvider.RequestID,
		ProviderTemplateID:       domainProvider.ProviderTemplateID,
		AuditStatus:              domainProvider.AuditStatus.String(),
		RejectReason:             domainProvider.RejectReason,
		LastReviewSubmissionTime: domainProvider.LastReviewSubmissionTime,
	}
}

// NewChannelTemplateRepository 创建仓储实例
func NewChannelTemplateRepository(dao dao.ChannelTemplateDAO) ChannelTemplateRepository {
	return &channelTemplateRepository{
		dao: dao,
	}
}
