package quota

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
)

type service struct {
	repo repository.QuotaRepository
}

func (s *service) ResetQuota(ctx context.Context, biz domain.BusinessConfig) error {
	if biz.Quota == nil {
		return errs.ErrNoQuotaConfig
	}
	sms := domain.Quota{
		BizID:   biz.ID,
		Quota:   int32(biz.Quota.Monthly.SMS),
		Channel: domain.ChannelSMS,
	}
	email := domain.Quota{
		BizID:   biz.ID,
		Quota:   int32(biz.Quota.Monthly.EMAIL),
		Channel: domain.ChannelEmail,
	}
	return s.repo.CreateOrUpdate(ctx, sms, email)
}

func NewQuotaService(repo repository.QuotaRepository) QuotaService {
	return &service{repo: repo}
}
