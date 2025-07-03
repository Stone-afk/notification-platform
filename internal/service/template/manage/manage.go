package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/audit"
	providersvc "notification-platform/internal/service/provider/manage"
	"notification-platform/internal/service/provider/sms/client"
	"regexp"
	"time"
)

// templateService 实现了ChannelTemplateService接口，提供模板管理的具体实现
type templateService struct {
	repo        repository.ChannelTemplateRepository
	providerSvc providersvc.Service
	auditSvc    audit.Service
	smsClients  map[string]client.Client
}

// 模版相关方法

func (s *templateService) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error) {
	if ownerID <= 0 {
		return nil, fmt.Errorf("%w: 业务方ID必须大于0", errs.ErrInvalidParameter)
	}

	if !ownerType.IsValid() {
		return nil, fmt.Errorf("%w: 所有者类型", errs.ErrInvalidParameter)
	}

	// 从仓储层获取模板列表
	templates, err := s.repo.GetTemplatesByOwner(ctx, ownerID, ownerType)
	if err != nil {
		return nil, fmt.Errorf("获取模板列表失败: %w", err)
	}

	return templates, nil
}

func (s *templateService) GetTemplateByIDAndProviderInfo(ctx context.Context, templateID int64, providerName string, channel domain.Channel) (domain.ChannelTemplate, error) {
	// 1. 获取模板基本信息
	template, err := s.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	if template.ID == 0 {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: templateID=%d", errs.ErrTemplateNotFound, templateID)
	}

	// 2. 获取指定的版本信息
	version, err := s.repo.GetTemplateVersionByID(ctx, template.ActiveVersionID)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	if version.AuditStatus != domain.AuditStatusApproved {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: versionID=%d", errs.ErrTemplateVersionNotApprovedByPlatform, version.ID)
	}

	// 3. 获取指定供应商信息
	providers, err := s.repo.GetProviderByNameAndChannel(ctx, templateID, version.ID, providerName, channel)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	if len(providers) == 0 {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: providerName=%s, channel=%s", errs.ErrProviderNotFound, providerName, channel)
	}

	// 4. 组装完整模板
	version.Providers = providers
	template.Versions = []domain.ChannelTemplateVersion{version}

	return template, nil
}

func (s *templateService) GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error) {
	return s.repo.GetTemplateByID(ctx, templateID)
}

func (s *templateService) CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error) {
	// 参数校验
	if err := template.Validate(); err != nil {
		return domain.ChannelTemplate{}, err
	}

	// 设置初始状态
	template.ActiveVersionID = 0 // 默认无活跃版本

	// 创建模板
	createdTemplate, err := s.repo.CreateTemplate(ctx, template)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 创建模板失败: %w", errs.ErrCreateTemplateFailed, err)
	}

	// 创建模板版本，填充伪数据
	version := domain.ChannelTemplateVersion{
		ChannelTemplateID: createdTemplate.ID,
		Name:              "版本名称，比如v1.0.0",
		Signature:         "提前配置好的可用的短信签名或者Email收件人",
		Content:           "模版变量使用${code}格式，也可以没有变量",
		Remark:            "模版使用场景或者用途说明，有利于供应商审核通过",
	}

	createdVersion, err := s.repo.CreateTemplateVersion(ctx, version)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 创建模板版本失败: %w", errs.ErrCreateTemplateFailed, err)
	}

	// 为每个供应商创建关联
	providers, err := s.providerSvc.GetByChannel(ctx, template.Channel)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 获取供应商列表失败: %w", errs.ErrCreateTemplateFailed, err)
	}
	if len(providers) == 0 {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 渠道 %s 没有可用的供应商，联系管理员配置供应商", errs.ErrCreateTemplateFailed, template.Channel)
	}
	templateProviders := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		templateProvider := domain.ChannelTemplateProvider{
			TemplateID:        createdTemplate.ID,
			TemplateVersionID: createdVersion.ID,
			ProviderID:        providers[i].ID,
			ProviderName:      providers[i].Name,
			ProviderChannel:   providers[i].Channel,
		}
		templateProviders = append(templateProviders, templateProvider)
	}
	createdProviders, err := s.repo.BatchCreateTemplateProviders(ctx, templateProviders)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 创建模板供应商关联失败: %w", errs.ErrCreateTemplateFailed, err)
	}

	// 组合
	createdVersion.Providers = createdProviders
	createdTemplate.Versions = []domain.ChannelTemplateVersion{createdVersion}
	return createdTemplate, nil
}

