package config

import (
	"context"
	"notification-platform/internal/domain"
)

//go:generate mockgen -source=./config.go -destination=./mocks/config.mock.go -package=configmocks -typed BusinessConfigService
type BusinessConfigService interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error)
	Delete(ctx context.Context, id int64) error
	// SaveConfig 保存非零字段
	SaveConfig(ctx context.Context, config domain.BusinessConfig) error
}
