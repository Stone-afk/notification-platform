package repository

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/cache"
	"notification-platform/internal/repository/dao"
)

type quotaRepository struct {
	dao dao.QuotaDAO
}

func (repo *quotaRepository) CreateOrUpdate(ctx context.Context, quota ...domain.Quota) error {
	qs := slice.Map(quota, func(_ int, src domain.Quota) dao.Quota {
		return dao.Quota{
			Quota:   src.Quota,
			BizID:   src.BizID,
			Channel: src.Channel.String(),
		}
	})
	return repo.dao.CreateOrUpdate(ctx, qs...)
}

func (repo *quotaRepository) Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error) {
	found, err := repo.dao.Find(ctx, bizID, channel.String())
	if err != nil {
		return domain.Quota{}, err
	}
	return domain.Quota{
		BizID:   found.BizID,
		Quota:   found.Quota,
		Channel: domain.Channel(found.Channel),
	}, nil
}

func NewQuotaRepository(dao dao.QuotaDAO) QuotaRepository {
	return &quotaRepository{dao: dao}
}

type quotaRepositoryV2 struct {
	ca cache.QuotaCache
}

func NewQuotaRepositoryV2(ca cache.QuotaCache) QuotaRepository {
	return &quotaRepositoryV2{ca}
}

func (q *quotaRepositoryV2) CreateOrUpdate(ctx context.Context, quota ...domain.Quota) error {
	return q.ca.CreateOrUpdate(ctx, quota...)
}

func (q *quotaRepositoryV2) Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error) {
	return q.ca.Find(ctx, bizID, channel)
}
