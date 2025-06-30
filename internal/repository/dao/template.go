package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ego-component/egorm"

	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"time"
)

// channelTemplateDAO 实现了ChannelTemplateDAO接口，提供对模板数据的数据库访问实现
type channelTemplateDAO struct {
	db *egorm.Component
}

// GetTemplatesByOwner 根据所有者获取模板列表
func (dao *channelTemplateDAO) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType string) ([]ChannelTemplate, error) {
	var templates []ChannelTemplate
	err := dao.db.WithContext(ctx).Where("owner_id = ? AND owner_type = ?", ownerID, ownerType).Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// GetTemplateByID 根据ID获取模板
func (dao *channelTemplateDAO) GetTemplateByID(ctx context.Context, id int64) (ChannelTemplate, error) {
	var template ChannelTemplate
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ChannelTemplate{}, fmt.Errorf("%w", errs.ErrTemplateNotFound)
		}
		return ChannelTemplate{}, err
	}
	return template, nil
}

// CreateTemplate 创建模板
func (dao *channelTemplateDAO) CreateTemplate(ctx context.Context, template ChannelTemplate) (ChannelTemplate, error) {
	now := time.Now().Unix()
	template.Ctime = now
	template.Utime = now

	result := dao.db.WithContext(ctx).Create(&template)
	if result.Error != nil {
		return ChannelTemplate{}, result.Error
	}
	return template, nil
}

// UpdateTemplate 更新模板基本信息
func (dao *channelTemplateDAO) UpdateTemplate(ctx context.Context, template ChannelTemplate) error {
	// 只允许用户更新name、description、business_type这三个字段
	updateData := map[string]any{
		"name":          template.Name,
		"description":   template.Description,
		"business_type": template.BusinessType,
		"utime":         time.Now().Unix(),
	}

	return dao.db.WithContext(ctx).Model(&ChannelTemplate{}).
		Where("id = ?", template.ID).
		Updates(updateData).Error
}

// SetTemplateActiveVersion 设置模板活跃版本
func (dao *channelTemplateDAO) SetTemplateActiveVersion(ctx context.Context, templateID, versionID int64) error {
	return dao.db.WithContext(ctx).Model(&ChannelTemplate{}).
		Where("id = ?", templateID).
		Updates(map[string]any{
			"active_version_id": versionID,
			"utime":             time.Now().Unix(),
		}).Error
}

// GetTemplateVersionsByTemplateIDs 根据模板IDs获取版本列表
func (dao *channelTemplateDAO) GetTemplateVersionsByTemplateIDs(ctx context.Context, templateIDs []int64) ([]ChannelTemplateVersion, error) {
	if len(templateIDs) == 0 {
		return []ChannelTemplateVersion{}, nil
	}

	var versions []ChannelTemplateVersion
	err := dao.db.WithContext(ctx).
		Where("channel_template_id IN ?", templateIDs).
		Find(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// GetTemplateVersionByID 根据ID获取模板版本
func (dao *channelTemplateDAO) GetTemplateVersionByID(ctx context.Context, versionID int64) (ChannelTemplateVersion, error) {
	var version ChannelTemplateVersion
	err := dao.db.WithContext(ctx).Where("id = ?", versionID).First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ChannelTemplateVersion{}, fmt.Errorf("%w", errs.ErrTemplateVersionNotFound)
		}
		return ChannelTemplateVersion{}, err
	}
	return version, nil
}

// CreateTemplateVersion 创建模板版本
func (dao *channelTemplateDAO) CreateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) (ChannelTemplateVersion, error) {
	now := time.Now().Unix()
	version.Ctime = now
	version.Utime = now
	err := dao.db.WithContext(ctx).Create(&version).Error
	if err != nil {
		return ChannelTemplateVersion{}, err
	}
	return version, nil
}

