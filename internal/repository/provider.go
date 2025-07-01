package repository

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository/dao"
)

type providerRepository struct {
	dao dao.ProviderDAO
}

func (repo *providerRepository) Create(ctx context.Context, provider domain.Provider) (domain.Provider, error) {
	created, err := repo.dao.Create(ctx, repo.toEntity(provider))
	if err != nil {
		return domain.Provider{}, err
	}
	return repo.toDomain(created), nil
}

func (repo *providerRepository) Update(ctx context.Context, provider domain.Provider) error {
	return repo.dao.Update(ctx, repo.toEntity(provider))
}

func (repo *providerRepository) FindByID(ctx context.Context, id int64) (domain.Provider, error) {
	provider, err := repo.dao.FindByID(ctx, id)
	if err != nil {
		// 处理未找到的情况，转换为领域错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Provider{}, fmt.Errorf("%w", errs.ErrProviderNotFound)
		}
		return domain.Provider{}, err
	}
	return repo.toDomain(provider), nil
}

func (repo *providerRepository) FindByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error) {
	providers, err := repo.dao.FindByChannel(ctx, channel.String())
	if err != nil {
		return nil, err
	}

	result := make([]domain.Provider, 0, len(providers))
	for i := range providers {
		result = append(result, repo.toDomain(providers[i]))
	}

	return result, nil
}

func (repo *providerRepository) toDomain(d dao.Provider) domain.Provider {
	return domain.Provider{
		ID:               d.ID,
		Name:             d.Name,
		Channel:          domain.Channel(d.Channel),
		Endpoint:         d.Endpoint,
		APIKey:           d.APIKey,
		APISecret:        d.APISecret,
		Weight:           d.Weight,
		QPSLimit:         d.QPSLimit,
		DailyLimit:       d.DailyLimit,
		AuditCallbackURL: d.AuditCallbackURL,
		Status:           domain.ProviderStatus(d.Status),
	}
}

func (repo *providerRepository) toEntity(provider domain.Provider) dao.Provider {
	daoProvider := dao.Provider{
		ID:               provider.ID,
		Name:             provider.Name,
		Channel:          provider.Channel.String(),
		Endpoint:         provider.Endpoint,
		APIKey:           provider.APIKey,
		APISecret:        provider.APISecret,
		Weight:           provider.Weight,
		QPSLimit:         provider.QPSLimit,
		DailyLimit:       provider.DailyLimit,
		AuditCallbackURL: provider.AuditCallbackURL,
		Status:           provider.Status.String(),
	}
	return daoProvider
}

func NewProviderRepository(dao dao.ProviderDAO) ProviderRepository {
	return &providerRepository{dao: dao}
}
