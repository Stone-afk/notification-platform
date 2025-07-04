package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gotomicro/ego/core/elog"
	"github.com/redis/go-redis/v9"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/cache"
)

var _ cache.ConfigCache = (*Cache)(nil)

type Cache struct {
	rdb    *redis.Client
	logger *elog.Component
}

func (c *Cache) Get(ctx context.Context, bizID int64) (domain.BusinessConfig, error) {
	key := cache.ConfigKey(bizID)
	// 从Redis获取数据
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 键不存在
			return domain.BusinessConfig{}, cache.ErrKeyNotFound
		}
		// 其他错误
		return domain.BusinessConfig{}, fmt.Errorf("failed to get config from redis %w", err)
	}

	// 反序列化数据
	var cfg domain.BusinessConfig
	err = json.Unmarshal([]byte(val), &cfg)
	if err != nil {
		return domain.BusinessConfig{}, fmt.Errorf("failed to unmarshal config data %w", err)
	}

	return cfg, nil
}

func (c *Cache) Set(ctx context.Context, cfg domain.BusinessConfig) error {
	key := cache.ConfigKey(cfg.ID)

	// 序列化数据
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config data %w", err)
	}

	// 存储到Redis
	err = c.rdb.Set(ctx, key, data, cache.DefaultExpiredTime).Err()
	if err != nil {
		return fmt.Errorf("failed to set config from redis %w", err)
	}
	return nil
}

func (c *Cache) Del(ctx context.Context, bizID int64) error {
	return c.rdb.Del(ctx, cache.ConfigKey(bizID)).Err()
}

func (c *Cache) GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error) {
	if len(bizIDs) == 0 {
		return make(map[int64]domain.BusinessConfig), nil
	}

	// 准备所有的键
	keys := make([]string, len(bizIDs))
	for i, bizID := range bizIDs {
		keys[i] = cache.ConfigKey(bizID)
	}

	// 使用 MGET 批量获取数据
	vals, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis执行MGET失败: %w", err)
	}

	// 处理结果
	result := make(map[int64]domain.BusinessConfig)
	for i, val := range vals {
		if val == nil {
			// 这个键不存在，跳过
			continue
		}

		// 将字符串转换为结构体
		strVal, ok := val.(string)
		if !ok {
			continue
		}

		var cfg domain.BusinessConfig
		if err := json.Unmarshal([]byte(strVal), &cfg); err != nil {
			// 解析错误，记录错误但继续处理其他配置
			c.logger.Error("从redis序列化数据失败", elog.Any("err", err), elog.String("key", keys[i]), elog.String("val", strVal))
			continue
		}
		result[bizIDs[i]] = cfg
	}

	return result, nil
}

func (c *Cache) SetConfigs(ctx context.Context, configs []domain.BusinessConfig) error {
	if len(configs) == 0 {
		return nil
	}

	// 使用管道批量设置，提高性能
	// 这边是一个性能优化的写法
	// 在集群模式下，命中听一个节点的 key 会被打包作为一个 pipeline
	// 要确保你的 Redis 客户端支持自动分组/智能路由
	pipe := c.rdb.Pipeline()

	for _, cfg := range configs {
		key := cache.ConfigKey(cfg.ID)

		// 序列化数据
		data, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config data for ID %d: %w", cfg.ID, err)
		}

		// 加入管道
		pipe.Set(ctx, key, data, cache.DefaultExpiredTime)
	}

	// 执行管道中的所有命令
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline for setting configs: %w", err)
	}

	return nil
}

func NewCache(rdb *redis.Client) *Cache {
	return &Cache{
		rdb: rdb,
	}
}
