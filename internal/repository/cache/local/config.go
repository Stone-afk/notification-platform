package local

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gotomicro/ego/core/elog"
	ca "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/cache"
	"strings"
	"time"
)

const (
	defaultTimeout = 3 * time.Second
)

var _ cache.ConfigCache = (*Cache)(nil)

type Cache struct {
	rdb        *redis.Client
	logger     *elog.Component
	localCache *ca.Cache
}

func (c *Cache) Get(ctx context.Context, bizID int64) (domain.BusinessConfig, error) {
	key := cache.ConfigKey(bizID)
	v, ok := c.localCache.Get(key)
	if !ok {
		return domain.BusinessConfig{}, cache.ErrKeyNotFound
	}
	vv, ok := v.(domain.BusinessConfig)
	if !ok {
		return domain.BusinessConfig{}, errors.New("数据类型不正确")
	}
	return vv, nil
}

func (c *Cache) Set(ctx context.Context, cfg domain.BusinessConfig) error {
	key := cache.ConfigKey(cfg.ID)
	c.localCache.Set(key, cfg, cache.DefaultExpiredTime)
	return nil
}

func (c *Cache) Del(ctx context.Context, bizID int64) error {
	c.localCache.Delete(cache.ConfigKey(bizID))
	return nil
}

func (c *Cache) GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error) {
	configMap := make(map[int64]domain.BusinessConfig)
	for _, bizID := range bizIDs {
		v, ok := c.localCache.Get(cache.ConfigKey(bizID))
		if ok {
			vv, ok := v.(domain.BusinessConfig)
			if !ok {
				return configMap, errors.New("数据类型不正确")
			}
			configMap[bizID] = vv
		}
	}
	return configMap, nil
}

func (c *Cache) SetConfigs(ctx context.Context, configs []domain.BusinessConfig) error {
	for _, config := range configs {
		c.localCache.Set(cache.ConfigKey(config.ID), config, cache.DefaultExpiredTime)
	}
	return nil
}

// 监控redis
func (c *Cache) loop(ctx context.Context) {
	// 就这个 channel 的表达式
	pubsub := c.rdb.PSubscribe(ctx, "__keyspace@*__:config:*")
	defer func() {
		err := pubsub.Close()
		if err != nil {
			c.logger.Error("关闭 Redis 订阅失败", elog.Any("err", err))
		}
	}()
	ch := pubsub.Channel()
	for msg := range ch {
		// 在线上环境，小心别把敏感数据打出来了
		// 比如说你的 channel 里面包含了手机号码，你就别打了
		c.logger.Info("监控到 Redis 更新消息",
			elog.String("key", msg.Channel), elog.String("payload", msg.Payload))

		const channelMinLen = 2
		channel := msg.Channel
		channelStrList := strings.SplitN(channel, ":", channelMinLen)
		if len(channelStrList) < channelMinLen {
			c.logger.Error("监听到非法 Redis key", elog.String("channel", msg.Channel))
			continue
		}
		// config:133 => 133
		const keyIdx = 1
		key := channelStrList[keyIdx]
		ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
		eventType := msg.Payload
		c.handleConfigChange(ctx, key, eventType)
		cancel()
	}
}

func (c *Cache) handleConfigChange(ctx context.Context, key, event string) {
	// 自定义业务逻辑（如动态更新配置）
	switch event {
	case "set":
		res := c.rdb.Get(ctx, key)
		if res.Err() != nil {
			c.logger.Error("订阅完获取键失败", elog.String("key", key))
		}
		var config domain.BusinessConfig
		err := json.Unmarshal([]byte(res.Val()), &config)
		if err != nil {
			c.logger.Error("序列化失败", elog.String("key", key), elog.String("val", res.Val()))
		}
		c.localCache.Set(key, config, cache.DefaultExpiredTime)
	case "del":
		c.localCache.Delete(key)
	}
}

func NewLocalCache(rdb *redis.Client, localCache *ca.Cache) *Cache {
	res := &Cache{
		rdb:        rdb,
		logger:     elog.DefaultLogger,
		localCache: localCache,
	}
	// 开启监控redis里的内容
	// 我要在这里监听 Redis 的 key 变更，更新本地缓存
	go res.loop(context.Background())
	return res
}
