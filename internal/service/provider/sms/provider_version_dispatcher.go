package sms

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
)

// ProviderDispatcher 仅用于grpc版本兼容演示
type ProviderDispatcher struct {
	ps map[string]provider.Provider
}

func (p *ProviderDispatcher) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	version, _ := ctx.Value("version").(string)
	return p.ps[version].Send(ctx, notification)
}
