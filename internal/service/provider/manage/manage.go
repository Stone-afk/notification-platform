package manage

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository"
)

// providerService 供应商服务实现
type providerService struct {
	repo repository.ProviderRepository
}

func (s *providerService) Create(ctx context.Context, provider domain.Provider) (domain.Provider, error) {
	//TODO implement me
	panic("implement me")
}

func (s *providerService) Update(ctx context.Context, provider domain.Provider) error {
	//TODO implement me
	panic("implement me")
}

func (s *providerService) GetByID(ctx context.Context, id int64) (domain.Provider, error) {
	//TODO implement me
	panic("implement me")
}

func (s *providerService) GetByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error) {
	//TODO implement me
	panic("implement me")
}

// NewProviderService 创建供应商服务
func NewProviderService(repo repository.ProviderRepository) Service {
	return &providerService{
		repo: repo,
	}
}
