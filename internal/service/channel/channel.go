package channel

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
)

// Dispatcher 渠道分发器，对外伪装成Channel，作为统一入口
type Dispatcher struct {
	channels map[domain.Channel]Channel
}

func (d *Dispatcher) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	channel, ok := d.channels[notification.Channel]
	if !ok {
		return domain.SendResponse{}, fmt.Errorf("%w: %s", errs.ErrNoAvailableChannel, notification.Channel)
	}
	return channel.Send(ctx, notification)
}

// NewDispatcher 创建渠道分发器
func NewDispatcher(channels map[domain.Channel]Channel) *Dispatcher {
	return &Dispatcher{
		channels: channels,
	}
}
