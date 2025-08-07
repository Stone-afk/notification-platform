//go:build wireinject

package ioc

import (
	notificationsvc "notification-platform/internal/service/notification"
	"time"

	"github.com/ecodeclub/ekit/pool"
	"github.com/gotomicro/ego/core/econf"
	"notification-platform/internal/service/quota"
	quotatask "notification-platform/internal/service/quota/task"
	"notification-platform/internal/service/scheduler"
	testioc "notification-platform/internal/test/ioc"

	"github.com/google/wire"
	grpcapi "notification-platform/internal/api/grpc"
	"notification-platform/internal/domain"
	prodioc "notification-platform/internal/ioc"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/cache/local"
	"notification-platform/internal/repository/cache/redis"
	"notification-platform/internal/repository/dao"
	auditsvc "notification-platform/internal/service/audit"
	"notification-platform/internal/service/channel"
	configsvc "notification-platform/internal/service/config"
	"notification-platform/internal/service/notification/callback"
	notificationsvctask "notification-platform/internal/service/notification/task"
	"notification-platform/internal/service/provider"
	providersvc "notification-platform/internal/service/provider/manage"
	"notification-platform/internal/service/provider/sequential"
	"notification-platform/internal/service/provider/sms"
	"notification-platform/internal/service/provider/sms/client"
	"notification-platform/internal/service/sender"
	"notification-platform/internal/service/sendstrategy"
	templatesvc "notification-platform/internal/service/template/manage"
)

var (
	BaseSet = wire.NewSet(
		prodioc.InitDB,
		prodioc.InitDistributedLock,
		prodioc.InitEtcdClient,
		prodioc.InitIDGenerator,
		prodioc.InitRedisClient,
		prodioc.InitGoCache,
		prodioc.InitRedisCmd,

		local.NewLocalCache,
		redis.NewCache,
	)
	configSvcSet = wire.NewSet(
		configsvc.NewBusinessConfigService,
		repository.NewBusinessConfigRepository,
		dao.NewBusinessConfigDAO)
	notificationSvcSet = wire.NewSet(
		redis.NewQuotaCache,
		notificationsvc.NewNotificationService,
		repository.NewNotificationRepository,
		dao.NewNotificationDAO,
		notificationsvctask.NewSendingTimeoutTask,
	)
	txNotificationSvcSet = wire.NewSet(
		notificationsvc.NewTxNotificationService,
		repository.NewTxNotificationRepository,
		dao.NewTxNotificationDAO,
		notificationsvctask.NewTxCheckTask,
	)
	senderSvcSet = wire.NewSet(
		newChannel,
		newTaskPool,
		sender.NewSender,
	)
	sendNotificationSvcSet = wire.NewSet(
		notificationsvc.NewSendService,
		sendstrategy.NewDispatcher,
		sendstrategy.NewImmediateStrategy,
		sendstrategy.NewDefaultStrategy,
	)
	callbackSvcSet = wire.NewSet(
		callback.NewCallbackService,
		repository.NewCallbackLogRepository,
		dao.NewCallbackLogDAO,
		notificationsvctask.NewAsyncRequestResultCallbackTask,
	)
	providerSvcSet = wire.NewSet(
		providersvc.NewProviderService,
		repository.NewProviderRepository,
		dao.NewProviderDAO,
		// 加密密钥
		prodioc.InitProviderEncryptKey,
	)
	templateSvcSet = wire.NewSet(
		templatesvc.NewChannelTemplateService,
		repository.NewChannelTemplateRepository,
		dao.NewChannelTemplateDAO,
	)
	schedulerSet = wire.NewSet(scheduler.NewScheduler)
	quotaSvcSet  = wire.NewSet(
		quota.NewQuotaService,
		quotatask.NewQuotaMonthlyResetCron,
		repository.NewQuotaRepository,
		dao.NewQuotaDAO)
)

func newTaskPool() pool.TaskPool {
	type Config struct {
		InitGo           int           `yaml:"initGo"`
		CoreGo           int32         `yaml:"coreGo"`
		MaxGo            int32         `yaml:"maxGo"`
		MaxIdleTime      time.Duration `yaml:"maxIdleTime"`
		QueueSize        int           `yaml:"queueSize"`
		QueueBacklogRate float64       `yaml:"queueBacklogRate"`
	}
	var cfg Config
	if err := econf.UnmarshalKey("pool", &cfg); err != nil {
		panic(err)
	}
	p, err := pool.NewOnDemandBlockTaskPool(cfg.InitGo, cfg.QueueSize,
		pool.WithQueueBacklogRate(cfg.QueueBacklogRate),
		pool.WithMaxIdleTime(cfg.MaxIdleTime),
		pool.WithCoreGo(cfg.CoreGo),
		pool.WithMaxGo(cfg.MaxGo))
	if err != nil {
		panic(err)
	}
	err = p.Start()
	if err != nil {
		panic(err)
	}
	return p
}

func newChannel(
	templateSvc templatesvc.ChannelTemplateService,
	clients map[string]client.Client,
) channel.Channel {
	return channel.NewDispatcher(map[domain.Channel]channel.Channel{
		domain.ChannelSMS: channel.NewSMSChannel(newSMSSelectorBuilder(templateSvc, clients)),
	})
}

func newSMSSelectorBuilder(
	templateSvc templatesvc.ChannelTemplateService,
	clients map[string]client.Client,
) *sequential.SelectorBuilder {
	// 构建SMS供应商
	providers := make([]provider.Provider, 0, len(clients))
	for k := range clients {
		providers = append(providers, sms.NewSMSProvider(
			k,
			templateSvc,
			clients[k],
		))
	}
	return sequential.NewSelectorBuilder(providers)
}

func InitGrpcServer(clients map[string]client.Client) *testioc.App {
	wire.Build(
		// 基础设施
		BaseSet,

		// --- 服务构建 ---

		// 配置服务
		configSvcSet,

		// 通知服务
		notificationSvcSet,
		sendNotificationSvcSet,
		senderSvcSet,

		// 回调服务
		callbackSvcSet,

		// 提供商服务
		providerSvcSet,

		// 模板服务
		templateSvcSet,

		// 审计服务
		auditsvc.NewService,

		// 事务通知服务
		txNotificationSvcSet,

		// 调度器
		schedulerSet,

		// 额度控制服务
		quotaSvcSet,

		// GRPC服务器
		grpcapi.NewNotificationServer,
		prodioc.InitGrpc,
		prodioc.InitTasks,
		prodioc.Crons,
		wire.Struct(new(testioc.App), "*"),
	)

	return new(testioc.App)
}
