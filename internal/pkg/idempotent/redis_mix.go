package idempotent

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	filterName     = "mix_filter"
	defaultTimeout = 10 * time.Minute
)

// RedisMix 结合布隆过滤器 和 redis存储值的方案
type RedisMix struct {
	client redis.Cmdable
}

func (r *RedisMix) Exists(ctx context.Context, key string) (bool, error) {
	res, err := r.client.BFAdd(ctx, filterName, key).Result()
	if err != nil {
		return false, err
	}
	// 加没加成功都需要
	redisRes, rerr := r.client.SetNX(ctx, r.getKey(key), "1", defaultTimeout).Result()
	if rerr != nil {
		return false, err
	}
	// 加成功了，说明不存在将存储值也设置一下
	if res {
		return false, nil
	}
	// 没加成功 说明已经存在再看一下存储值有咩有
	return !redisRes, nil
}

func (r *RedisMix) MExists(ctx context.Context, keys ...string) ([]bool, error) {
	results := make([]bool, len(keys))
	needCheckKeys := make([]string, 0, len(keys))
	setKeys := make([]string, 0, len(keys))

	// 批量执行BFMAdd
	bfmCmd := r.client.BFMAdd(ctx, filterName, slice.Map(keys, func(_ int, src string) any {
		return src
	})...)
	res, err := bfmCmd.Result()
	if err != nil {
		return nil, err
	}

	// 处理BFMAdd结果
	for i, added := range res {
		// 添加没成功说明，已经存在,需要二次确认
		if !added {
			results[i] = true
			needCheckKeys = append(needCheckKeys, keys[i])
		} else {
			// 添加成功，说明不存在，作为redis值也需要存储一下
			setKeys = append(setKeys, keys[i])
		}
	}

	// 批量执行SetNX
	setCmds := make(map[string]*redis.BoolCmd)
	pipe := r.client.Pipeline()
	// 布隆过滤器判断，判定存在可能存在假阳性，需要二次确认往redis中存储一下值
	if len(needCheckKeys) > 0 {
		for _, key := range needCheckKeys {
			setCmds[key] = pipe.SetNX(ctx, r.getKey(key), "1", defaultTimeout)
		}
	}
	// 布隆过滤过滤器，判定不存在，也需要在redis中存储一下值
	if len(setKeys) > 0 {
		for _, key := range setKeys {
			_ = pipe.SetNX(ctx, r.getKey(key), "1", defaultTimeout)
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	// 处理SetNX结果并维护原始顺序
	for i, key := range keys {
		if cmd, ok := setCmds[key]; ok {
			res, err := cmd.Result()
			if err != nil {
				return nil, err
			}
			results[i] = !res
		}
	}
	return results, nil
}

func (r *RedisMix) getKey(key string) string {
	return fmt.Sprintf("mixIdempotency:%s", key)
}

func NewRedisMix(client redis.Cmdable) *RedisMix {
	return &RedisMix{
		client: client,
	}
}
