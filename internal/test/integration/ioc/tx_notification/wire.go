//go:build wireinject

package tx_notification

import (
	"github.com/google/wire"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/cache/redis"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/service/config"
	"notification-platform/internal/service/notification"
	notificationtask "notification-platform/internal/service/notification/task"
	"notification-platform/internal/service/sender"
	testioc "notification-platform/internal/test/ioc"
)

type App struct {
	Svc  notification.TxNotificationService
	Task *notificationtask.TxCheckTask
}

func InitTxNotificationService(configSvc config.BusinessConfigService, sender sender.NotificationSender) *App {
	wire.Build(
		testioc.BaseSet,
		dao.NewTxNotificationDAO,
		dao.NewNotificationDAO,
		redis.NewQuotaCache,
		repository.NewNotificationRepository,
		repository.NewTxNotificationRepository,
		notification.NewTxNotificationService,
		notificationtask.NewTxCheckTask,
		wire.Struct(new(App), "*"),
	)
	return new(App)
}
