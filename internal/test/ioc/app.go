package ioc

import (
	"context"
	"github.com/gotomicro/ego/server/egrpc"
	"github.com/gotomicro/ego/task/ecron"
	prodioc "notification-platform/internal/ioc"
	"notification-platform/internal/repository"
	configsvc "notification-platform/internal/service/config"
	notificationsvc "notification-platform/internal/service/notification"
	"notification-platform/internal/service/notification/callback"
	providersvc "notification-platform/internal/service/provider/manage"
	quotasvc "notification-platform/internal/service/quota"
	templatesvc "notification-platform/internal/service/template/manage"
)

type App struct {
	GrpcServer *egrpc.Component
	Tasks      []prodioc.Task
	Crons      []ecron.Ecron

	CallbackSvc     callback.CallbackService
	CallbackLogRepo repository.CallbackLogRepository

	ConfigSvc  configsvc.BusinessConfigService
	ConfigRepo repository.BusinessConfigRepository

	NotificationSvc     notificationsvc.NotificationService
	SendNotificationSvc notificationsvc.SendNotificationService
	NotificationRepo    repository.NotificationRepository

	ProviderSvc  providersvc.Service
	ProviderRepo repository.ProviderRepository

	QuotaSvc  quotasvc.QuotaService
	QuotaRepo repository.QuotaRepository

	TemplateSvc  templatesvc.ChannelTemplateService
	TemplateRepo repository.ChannelTemplateRepository

	TxNotificationSvc  notificationsvc.TxNotificationService
	TxNotificationRepo repository.TxNotificationRepository
}

func (a *App) StartTasks(ctx context.Context) {
	for _, t := range a.Tasks {
		go func(t prodioc.Task) {
			t.Start(ctx)
		}(t)
	}
}
