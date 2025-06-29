package client

import (
	"errors"
	"notification-platform/internal/domain"
)

const (
	OK = "OK"
)

// 通用错误定义
var (
	ErrCreateTemplateFailed = errors.New("创建模版失败")
	ErrQueryTemplateStatus  = errors.New("查询模版状态失败")
	ErrSendFailed           = errors.New("发送短信失败")
	ErrQuerySendDetails     = errors.New("查询发送详情失败")
	ErrInvalidParameter     = errors.New("参数无效")
)

type (
	AuditStatus  int
	TemplateType int32
)

type SendStatus int

const (
	TemplateTypeInternational TemplateType = 0 // 国际/港澳台消息 仅阿里云使用
	TemplateTypeMarketing     TemplateType = 1 // 营销短信
	TemplateTypeNotification  TemplateType = 2 // 通知短信
	TemplateTypeVerification  TemplateType = 3 // 验证码

	AuditStatusPending  AuditStatus = 0 // 审核中
	AuditStatusApproved AuditStatus = 1 // 审核通过
	AuditStatusRejected AuditStatus = 2 // 审核拒绝

	SendStatusWaiting SendStatus = 1 // 等待回执
	SendStatusSuccess SendStatus = 3 // 发送成功
	SendStatusFailed  SendStatus = 2 // 发送失败
)

func (a AuditStatus) IsPending() bool {
	return a == AuditStatusPending
}

func (a AuditStatus) IsApproved() bool {
	return a == AuditStatusApproved
}

func (a AuditStatus) IsRejected() bool {
	return a == AuditStatusRejected
}

func (a AuditStatus) ToDomain() domain.AuditStatus {
	switch a {
	case AuditStatusPending:
		return domain.AuditStatusInReview
	case AuditStatusApproved:
		return domain.AuditStatusApproved
	case AuditStatusRejected:
		return domain.AuditStatusRejected
	default:
		return domain.AuditStatusInReview
	}
}

// Client 短信客户端接口 (抽象)
//
//go:generate mockgen -source=./types.go -destination=./mocks/sms.mock.go -package=smsmocks -typed Client
type Client interface {
	// CreateTemplate 创建模板
	CreateTemplate(req CreateTemplateReq) (CreateTemplateResp, error)
	// BatchQueryTemplateStatus 批量查询模板状态
	BatchQueryTemplateStatus(req BatchQueryTemplateStatusReq) (BatchQueryTemplateStatusResp, error)
	// Send 发送短信
	Send(req SendReq) (SendResp, error)
}

// CreateTemplateReq 创建短信模板请求参数
type CreateTemplateReq struct {
	TemplateName    string       // 模板名称
	TemplateContent string       // 模板内容
	TemplateType    TemplateType // 短信类型
	Remark          string       // 备注
}

// CreateTemplateResp 创建短信模板响应参数
type CreateTemplateResp struct {
	RequestID  string // 请求 ID,   阿里云、腾讯云共用
	TemplateID string // 模板 ID, 阿里云、腾讯云共用 (阿里云返回 TemplateCode, 腾讯云返回处理过的 TemplateID)
}

// BatchQueryTemplateStatusReq 批量查询短信模板状态请求参数
type BatchQueryTemplateStatusReq struct {
	TemplateIDs []string // 模板 ID, 阿里云、腾讯云共用
}

// BatchQueryTemplateStatusResp 批量查询短信模板状态响应参数
type BatchQueryTemplateStatusResp struct {
	Results map[string]QueryTemplateStatusResp // 键是模版ID，值是每个模版对应的响应
}

// QueryTemplateStatusResp 单个模版查询状态响应
type QueryTemplateStatusResp struct {
	RequestID   string      // 请求 ID,   阿里云、腾讯云共用
	TemplateID  string      // 模板 ID, 阿里云、腾讯云共用
	AuditStatus AuditStatus // 模板审核状态, 阿里云、腾讯云共用 (0: 审核中, 1: 审核通过, 2: 审核失败/拒绝)
	Reason      string      // 审核失败原因, 阿里云、腾讯云共用
}

// SendReq 发送短信请求参数
type SendReq struct {
	PhoneNumbers  []string          // 手机号码, 阿里云、腾讯云共用
	SignName      string            // 签名名称, 阿里云、腾讯云共用
	TemplateID    string            // 模板 ID, 阿里云、腾讯云共用
	TemplateParam map[string]string // 模板参数, 阿里云、腾讯云共用, key-value 形式
}

// SendResp 发送短信响应参数
type SendResp struct {
	RequestID    string                    // 请求 ID,      阿里云、腾讯云共用
	PhoneNumbers map[string]SendRespStatus // 去掉+86后的手机号
}

type SendRespStatus struct {
	Code    string
	Message string
}

// QuerySendDetailsReq 查询短信发送详情请求参数
type QuerySendDetailsReq struct {
	PhoneNumber string // 手机号码, 阿里云、腾讯云共用
	BizID       string // 发送回执 ID, 阿里云使用BizId，腾讯云使用SerialNo
	SendDate    string // 短信发送日期，格式yyyyMMdd, 阿里云使用
	PageSize    int    // 分页大小,   阿里云、腾讯云共用
	CurrentPage int    // 当前页码,    阿里云、腾讯云共用
	// 以下为腾讯云独有
	BeginTime int64  // 起始时间。 (UNIX 时间戳, 腾讯云使用)
	EndTime   int64  // 结束时间。 (UNIX 时间戳, 腾讯云使用)
	Offset    uint64 // 偏移量。 (腾讯云使用)
	Limit     uint64 // 最大条数。 (腾讯云使用)
}

// QuerySendDetailsResp 查询短信发送详情响应参数
type QuerySendDetailsResp struct {
	RequestID         string       // 请求 ID,     阿里云、腾讯云共用
	TotalCount        int          // 短信发送总条数, 阿里云、腾讯云共用
	SmsSendDetailDTOs []SendDetail // 短信发送详情列表, 阿里云、腾讯云共用
}

// SendDetail 短信发送详情
type SendDetail struct {
	PhoneNum   string // 手机号码,      阿里云、腾讯云共用
	SendStatus int    // 发送状态,      阿里云、腾讯云共用 (1: 等待回执, 2: 发送失败, 3: 发送成功)
	Content    string // 短信内容,      阿里云、腾讯云共用
	// 下面内容阿里云独有
	TemplateCode string // 模版CODE
	SendDate     string // 发送时间
	ReceiveDate  string // 接收时间
	ErrCode      string // 错误码
	OutID        string // 外部流水扩展字段
	// 以下为腾讯云独有
	SerialNo        string // 发送序列号
	ReportStatus    int    // 实际是否收到短信接收状态
	UserReceiveTime string // 用户接收时间
}
