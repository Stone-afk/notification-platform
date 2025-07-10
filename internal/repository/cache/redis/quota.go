package redis

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/gotomicro/ego/core/elog"
	"github.com/redis/go-redis/v9"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/cache"
)

var (
	ErrQuotaLessThenZero = errors.New("额度小于0")
	//go:embed lua/quota.lua
	quotaScript string
	//go:embed lua/batch_decr_quota.lua
	batchDecrQuotaScript string
	//go:embed lua/batch_incr_quota.lua
	batchIncrQuotaScript string
)

type quotaCache struct {
	client redis.Cmdable
	logger *elog.Component
}

func (c *quotaCache) CreateOrUpdate(ctx context.Context, quotas ...domain.Quota) error {
	const (
		number = 2
	)
	vals := make([]any, 0, number*len(quotas))
	for _, quota := range quotas {
		vals = append(vals, c.key(quota), quota.Quota)
	}
	return c.client.MSet(ctx, vals...).Err()
}

func (c *quotaCache) Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error) {
	quota, err := c.client.Get(ctx, c.key(domain.Quota{
		BizID:   bizID,
		Channel: channel,
	})).Int()
	if err != nil {
		return domain.Quota{}, err
	}
	return domain.Quota{
		BizID:   bizID,
		Channel: channel,
		Quota:   int32(quota),
	}, nil
}

func (c *quotaCache) Incr(ctx context.Context, bizID int64, channel domain.Channel, quota int32) error {
	return c.client.Eval(ctx, quotaScript, []string{c.key(domain.Quota{
		BizID:   bizID,
		Channel: channel,
	})}, quota).Err()
}

func (c *quotaCache) Decr(ctx context.Context, bizID int64, channel domain.Channel, quota int32) error {
	res, err := c.client.DecrBy(ctx, c.key(domain.Quota{
		BizID:   bizID,
		Channel: channel,
	}), int64(quota)).Result()
	if err != nil {
		return err
	}
	if res < 0 {
		elog.Error("库存不足", elog.Int("biz_id", int(bizID)), elog.String("channel", channel.String()))
		return ErrQuotaLessThenZero
	}
	return nil
}

func (c *quotaCache) MutiIncr(ctx context.Context, items []cache.IncrItem) error {
	//TODO implement me
	panic("implement me")
}

func (c *quotaCache) MutiDecr(ctx context.Context, items []cache.IncrItem) error {
	//TODO implement me
	panic("implement me")
}

func (c *quotaCache) key(quota domain.Quota) string {
	return fmt.Sprintf("quota:%d:%s", quota.BizID, quota.Channel)
}

func NewQuotaCache(client redis.Cmdable) cache.QuotaCache {
	return &quotaCache{
		client: client,
		logger: elog.DefaultLogger,
	}
}
