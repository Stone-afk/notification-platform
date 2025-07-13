package tracing

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
	"strconv"
)

// Provider 为供应商实现添加链路追踪的装饰器
type Provider struct {
	provider provider.Provider
	tracer   trace.Tracer
	name     string
}

func (p *Provider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	ctx, span := p.tracer.Start(ctx, "Provider.Send",
		trace.WithAttributes(
			attribute.String("provider.name", p.name),
			attribute.String("notification.id", strconv.FormatUint(notification.ID, 10)),
			attribute.String("notification.bizId", strconv.FormatInt(notification.BizID, 10)),
			attribute.String("notification.key", notification.Key),
			attribute.String("notification.channel", notification.Channel.String()),
		))
	defer span.End()

	// 调用底层供应商发送通知
	response, err := p.provider.Send(ctx, notification)

	if err != nil {
		// 记录错误信息
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		// 记录成功响应的属性
		span.SetAttributes(
			attribute.String("notification.id", strconv.FormatUint(response.NotificationID, 10)),
			attribute.String("notification.status", response.Status.String()),
		)
	}

	return response, err
}

// NewProvider 创建一个新的带有链路追踪的供应商
// name 应该传入类似于 tencent, ali 这种名字
func NewProvider(p provider.Provider, name string) *Provider {
	return &Provider{
		provider: p,
		name:     name,
		tracer:   otel.Tracer("notification-platform/provider"),
	}
}
