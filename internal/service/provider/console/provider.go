package console

import (
	"context"

	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
)

// 你可以在这里提供一个 provider 输出到控制台的实现

type Provider struct {
	logger *elog.Component
}

func NewProvider() *Provider {
	return &Provider{
		logger: elog.DefaultLogger,
	}
}

func (p *Provider) Send(_ context.Context, notification domain.Notification) (domain.SendResponse, error) {
	p.logger.Info("发送通知", elog.Any("notification", notification))
	return domain.SendResponse{Status: domain.SendStatusSucceeded}, nil
}