func (dao *channelTemplateDAO) ForkTemplateVersion(ctx context.Context, versionID int64) (ChannelTemplateVersion, error) {
	now := time.Now().Unix()
	var created ChannelTemplateVersion

	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 找到被拷贝记录
		var old ChannelTemplateVersion
		if err := tx.First(&old, "id = ?", versionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w", errs.ErrTemplateVersionNotFound)
			}
			return err
		}
		// 拷贝记录
		fork := ChannelTemplateVersion{
			ChannelTemplateID:        old.ChannelTemplateID,
			Name:                     "Forked" + old.Name,
			Signature:                old.Signature,
			Content:                  old.Content,
			Remark:                   old.Remark,
			AuditID:                  0,
			AuditorID:                0,
			AuditTime:                0,
			AuditStatus:              domain.AuditStatusPending.String(),
			RejectReason:             "",
			LastReviewSubmissionTime: 0,
			Ctime:                    now,
			Utime:                    now,
		}
		if err := tx.Create(&fork).Error; err != nil {
			return err
		}
		created = fork

		// 获取供应商
		var providers []ChannelTemplateProvider
		if err := tx.Model(&ChannelTemplateProvider{}).
			Where("template_id = ? AND template_version_id = ?",
				old.ChannelTemplateID, old.ID).Find(&providers).Error; err != nil {
			return err
		}

		forkedProviders := slice.Map(providers, func(_ int, src ChannelTemplateProvider) ChannelTemplateProvider {
			return ChannelTemplateProvider{
				TemplateID:               fork.ChannelTemplateID,
				TemplateVersionID:        fork.ID,
				ProviderID:               src.ProviderID,
				ProviderName:             src.ProviderName,
				ProviderChannel:          src.ProviderChannel,
				RequestID:                "",
				ProviderTemplateID:       "",
				AuditStatus:              domain.AuditStatusPending.String(),
				RejectReason:             "",
				LastReviewSubmissionTime: 0,
				Ctime:                    now,
				Utime:                    now,
			}
		})
		if err := tx.Create(&forkedProviders).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ChannelTemplateVersion{}, err
	}
	return created, nil
}

// GetProvidersByVersionIDs 根据版本IDs获取供应商关联
func (dao *channelTemplateDAO) GetProvidersByVersionIDs(ctx context.Context, versionIDs []int64) ([]ChannelTemplateProvider, error) {
	if len(versionIDs) == 0 {
		return []ChannelTemplateProvider{}, nil
	}

	var providers []ChannelTemplateProvider
	err := dao.db.WithContext(ctx).
		Where("template_version_id IN (?)", versionIDs).
		Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// GetProviderByNameAndChannel 根据名称和渠道获取已通过审核的供应商
func (dao *channelTemplateDAO) GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName string, channel string) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := dao.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("template_id = ? AND template_version_id = ? AND provider_name = ? AND provider_channel = ? AND audit_status = ?",
			templateID, versionID, providerName, channel, domain.AuditStatusApproved).Find(&providers).Error
	return providers, err
}

// BatchCreateTemplateProviders 批量创建模板供应商关联
func (dao *channelTemplateDAO) BatchCreateTemplateProviders(ctx context.Context, providers []ChannelTemplateProvider) ([]ChannelTemplateProvider, error) {
	if len(providers) == 0 {
		return []ChannelTemplateProvider{}, nil
	}

	now := time.Now().Unix()
	for i := range providers {
		providers[i].Ctime = now
		providers[i].Utime = now
	}

	result := dao.db.WithContext(ctx).Create(&providers)
	if result.Error != nil {
		return nil, result.Error
	}
	return providers, nil
}

// GetApprovedProvidersByTemplateIDAndVersionID 根据模版ID和版本ID查找审核通过的供应商
func (dao *channelTemplateDAO) GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := dao.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("template_id = ? AND template_version_id = ? AND audit_status = ?",
			templateID, versionID, domain.AuditStatusApproved).Find(&providers).Error
	return providers, err
}

func (dao *channelTemplateDAO) GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := dao.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("template_id = ? AND template_version_id = ?",
			templateID, versionID).Find(&providers).Error
	return providers, err
}

// UpdateTemplateVersion 更新模板版本信息
func (dao *channelTemplateDAO) UpdateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) error {
	// 只允许更新部分字段
	updateData := map[string]any{
		"name":      version.Name,
		"signature": version.Signature,
		"content":   version.Content,
		"remark":    version.Remark,
		"utime":     time.Now().Unix(),
	}

	return dao.db.WithContext(ctx).Model(&ChannelTemplateVersion{}).
		Where("id = ?", version.ID).
		Updates(updateData).Error
}

// BatchUpdateTemplateVersionAuditInfo 更新模板版本审核信息
func (dao *channelTemplateDAO) BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []ChannelTemplateVersion) error {
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range versions {
			updateData := map[string]any{
				"audit_status": versions[i].AuditStatus,
				"utime":        time.Now().Unix(),
			}

			// 有条件地添加其他字段
			if versions[i].AuditID > 0 {
				updateData["audit_id"] = versions[i].AuditID
			}
			if versions[i].AuditorID > 0 {
				updateData["auditor_id"] = versions[i].AuditorID
			}
			if versions[i].AuditTime > 0 {
				updateData["audit_time"] = versions[i].AuditTime
			}
			if versions[i].RejectReason != "" {
				updateData["reject_reason"] = versions[i].RejectReason
			}
			if versions[i].LastReviewSubmissionTime > 0 {
				updateData["last_review_submission_time"] = versions[i].LastReviewSubmissionTime
			}

			if err := tx.Model(&ChannelTemplateVersion{}).
				Where("id = ?", versions[i].ID).
				Updates(updateData).Error; err != nil {
				return err
			}

		}
		return nil
	})
}

