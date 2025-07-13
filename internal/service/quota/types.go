package quota

import (
	"context"
	"notification-platform/internal/domain"
)

type QuotaService interface {
	ResetQuota(ctx context.Context, biz domain.BusinessConfig) error
}
