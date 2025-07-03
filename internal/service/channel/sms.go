package channel

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/service/provider"
)

type baseChannel struct {
	builder provider.SelectorBuilder
}

func (s *baseChannel) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	selector, err := s.builder.Build()
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, err)
	}

	for {
		// 获取供应商
		p, err1 := selector.Next(ctx, notification)
		if err1 != nil {
			// 没有可用的供应商
			return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, err1)
		}

		// 使用当前供应商发送
		resp, err2 := p.Send(ctx, notification)
		if err2 == nil {
			return resp, nil
		}
	}
}

type smsChannel struct {
	baseChannel
}

func NewSMSChannel(builder provider.SelectorBuilder) Channel {
	return &smsChannel{
		baseChannel{
			builder: builder,
		},
	}
}
