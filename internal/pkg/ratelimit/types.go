package ratelimit

import (
	"time"

	"golang.org/x/net/context"
)

//go:generate mockgen -source=./types.go -package=limitmocks -destination=./mocks/limiter.mock.go Limiter
type Limiter interface {
	// Limit 判断是否应该限流
	Limit(ctx context.Context, key string) (bool, error)
	// LastLimitTime 获取最近一次限流发生的时间，如果没有发生过限流则返回零值
	LastLimitTime(ctx context.Context, key string) (time.Time, error)
}
