//go:build wireinject

package notification

import (
	"github.com/google/wire"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/cache"
	"notification-platform/internal/repository/cache/redis"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/service/notification"
	testioc "notification-platform/internal/test/ioc"
)

type Service struct {
	Svc             notification.NotificationService
	QuotaCache      cache.QuotaCache
	Repo            repository.NotificationRepository
	QuotaRepo       repository.QuotaRepository
	CallbackLogRepo repository.CallbackLogRepository
}

func Init() *Service {
	wire.Build(
		testioc.BaseSet,
		redis.NewQuotaCache,
		repository.NewNotificationRepository,
		notification.NewNotificationService,
		dao.NewNotificationDAO,

		repository.NewQuotaRepositoryV2,

		repository.NewCallbackLogRepository,
		dao.NewCallbackLogDAO,

		wire.Struct(new(Service), "*"),
	)
	return nil
}
