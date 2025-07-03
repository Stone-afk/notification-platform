package sms

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
	"notification-platform/internal/service/provider/sms/client"
	"notification-platform/internal/service/template/manage"
)

// smsProviderV2 SMS供应商
// 演示 grpc 版本兼容
type smsProviderV2 struct {
	name        string
	templateSvc manage.ChannelTemplateService
	baseProvider
}

// NewSmsProviderV2 SMS供应商
func NewSmsProviderV2(name string, templateSvc manage.ChannelTemplateService, client client.Client) provider.Provider {
	return &smsProviderV2{
		name:        name,
		templateSvc: templateSvc,
		baseProvider: baseProvider{
			client: client,
		},
	}
}

// Send 发送短信
func (p *smsProviderV2) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	// tmpl, err := p.templateSvc.GetTemplateV1(ctx,
	//	notification.Template.ID,
	//	notification.Template.Version,
	//	p.name, domain.ChannelSMS)
	//
	// if err != nil {
	//	return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, err)
	// }
	//
	// activeVersion := tmpl.ActiveVersion()
	// if activeVersion == nil {
	//	return domain.SendResponse{}, fmt.Errorf("%w: 无已发布模版", errs.ErrSendNotificationFailed)
	// }
	return p.send(ctx, notification, &domain.ChannelTemplateVersion{})
}