// UpdateTemplateProviderAuditInfo 更新模板供应商审核信息
func (dao *channelTemplateDAO) UpdateTemplateProviderAuditInfo(ctx context.Context, provider ChannelTemplateProvider) error {
	// 构建更新数据
	updateData := map[string]any{
		"utime": time.Now().Unix(),
	}

	// 有条件地添加其他字段
	if provider.RequestID != "" {
		updateData["request_id"] = provider.RequestID
	}
	if provider.ProviderTemplateID != "" {
		updateData["provider_template_id"] = provider.ProviderTemplateID
	}
	if provider.AuditStatus != "" {
		updateData["audit_status"] = provider.AuditStatus
	}
	if provider.RejectReason != "" {
		updateData["reject_reason"] = provider.RejectReason
	}
	if provider.LastReviewSubmissionTime > 0 {
		updateData["last_review_submission_time"] = provider.LastReviewSubmissionTime
	}

	return dao.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("id = ?", provider.ID).
		Updates(updateData).Error
}

// BatchUpdateTemplateProvidersAuditInfo 批量更新模板供应商审核信息
func (dao *channelTemplateDAO) BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []ChannelTemplateProvider) error {
	if len(providers) == 0 {
		return nil
	}

	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range providers {
			// 构建更新数据
			updateData := map[string]any{
				"utime": time.Now().Unix(),
			}

			// 有条件地添加其他字段
			if providers[i].RequestID != "" {
				updateData["request_id"] = providers[i].RequestID
			}
			if providers[i].ProviderTemplateID != "" {
				updateData["provider_template_id"] = providers[i].ProviderTemplateID
			}
			if providers[i].AuditStatus != "" {
				updateData["audit_status"] = providers[i].AuditStatus
			}
			if providers[i].RejectReason != "" {
				updateData["reject_reason"] = providers[i].RejectReason
			}
			if providers[i].LastReviewSubmissionTime > 0 {
				updateData["last_review_submission_time"] = providers[i].LastReviewSubmissionTime
			}

			err := tx.Model(&ChannelTemplateProvider{}).
				Where("id = ?", providers[i].ID).
				Updates(updateData).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
func (dao *channelTemplateDAO) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := dao.db.WithContext(ctx).
		Where("(audit_status = ? OR audit_status = ?) AND utime <= ?",
			domain.AuditStatusPending.String(),
			domain.AuditStatusInReview.String(),
			utime).
		Offset(offset).
		Limit(limit).
		Find(&providers).Error
	return providers, err
}

// TotalPendingOrInReviewProviders 统计未审核或审核中的供应商关联总数
func (dao *channelTemplateDAO) TotalPendingOrInReviewProviders(ctx context.Context, utime int64) (int64, error) {
	var res int64
	err := dao.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("(audit_status = ? OR audit_status = ?) AND utime <= ?",
			domain.AuditStatusPending.String(),
			domain.AuditStatusInReview.String(),
			utime).
		Count(&res).Error
	return res, err
}

// NewChannelTemplateDAO 创建模板DAO实例
func NewChannelTemplateDAO(db *egorm.Component) ChannelTemplateDAO {
	return &channelTemplateDAO{
		db: db,
	}
}

// ChannelTemplate 渠道模板表
type ChannelTemplate struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;comment:'渠道模版ID'"`
	OwnerID         int64  `gorm:"type:BIGINT;NOT NULL;comment:'用户ID或部门ID'"`
	OwnerType       string `gorm:"type:ENUM('person', 'organization');NOT NULL;comment:'业务方类型：person-个人,organization-组织'"`
	Name            string `gorm:"type:VARCHAR(128);NOT NULL;comment:'模板名称'"`
	Description     string `gorm:"type:VARCHAR(512);NOT NULL;comment:'模板描述'"`
	Channel         string `gorm:"type:ENUM('SMS','EMAIL','IN_APP');NOT NULL;comment:'渠道类型'"`
	BusinessType    int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:1;comment:'业务类型：1-推广营销、2-通知、3-验证码等'"`
	ActiveVersionID int64  `gorm:"type:BIGINT;DEFAULT:0;index:idx_active_version;comment:'当前启用的版本ID，0表示无活跃版本'"`
	Ctime           int64
	Utime           int64
}

