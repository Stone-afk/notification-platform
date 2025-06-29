package errs

import (
	"errors"
)

// 定义统一的错误类型
var (
	// 业务错误
	ErrInvalidParameter                     = errors.New("参数错误")
	ErrSendNotificationFailed               = errors.New("发送通知失败")
	ErrNotificationNotFound                 = errors.New("通知记录不存在")
	ErrCreateNotificationFailed             = errors.New("创建通知失败")
	ErrBizIDNotFound                        = errors.New("BizID不存在")
	ErrTemplateNotFound                     = errors.New("模板不存在")
	ErrTemplateVersionNotFound              = errors.New("模板版本不存在")
	ErrTemplateVersionNotApprovedByPlatform = errors.New("模板版本未被内部审核通过")
	ErrTemplateVersionNotApprovedByProvider = errors.New("模板版本未被供应商审核通过")
	ErrTemplateAndVersionMisMatch           = errors.New("模板和版本不匹配")
	ErrChannelDisabled                      = errors.New("渠道已禁用")
	ErrRateLimited                          = errors.New("请求频率受限")
	ErrCircuitBreaker                       = errors.New("服务熔断，请稍后重试")
	ErrNoAvailableProvider                  = errors.New("无可用供应商")
	ErrNoAvailableChannel                   = errors.New("无可用渠道")
	ErrConfigNotFound                       = errors.New("业务配置不存在")
	ErrNoQuotaConfig                        = errors.New("没有提供 Quota 有关的配置")
	ErrNoQuota                              = errors.New("额度已经用完")
	ErrQuotaNotFound                        = errors.New("额度记录不存在")
	ErrProviderNotFound                     = errors.New("供应商记录不存在")
	ErrUnknownChannel                       = errors.New("未知渠道类型")
	ErrInvalidOperation                     = errors.New("无效的操作")

	ErrCreateTemplateFailed                    = errors.New("创建模版失败")
	ErrUpdateTemplateFailed                    = errors.New("更新模版失败")
	ErrForkVersionFailed                       = errors.New("拷贝模版版本失败")
	ErrUpdateTemplateVersionFailed             = errors.New("更新版本失败")
	ErrUpdateTemplateVersionAuditStatusFailed  = errors.New("更新版本审核状态失败")
	ErrUpdateTemplateProviderAuditStatusFailed = errors.New("更新渠道供应商审核状态失败")
	ErrSubmitVersionForInternalReviewFailed    = errors.New("提交模版版本内部审核失败")
	ErrSubmitVersionForProviderReviewFailed    = errors.New("提交模版版本供应商审核失败")

	ErrNoAvailableFailoverService = errors.New("没有需要接管的故障服务")

	// 系统错误
	ErrNotificationDuplicate       = errors.New("通知记录主键冲突")
	ErrNotificationVersionMismatch = errors.New("通知记录版本不匹配")
	ErrCreateCallbackLogFailed     = errors.New("创建回调记录失败")
	ErrDatabaseError               = errors.New("数据库错误")
	ErrExternalServiceError        = errors.New("外部服务调用错误")
	ErrBatchSizeOverLimit          = errors.New("批量大小超过限制")

	ErrToAsync     = errors.New("服务崩溃已转异步")
	ErrExceedLimit = errors.New("抢资源超出限制")

	ErrErrorConditionIsMet = errors.New("错误事件出现次数或错误率达到阈值")
)
