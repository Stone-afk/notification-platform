package audit

import (
	"context"
	"notification-platform/internal/domain"
)

//go:generate mockgen -source=./audit_result_event_producer.go -package=evtmocks -destination=../mocks/audit.mock.go -typed ResultCallbackEventProducer
type ResultCallbackEventProducer interface {
	Produce(ctx context.Context, evt CallbackResultEvent) error
}

const (
	eventName = "audit_result_events"
)

type CallbackResultEvent struct {
	ResourceID   int64               `json:"resourceId"`   // 模版版本ID
	ResourceType domain.ResourceType `json:"resourceType"` // TEMPLATE
	AuditID      int64               `json:"auditId"`      // 审核记录ID
	AuditorID    int64               `json:"auditorId"`    // 审核人ID
	AuditTime    int64               `json:"auditTime"`    // 审核时间
	AuditStatus  string              `json:"auditStatus"`  // 审核状态
	RejectReason string              `json:"rejectReason"` // 拒绝原因
}