// TableName 重命名表
func (ChannelTemplate) TableName() string {
	return "channel_template"
}

// ChannelTemplateVersion 渠道模板版本表
type ChannelTemplateVersion struct {
	ID                int64  `gorm:"primaryKey;autoIncrement;comment:'渠道模版版本ID'"`
	ChannelTemplateID int64  `gorm:"type:BIGINT;NOT NULL;index:idx_channel_template_id;comment:'关联渠道模版ID'"`
	Name              string `gorm:"type:VARCHAR(32);NOT NULL;comment:'版本名称，如v1.0.0'"`
	Signature         string `gorm:"type:VARCHAR(64);comment:'已通过所有供应商审核的短信签名/邮件发件人'"`
	Content           string `gorm:"type:TEXT;NOT NULL;comment:'原始模板内容，使用平台统一变量格式，如${name}'"`
	Remark            string `gorm:"type:TEXT;NOT NULL;comment:'申请说明,描述使用短信的业务场景，并提供短信完整示例（填入变量内容），信息完整有助于提高模板审核通过率。'"`
	// 审核相关信息，AuditID之后的为冗余的信息
	AuditID                  int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:0;comment:'审核表ID, 0表示尚未提交审核或者未拿到审核结果'"`
	AuditorID                int64  `gorm:"type:BIGINT;comment:'审核人ID'"`
	AuditTime                int64  `gorm:"comment:'审核时间'"`
	AuditStatus              string `gorm:"type:ENUM('PENDING','IN_REVIEW','REJECTED','APPROVED');NOT NULL;DEFAULT:'PENDING';comment:'内部审核状态，PENDING表示未提交审核；IN_REVIEW表示已提交审核；APPROVED表示审核通过；REJECTED表示审核未通过'"`
	RejectReason             string `gorm:"type:VARCHAR(512);comment:'拒绝原因'"`
	LastReviewSubmissionTime int64  `gorm:"comment:'上一次提交审核时间'"`
	Ctime                    int64
	Utime                    int64
}

// TableName 重命名表
func (ChannelTemplateVersion) TableName() string {
	return "channel_template_version"
}

type ChannelTemplateProvider struct {
	ID                       int64  `gorm:"primaryKey;autoIncrement;comment:'渠道模版-供应商关联ID'"`
	TemplateID               int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:idx_template_version_provider,priority:1;uniqueIndex:idx_tmpl_ver_name_chan,priority:1;comment:'渠道模版ID'"`
	TemplateVersionID        int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:idx_template_version_provider,priority:2;uniqueIndex:idx_tmpl_ver_name_chan,priority:2;comment:'渠道模版版本ID'"`
	ProviderID               int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:idx_template_version_provider,priority:3;comment:'供应商ID'"`
	ProviderName             string `gorm:"type:VARCHAR(64);NOT NULL;uniqueIndex:idx_tmpl_ver_name_chan,priority:3;comment:'供应商名称'"`
	ProviderChannel          string `gorm:"type:ENUM('SMS','EMAIL','IN_APP');NOT NULL;uniqueIndex:idx_tmpl_ver_name_chan,priority:4;comment:'渠道类型'"`
	RequestID                string `gorm:"type:VARCHAR(256);index:idx_request_id;comment:'审核请求在供应商侧的ID，用于排查问题'"`
	ProviderTemplateID       string `gorm:"type:VARCHAR(256);comment:'当前版本模版在供应商侧的ID，审核通过后才会有值'"`
	AuditStatus              string `gorm:"type:ENUM('PENDING','IN_REVIEW','REJECTED','APPROVED');NOT NULL;DEFAULT:'PENDING';index:idx_audit_status;comment:'供应商侧模版审核状态，PENDING表示未提交审核；IN_REVIEW表示已提交审核；APPROVED表示审核通过；REJECTED表示审核未通过'"`
	RejectReason             string `gorm:"type:VARCHAR(512);comment:'供应商侧拒绝原因'"`
	LastReviewSubmissionTime int64  `gorm:"comment:'上一次提交审核时间'"`
	Ctime                    int64
	Utime                    int64
}

// TableName 重命名表
func (ChannelTemplateProvider) TableName() string {
	return "channel_template_provider"
}
