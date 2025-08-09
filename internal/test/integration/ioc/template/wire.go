//go:build wireinject

package template

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/wire"
	auditevt "notification-platform/internal/event/audit"
	templateevt "notification-platform/internal/event/template"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/dao"
	auditsvc "notification-platform/internal/service/audit"
	providersvc "notification-platform/internal/service/provider/manage"
	"notification-platform/internal/service/provider/sms/client"
	templatesvc "notification-platform/internal/service/template/manage"
	testioc "notification-platform/internal/test/ioc"
)

type Service struct {
	Svc                 templatesvc.ChannelTemplateService
	Repo                repository.ChannelTemplateRepository
	AuditResultConsumer *templateevt.AuditResultConsumer
	AuditResultProducer auditevt.ResultCallbackEventProducer
}

func Init(
	providerSvc providersvc.Service,
	auditSvc auditsvc.Service,
	clients map[string]client.Client,
	producer *kafka.Producer,
	consumer *kafka.Consumer,
	batchSize int,
	batchTimeout time.Duration,
) (*Service, error) {
	wire.Build(
		testioc.BaseSet,
		templatesvc.NewChannelTemplateService,
		repository.NewChannelTemplateRepository,
		dao.NewChannelTemplateDAO,

		templateevt.NewAuditResultConsumer,

		auditevt.NewResultCallbackEventProducer,

		wire.Struct(new(Service), "*"),
	)
	return nil, nil
}