func (s *templateService) UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("%w: 模板名称", errs.ErrInvalidParameter)
	}

	if template.Description == "" {
		return fmt.Errorf("%w: 模板描述", errs.ErrInvalidParameter)
	}

	if !template.BusinessType.IsValid() {
		return fmt.Errorf("%w: 业务类型", errs.ErrInvalidParameter)
	}
	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateFailed, err)
	}

	return nil
}

func (s *templateService) PublishTemplate(ctx context.Context, templateID, versionID int64) error {
	if templateID <= 0 {
		return fmt.Errorf("%w: 模板ID必须大于0", errs.ErrInvalidParameter)
	}

	if versionID <= 0 {
		return fmt.Errorf("%w: 版本ID必须大于0", errs.ErrInvalidParameter)
	}

	// 检查版本是否存在并且已通过内部审核
	version, err := s.repo.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return err
	}

	// 确认版本属于该模板
	if version.ChannelTemplateID != templateID {
		return fmt.Errorf("%w: %w", errs.ErrInvalidParameter, errs.ErrTemplateAndVersionMisMatch)
	}

	// 检查版本是否通过内部审核
	if version.AuditStatus != domain.AuditStatusApproved {
		return fmt.Errorf("%w: %w: 版本ID", errs.ErrInvalidParameter, errs.ErrTemplateVersionNotApprovedByPlatform)
	}

	// 检查是否有通过供应商审核的记录
	providers, err := s.repo.GetApprovedProvidersByTemplateIDAndVersionID(ctx, templateID, versionID)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return fmt.Errorf("%w", errs.ErrTemplateVersionNotApprovedByPlatform)
	}

	// 设置活跃版本
	err = s.repo.SetTemplateActiveVersion(ctx, templateID, versionID)
	if err != nil {
		return fmt.Errorf("发布模版失败: %w", err)
	}
	return nil
}

// 模版版本相关方法

func (s *templateService) ForkVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	return s.repo.ForkTemplateVersion(ctx, versionID)
}

func (s *templateService) UpdateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error {
	// 参数校验
	if version.ID <= 0 {
		return fmt.Errorf("%w: 版本ID必须大于0", errs.ErrInvalidParameter)
	}

	// 获取当前版本
	currentVersion, err := s.repo.GetTemplateVersionByID(ctx, version.ID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateVersionFailed, err)
	}

	// 检查版本状态，只有PENDING或REJECTED状态的版本才能修改
	if currentVersion.AuditStatus != domain.AuditStatusPending && currentVersion.AuditStatus != domain.AuditStatusRejected {
		return fmt.Errorf("%w: %w: 只有待审核或拒绝状态的版本可以修改", errs.ErrUpdateTemplateVersionFailed, errs.ErrInvalidOperation)
	}

	// 允许更新部分字段
	updateVersion := domain.ChannelTemplateVersion{
		ID:        version.ID,
		Name:      version.Name,
		Signature: version.Signature,
		Content:   version.Content,
		Remark:    version.Remark,
	}

	// 更新版本
	err = s.repo.UpdateTemplateVersion(ctx, updateVersion)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateVersionFailed, err)
	}
	return nil
}

