//go:build wireinject

package config

import (
	"github.com/google/wire"
	ca "github.com/patrickmn/go-cache"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/cache/local"
	"notification-platform/internal/repository/cache/redis"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/service/config"
	testioc "notification-platform/internal/test/ioc"
)

func InitConfigService(localCache *ca.Cache) config.BusinessConfigService {
	wire.Build(
		testioc.BaseSet,
		local.NewLocalCache,
		redis.NewCache,
		dao.NewBusinessConfigDAO,
		repository.NewBusinessConfigRepository,
		config.NewBusinessConfigService,
	)
	return new(config.BusinessConfigServiceV1)
}
