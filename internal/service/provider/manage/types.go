package manage

import (
	"context"
	"notification-platform/internal/domain"
)

//go:generate mockgen -source=./manage.go -destination=../mocks/manage.mock.go -package=providermocks -typed Service
type Service interface {
	// Create 创建供应商
	Create(ctx context.Context, provider domain.Provider) (domain.Provider, error)
	// Update 更新供应商
	Update(ctx context.Context, provider domain.Provider) error
	// GetByID 根据ID获取供应商
	GetByID(ctx context.Context, id int64) (domain.Provider, error)
	// GetByChannel 获取指定渠道的所有供应商
	GetByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error)
}
