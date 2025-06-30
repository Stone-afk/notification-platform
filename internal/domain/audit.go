package domain

type ResourceType string

const (
	ResourceTypeTemplate ResourceType = "Template"
)

func (r ResourceType) IsTemplate() bool {
	return r == ResourceTypeTemplate
}

type Audit struct {
	ResourceID   int64        // 模版版本ID
	ResourceType ResourceType // TEMPLATE
	Content      string       // 完整JSON串，模版信息-基本+版本+渠道名（多个）
}

type AuditContent struct {
	OwnerID       int64    `json:"ownerId"`       // 拥有者ID，用户ID或部门ID
	OwnerType     string   `json:"ownerType"`     // 拥有者类型
	Name          string   `json:"name"`          // 模板名称
	Description   string   `json:"description"`   // 模板描述
	Channel       string   `json:"channel"`       // 渠道类型
	BusinessType  string   `json:"businessType"`  // 业务类型
	Version       string   `json:"version"`       // 版本名称
	Signature     string   `json:"signature"`     // 签名
	Content       string   `json:"content"`       // 模板内容
	Remark        string   `json:"remark"`        // 申请说明
	ProviderNames []string `json:"providerNames"` // 供应商名称
}
