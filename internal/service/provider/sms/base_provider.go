package sms

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/service/provider/sms/client"
)

// 仅用于grpc版本兼容演示
type baseProvider struct {
	client client.Client
}

func (p *baseProvider) send(_ context.Context,
	notification domain.Notification, activeVersion *domain.ChannelTemplateVersion,
) (domain.SendResponse, error) {
	const first = 0
	resp, err := p.client.Send(client.SendReq{
		PhoneNumbers:  notification.Receivers,
		SignName:      activeVersion.Signature,
		TemplateID:    activeVersion.Providers[first].ProviderTemplateID,
		TemplateParam: notification.Template.Params,
	})
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, err)
	}

	for _, status := range resp.PhoneNumbers {
		if status.Code != "OK" {
			return domain.SendResponse{}, fmt.Errorf("%w: Code = %s, Message = %s", errs.ErrSendNotificationFailed, status.Code, status.Message)
		}
	}

	return domain.SendResponse{
		NotificationID: notification.ID,
		Status:         domain.SendStatusSucceeded,
	}, nil
}
