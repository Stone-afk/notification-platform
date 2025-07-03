package manage

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
)

// providerService 供应商服务实现
type providerService struct {
	repo repository.ProviderRepository
}

func (s *providerService) Create(ctx context.Context, provider domain.Provider) (domain.Provider, error) {
	if err := provider.Validate(); err != nil {
		return domain.Provider{}, err
	}
	return s.repo.Create(ctx, provider)
}

func (s *providerService) Update(ctx context.Context, provider domain.Provider) error {
	if err := provider.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, provider)
}

func (s *providerService) GetByID(ctx context.Context, id int64) (domain.Provider, error) {
	if id <= 0 {
		return domain.Provider{}, fmt.Errorf("%w: 供应商ID必须大于0", errs.ErrInvalidParameter)
	}
	return s.repo.FindByID(ctx, id)
}

func (s *providerService) GetByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error) {
	if !channel.IsValid() {
		return nil, fmt.Errorf("%w: 不支持的渠道类型", errs.ErrInvalidParameter)
	}
	return s.repo.FindByChannel(ctx, channel)
}

// NewProviderService 创建供应商服务
func NewProviderService(repo repository.ProviderRepository) Service {
	return &providerService{
		repo: repo,
	}
}
