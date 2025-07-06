package quota

import (
	"context"
	"notification-platform/internal/domain"
)

type Service interface {
	ResetQuota(ctx context.Context, biz domain.BusinessConfig) error
}
