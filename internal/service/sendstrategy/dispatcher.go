package sendstrategy

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
)

// Dispatcher 通知发送分发器
// 根据通知的策略类型选择合适的发送策略
type Dispatcher struct {
	immediate       *ImmediateSendStrategy
	defaultStrategy *DefaultSendStrategy
}

func (d *Dispatcher) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	// 执行发送
	return d.selectStrategy(notification).Send(ctx, notification)
}

func (d *Dispatcher) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}
	const first = 0
	// 执行发送
	return d.selectStrategy(notifications[first]).BatchSend(ctx, notifications)
}

func (d *Dispatcher) selectStrategy(not domain.Notification) SendStrategy {
	if not.SendStrategyConfig.Type == domain.SendStrategyImmediate {
		return d.immediate
	}
	return d.defaultStrategy
}

// NewDispatcher 创建通知发送分发器
func NewDispatcher(
	immediate *ImmediateSendStrategy,
	defaultStrategy *DefaultSendStrategy,
) SendStrategy {
	return &Dispatcher{
		immediate:       immediate,
		defaultStrategy: defaultStrategy,
	}
}
