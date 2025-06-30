package repository

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
)

type providerRepository struct {
	dao dao.ProviderDAO
}

func (repo *providerRepository) Create(ctx context.Context, provider domain.Provider) (domain.Provider, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *providerRepository) Update(ctx context.Context, provider domain.Provider) error {
	//TODO implement me
	panic("implement me")
}

func (repo *providerRepository) FindByID(ctx context.Context, id int64) (domain.Provider, error) {
	//TODO implement me
	panic("implement me")
}

func (repo *providerRepository) FindByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error) {
	//TODO implement me
	panic("implement me")
}

func NewProviderRepository(dao dao.ProviderDAO) ProviderRepository {
	return &providerRepository{dao: dao}
}
