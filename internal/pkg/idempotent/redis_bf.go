package idempotent

import (
	"context"
	"errors"
	"github.com/ecodeclub/ekit/slice"
	"github.com/redis/go-redis/v9"
)

type BloomIdempotencyService struct {
	client     redis.Cmdable
	filterName string
	capacity   uint64  // 预期容量
	errorRate  float64 // 误判率
}

func (s *BloomIdempotencyService) Exists(ctx context.Context, key string) (bool, error) {
	res, err := s.client.BFAdd(ctx, s.filterName, key).Result()
	return !res, err
}

func (s *BloomIdempotencyService) MExists(ctx context.Context, keys ...string) ([]bool, error) {
	if len(keys) == 0 {
		return nil, errors.New("empty keys")
	}
	// 执行批量查询
	res := s.client.BFMAdd(ctx, s.filterName, slice.Map(keys, func(_ int, src string) any {
		return src
	})...)
	val, err := res.Result()
	if err != nil {
		return nil, err
	}
	return slice.Map(val, func(_ int, src bool) bool {
		return !src
	}), nil
}

func NewBloomService(client redis.Cmdable, filterName string,
	capacity uint64, errorRate float64,
) *BloomIdempotencyService {
	return &BloomIdempotencyService{
		client:     client,
		filterName: filterName,
		capacity:   capacity,
		errorRate:  errorRate,
	}
}
