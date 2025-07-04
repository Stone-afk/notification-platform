package cache

import (
	"context"
	"errors"
	"fmt"
	"notification-platform/internal/domain"
	"time"
)

var ErrKeyNotFound = errors.New("key not found")

const (
	ConfigPrefix       = "config"
	DefaultExpiredTime = 10 * time.Minute
)

type IncrItem struct {
	BizID   int64
	Channel domain.Channel
	Val     int32
}

type ConfigCache interface {
	Get(ctx context.Context, bizID int64) (domain.BusinessConfig, error)
	Set(ctx context.Context, cfg domain.BusinessConfig) error
	Del(ctx context.Context, bizID int64) error
	GetConfigs(ctx context.Context, bizIDs []int64) (map[int64]domain.BusinessConfig, error)
	SetConfigs(ctx context.Context, configs []domain.BusinessConfig) error
}

func ConfigKey(bizID int64) string {
	return fmt.Sprintf("%s:%d", ConfigPrefix, bizID)
}

type QuotaCache interface {
	CreateOrUpdate(ctx context.Context, quota ...domain.Quota) error
	Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error)
	Incr(ctx context.Context, bizID int64, channel domain.Channel, quota int32) error
	Decr(ctx context.Context, bizID int64, channel domain.Channel, quota int32) error
	MutiIncr(ctx context.Context, items []IncrItem) error
	MutiDecr(ctx context.Context, items []IncrItem) error
}
