package audit

import (
	"context"
	"notification-platform/internal/domain"
)

//go:generate mockgen -source=./audit.go -destination=./mocks/audit.mock.go -package=auditmocks -typed Service
type Service interface {
	CreateAudit(ctx context.Context, req domain.Audit) (int64, error)
}
