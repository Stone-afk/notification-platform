package ratelimit

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

var (
	//go:embed lua/slide_window.lua
	slidingWindowScript string

	//go:embed lua/last_limit_time.lua
	lastLimitTimeScript string

	_ Limiter = (*RedisSlidingWindowLimiter)(nil)
)

type RedisSlidingWindowLimiter struct {
	cmd       redis.Cmdable
	interval  time.Duration
	rate      int
	keyPrefix string
}

// NewRedisSlidingWindowLimiter 创建一个基于Redis的滑动窗口限流器
func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, interval time.Duration, rate int) *RedisSlidingWindowLimiter {
	return &RedisSlidingWindowLimiter{
		cmd:       cmd,
		interval:  interval,
		rate:      rate,
		keyPrefix: "ratelimit:",
	}
}

// Limit 判断是否应该限流
func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, slidingWindowScript,
		[]string{r.getCountKey(key), r.getLimitedEventKey(key)},
		r.interval.Milliseconds(),
		r.rate,
		time.Now().UnixMilli(),
	).Bool()
}

// getCountKey 获取请求计数的Redis键
func (r *RedisSlidingWindowLimiter) getCountKey(key string) string {
	return fmt.Sprintf("%scount:%s", r.keyPrefix, key)
}

// getLimitedEventKey 获取限流事件记录的Redis键
func (r *RedisSlidingWindowLimiter) getLimitedEventKey(key string) string {
	return fmt.Sprintf("%slimitedEvent:%s", r.keyPrefix, key)
}

// LastLimitTime 获取最近一次限流发生的时间，如果没有发生过限流则返回零值
func (r *RedisSlidingWindowLimiter) LastLimitTime(ctx context.Context, key string) (time.Time, error) {
	result, err := r.cmd.Eval(ctx, lastLimitTimeScript,
		[]string{r.getLimitedEventKey(key)}).Int64()
	if err != nil {
		return time.Time{}, err
	}

	if result == 0 {
		// 没有限流事件发生，返回零值
		return time.Time{}, nil
	}

	// 将毫秒时间戳转换为time.Time
	return time.UnixMilli(result), nil
}