func (s *templateService) SubmitForInternalReview(ctx context.Context, versionID int64) error {
	if versionID <= 0 {
		return fmt.Errorf("%w: 版本ID必须大于0", errs.ErrInvalidParameter)
	}

	// 获取版本信息
	version, err := s.repo.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	if version.AuditStatus == domain.AuditStatusInReview || version.AuditStatus == domain.AuditStatusApproved {
		return nil
	}

	// 获取模板信息
	template, err := s.repo.GetTemplateByID(ctx, version.ChannelTemplateID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	// 获取版本关联的供应商
	providers, err := s.repo.GetProvidersByTemplateIDAndVersionID(ctx, template.ID, version.ID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	content, err := s.getJSONAuditContent(template, version, providers)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	// 创建审核记录
	auditID, err := s.auditSvc.CreateAudit(ctx, domain.Audit{
		ResourceID:   version.ID,
		ResourceType: domain.ResourceTypeTemplate,
		Content:      content,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	// 更新版本审核状态
	updateVersions := []domain.ChannelTemplateVersion{
		{
			ID:                       version.ID,
			AuditID:                  auditID,
			AuditStatus:              domain.AuditStatusInReview,
			LastReviewSubmissionTime: time.Now().Unix(),
		},
	}

	err = s.repo.BatchUpdateTemplateVersionAuditInfo(ctx, updateVersions)
	if err != nil {
		return fmt.Errorf("%w: 更新版本审核状态失败: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	return nil
}

func (s *templateService) getJSONAuditContent(template domain.ChannelTemplate, version domain.ChannelTemplateVersion, providers []domain.ChannelTemplateProvider) (string, error) {
	content := domain.AuditContent{
		OwnerID:      template.OwnerID,
		OwnerType:    template.OwnerType.String(),
		Name:         template.Name,
		Description:  template.Description,
		Channel:      template.Channel.String(),
		BusinessType: template.BusinessType.String(),
		Version:      version.Name,
		Signature:    version.Signature,
		Content:      version.Content,
		Remark:       version.Remark,
		ProviderNames: slice.Map(providers, func(_ int, src domain.ChannelTemplateProvider) string {
			return src.ProviderName
		}),
	}
	b, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("序列化审核内容失败: %w", err)
	}
	return string(b), nil
}

func (s *templateService) BatchUpdateVersionAuditStatus(ctx context.Context, versions []domain.ChannelTemplateVersion) error {
	if len(versions) == 0 {
		return nil
	}
	if err := s.repo.BatchUpdateTemplateVersionAuditInfo(ctx, versions); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateVersionAuditStatusFailed, err)
	}
	return nil
}

// 供应商相关方法

func (s *templateService) BatchSubmitForProviderReview(ctx context.Context, versionIDs []int64) error {
	for i := range versionIDs {
		_ = s.submitForProviderReview(ctx, versionIDs[i])
	}
	return nil
}

func (s *templateService) submitForProviderReview(ctx context.Context, versionID int64) error {
	// 获取版本信息
	version, err := s.repo.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return err
	}

	// 获取模板信息
	template, err := s.repo.GetTemplateByID(ctx, version.ChannelTemplateID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	// 获取供应商关联信息
	providers, err := s.repo.GetProvidersByTemplateIDAndVersionID(ctx, template.ID, versionID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	for i := range providers {
		if providers[i].AuditStatus == domain.AuditStatusPending ||
			providers[i].AuditStatus == domain.AuditStatusRejected {
			_ = s.submit(ctx, template, version, providers[i])
		}
	}
	return nil
}

func (s *templateService) submit(ctx context.Context, template domain.ChannelTemplate, version domain.ChannelTemplateVersion, provider domain.ChannelTemplateProvider) error {
	// 当前仅支持SMS渠道
	if provider.ProviderChannel != domain.ChannelSMS {
		return nil
	}
	// 获取对应的SMS客户端
	cli, err := s.getSMSClient(provider.ProviderName)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	// 构建供应商审核请求并调用
	resp, err := cli.CreateTemplate(client.CreateTemplateReq{
		TemplateName:    version.Name,
		TemplateContent: s.replacePlaceholders(version.Content, provider),
		TemplateType:    client.TemplateType(template.BusinessType),
		Remark:          version.Remark,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	// 更新供应商关联
	err = s.repo.UpdateTemplateProviderAuditInfo(ctx, domain.ChannelTemplateProvider{
		ID:                       provider.ID,
		RequestID:                resp.RequestID,
		ProviderTemplateID:       resp.TemplateID,
		AuditStatus:              domain.AuditStatusInReview,
		LastReviewSubmissionTime: time.Now().Unix(),
	})
	if err != nil {
		return fmt.Errorf("%w: 更新供应商关联失败: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}
	return nil
}

func (s *templateService) replacePlaceholders(content string, provider domain.ChannelTemplateProvider) string {
	// 仅腾讯云需要替换占位符
	if provider.ProviderName != "tencentcloud" {
		return content
	}
	re := regexp.MustCompile(`\$\{[^}]+\}`)
	counter := 0
	output := re.ReplaceAllStringFunc(content, func(_ string) string {
		counter++
		return fmt.Sprintf("{%d}", counter)
	})
	return output
}

func (s *templateService) getSMSClient(providerName string) (client.Client, error) {
	smsClient, ok := s.smsClients[providerName]
	if !ok {
		return nil, fmt.Errorf("未找到对应的供应商客户端")
	}
	return smsClient, nil
}

func (s *templateService) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error) {
	return s.repo.GetPendingOrInReviewProviders(ctx, offset, limit, utime)
}

func (s *templateService) BatchQueryAndUpdateProviderAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	if len(providers) == 0 {
		return nil
	}

	// 按渠道和供应商名称分组处理
	groupedProviders := make(map[domain.Channel]map[string][]domain.ChannelTemplateProvider)
	for i := range providers {
		channel := providers[i].ProviderChannel
		name := providers[i].ProviderName
		if _, ok := groupedProviders[channel]; !ok {
			groupedProviders[channel] = make(map[string][]domain.ChannelTemplateProvider)
		}
		groupedProviders[channel][name] = append(groupedProviders[channel][name], providers[i])
	}

	// 处理每个渠道的供应商
	for channel := range groupedProviders {
		for name := range groupedProviders[channel] {
			if channel.IsSMS() {
				if err := s.batchQueryAndUpdateSMSProvidersAuditInfo(ctx, groupedProviders[channel][name]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *templateService) batchQueryAndUpdateSMSProvidersAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	const first = 0
	smsClient, err := s.getSMSClient(providers[first].ProviderName)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateProviderAuditStatusFailed, err)
	}

	// 获取供应商侧的模版ID 和 映射关系
	templateIDs := make([]string, 0, len(providers))
	providerMap := make(map[string]domain.ChannelTemplateProvider, len(providers))
	for i := range providers {
		templateIDs = append(templateIDs, providers[i].ProviderTemplateID)
		providerMap[providers[i].ProviderTemplateID] = providers[i]
	}

	// 批量查询模版状态
	results, err := smsClient.BatchQueryTemplateStatus(client.BatchQueryTemplateStatusReq{
		TemplateIDs: templateIDs,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateProviderAuditStatusFailed, err)
	}

	// 更新对应的状态信息
	updates := make([]domain.ChannelTemplateProvider, 0, len(results.Results))
	for i := range results.Results {
		p, ok := providerMap[results.Results[i].TemplateID]
		if !ok {
			continue
		}
		p.RequestID = results.Results[i].RequestID
		p.AuditStatus = results.Results[i].AuditStatus.ToDomain()
		p.RejectReason = results.Results[i].Reason
		updates = append(updates, p)
	}
	return s.repo.BatchUpdateTemplateProvidersAuditInfo(ctx, updates)
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
