//go:build wireinject

package callback

import (
	"github.com/google/wire"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/cache/redis"
	"notification-platform/internal/repository/dao"
	configsvc "notification-platform/internal/service/config"
	callbacksvc "notification-platform/internal/service/notification/callback"
	testioc "notification-platform/internal/test/ioc"
)

type Service struct {
	Svc              callbacksvc.CallbackService
	Repo             repository.CallbackLogRepository
	NotificationRepo repository.NotificationRepository
	QuotaRepo        repository.QuotaRepository
}

func Init(cnfigSvc configsvc.BusinessConfigService) *Service {
	wire.Build(
		testioc.BaseSet,
		callbacksvc.NewCallbackService,
		repository.NewCallbackLogRepository,
		dao.NewCallbackLogDAO,
		repository.NewNotificationRepository,
		dao.NewNotificationDAO,
		redis.NewQuotaCache,
		repository.NewQuotaRepositoryV2,

		wire.Struct(new(Service), "*"),
	)
	return nil
}
