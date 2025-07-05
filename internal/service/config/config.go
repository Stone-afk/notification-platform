package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/ego-component/egorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
)

var ErrIDNotSet = errors.New("业务id没有设置")

type BusinessConfigServiceV1 struct {
	repo repository.BusinessConfigRepository
}

func (s *BusinessConfigServiceV1) GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error) {
	// 参数校验
	if len(ids) == 0 {
		return make(map[int64]domain.BusinessConfig), nil
	}

	// 调用仓库层方法
	return s.repo.GetByIDs(ctx, ids)
}

func (s *BusinessConfigServiceV1) GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error) {
	// 参数校验
	if id <= 0 {
		return domain.BusinessConfig{}, fmt.Errorf("%w", errs.ErrInvalidParameter)
	}

	// 调用仓库层方法
	config, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, egorm.ErrRecordNotFound) {
			return domain.BusinessConfig{}, fmt.Errorf("%w", errs.ErrConfigNotFound)
		}
		return domain.BusinessConfig{}, err
	}

	return config, nil
}

func (s *BusinessConfigServiceV1) Delete(ctx context.Context, id int64) error {
	// 参数校验
	if id <= 0 {
		return fmt.Errorf("%w", errs.ErrInvalidParameter)
	}

	// 调用仓库层删除方法
	return s.repo.Delete(ctx, id)
}

func (s *BusinessConfigServiceV1) SaveConfig(ctx context.Context, config domain.BusinessConfig) error {
	// 参数校验
	if config.ID <= 0 {
		return ErrIDNotSet
	}
	// 调用仓库层保存方法
	return s.repo.SaveConfig(ctx, config)
}

// NewBusinessConfigService 创建业务配置服务实例
func NewBusinessConfigService(repo repository.BusinessConfigRepository) BusinessConfigService {
	return &BusinessConfigServiceV1{
		repo: repo,
	}
}
