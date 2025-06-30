package domain

import (
	"fmt"
	"notification-platform/internal/errs"
)

// AuditStatus 审核状态
type AuditStatus string

const (
	AuditStatusPending  AuditStatus = "PENDING"   // 待审核
	AuditStatusInReview AuditStatus = "IN_REVIEW" // 审核中
	AuditStatusRejected AuditStatus = "REJECTED"  // 已拒绝
	AuditStatusApproved AuditStatus = "APPROVED"  // 已通过
)

func (a AuditStatus) String() string {
	return string(a)
}

func (a AuditStatus) IsPending() bool {
	return a == AuditStatusPending
}

func (a AuditStatus) IsInReview() bool {
	return a == AuditStatusInReview
}

func (a AuditStatus) IsRejected() bool {
	return a == AuditStatusRejected
}

func (a AuditStatus) IsApproved() bool {
	return a == AuditStatusApproved
}

func (a AuditStatus) IsValid() bool {
	switch a {
	case AuditStatusPending, AuditStatusInReview, AuditStatusApproved, AuditStatusRejected:
		return true
	default:
		return false
	}
}

// OwnerType 拥有者类型
type OwnerType string

const (
	OwnerTypePerson       OwnerType = "person"       // 个人
	OwnerTypeOrganization OwnerType = "organization" // 组织
)

func (o OwnerType) String() string {
	return string(o)
}

func (o OwnerType) IsValid() bool {
	return o == OwnerTypePerson || o == OwnerTypeOrganization
}

type BusinessType int64

const (
	// BusinessTypePromotion 推广营销
	BusinessTypePromotion BusinessType = 1
	// BusinessTypeNotification 通知
	BusinessTypeNotification BusinessType = 2
	// BusinessTypeVerificationCode 验证码
	BusinessTypeVerificationCode BusinessType = 3
)

func (b BusinessType) ToInt64() int64 {
	return int64(b)
}

func (b BusinessType) IsValid() bool {
	return b == BusinessTypePromotion ||
		b == BusinessTypeNotification || b == BusinessTypeVerificationCode
}

func (b BusinessType) String() string {
	switch b {
	case BusinessTypePromotion:
		return "推广营销"
	case BusinessTypeNotification:
		return "通知"
	case BusinessTypeVerificationCode:
		return "验证码"
	default:
		return "未知类型"
	}
}

// ChannelTemplateProvider 渠道模板供应商关联
type ChannelTemplateProvider struct {
	ID                       int64       // 关联ID
	TemplateID               int64       // 模板ID
	TemplateVersionID        int64       // 模版版本ID
	ProviderID               int64       // 供应商ID
	ProviderName             string      // 供应商名称
	ProviderChannel          Channel     // 供应商渠道类型
	RequestID                string      // 审核请求ID
	ProviderTemplateID       string      // 供应商侧模板ID
	AuditStatus              AuditStatus // 审核状态
	RejectReason             string      // 拒绝原因
	LastReviewSubmissionTime int64       // 上次提交审核时间
	Ctime                    int64       // 创建时间
	Utime                    int64       // 更新时间
}

// ChannelTemplateVersion 渠道模板版本
type ChannelTemplateVersion struct {
	ID                       int64       // 版本ID
	ChannelTemplateID        int64       // 模板ID
	Name                     string      // 版本名称
	Signature                string      // 签名
	Content                  string      // 模板内容
	Remark                   string      // 申请说明
	AuditID                  int64       // 审核记录ID
	AuditorID                int64       // 审核人ID
	AuditTime                int64       // 审核时间
	AuditStatus              AuditStatus // 审核状态
	RejectReason             string      // 拒绝原因
	LastReviewSubmissionTime int64       // 上次提交审核时间
	Ctime                    int64       // 创建时间
	Utime                    int64       // 更新时间

	Providers []ChannelTemplateProvider // 关联的所有供应商
}

// ChannelTemplate 渠道模板
type ChannelTemplate struct {
	ID              int64        // 模板ID
	OwnerID         int64        // 拥有者ID，用户ID或部门ID
	OwnerType       OwnerType    // 拥有者类型
	Name            string       // 模板名称
	Description     string       // 模板描述
	Channel         Channel      // 渠道类型
	BusinessType    BusinessType // 业务类型
	ActiveVersionID int64        // 活跃版本ID，0表示无活跃版本
	Ctime           int64        // 创建时间
	Utime           int64        // 更新时间

	Versions []ChannelTemplateVersion // 关联的所有版本
}

func (t *ChannelTemplate) Validate() error {
	if t.OwnerID <= 0 {
		return fmt.Errorf("%w: 所有者ID", errs.ErrInvalidParameter)
	}

	if !t.OwnerType.IsValid() {
		return fmt.Errorf("%w: 所有者类型", errs.ErrInvalidParameter)
	}

	if t.Name == "" {
		return fmt.Errorf("%w: 模板名称", errs.ErrInvalidParameter)
	}

	if t.Description == "" {
		return fmt.Errorf("%w: 模板描述", errs.ErrInvalidParameter)
	}

	if !t.Channel.IsValid() {
		return fmt.Errorf("%w: 渠道类型", errs.ErrInvalidParameter)
	}

	if !t.BusinessType.IsValid() {
		return fmt.Errorf("%w: 业务类型", errs.ErrInvalidParameter)
	}
	return nil
}

// HasPublished 是否已发布
func (t *ChannelTemplate) HasPublished() bool {
	return t.ActiveVersionID != 0
}

// ActiveVersion 获取当前活跃版本
func (t *ChannelTemplate) ActiveVersion() *ChannelTemplateVersion {
	if t.ActiveVersionID == 0 {
		return nil
	}

	for i := range t.Versions {
		if t.Versions[i].ID == t.ActiveVersionID {
			return &t.Versions[i]
		}
	}
	return nil
}

// GetVersion 根据版本ID获取版本
func (t *ChannelTemplate) GetVersion(versionID int64) *ChannelTemplateVersion {
	for i := range t.Versions {
		if t.Versions[i].ID == versionID {
			return &t.Versions[i]
		}
	}
	return nil
}

// GetProvidersByVersion 获取特定版本的所有供应商关联
func (t *ChannelTemplate) GetProvidersByVersion(versionID int64) []ChannelTemplateProvider {
	version := t.GetVersion(versionID)
	if version == nil {
		return nil
	}
	return version.Providers
}

// GetProvider 获取特定版本和供应商的关联
func (t *ChannelTemplate) GetProvider(versionID, providerID int64) *ChannelTemplateProvider {
	version := t.GetVersion(versionID)
	if version == nil {
		return nil
	}
	for i := range version.Providers {
		if version.Providers[i].ProviderID == providerID {
			return &version.Providers[i]
		}
	}
	return nil
}

// HasApprovedVersion 检查是否有已审核通过的版本
func (t *ChannelTemplate) HasApprovedVersion() bool {
	for i := range t.Versions {
		if t.Versions[i].AuditStatus == AuditStatusApproved {
			return true
		}
	}
	return false
}
