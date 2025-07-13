package sender

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"notification-platform/internal/domain"
	"strconv"
	"strings"
)

// TracingSender 为通知发送添加链路追踪的装饰器
type TracingSender struct {
	sender NotificationSender
	tracer trace.Tracer
}

func (s *TracingSender) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	ctx, span := s.tracer.Start(ctx, "NotificationSender.Send",
		trace.WithAttributes(
			attribute.String("notification.id", strconv.FormatUint(notification.ID, 10)),
			attribute.String("notification.bizId", strconv.FormatInt(notification.BizID, 10)),
			attribute.String("notification.key", notification.Key),
			attribute.String("notification.channel", string(notification.Channel)),
		))
	defer span.End()

	response, err := s.sender.Send(ctx, notification)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(
			attribute.String("notification.id", strconv.FormatUint(response.NotificationID, 10)),
			attribute.String("notification.status", string(response.Status)),
		)
	}

	return response, err
}

func (s *TracingSender) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	ctx, span := s.tracer.Start(ctx, "NotificationSender.BatchSend",
		trace.WithAttributes(
			attribute.Int("notification.count", len(notifications)),
		))
	defer span.End()

	// 提取所有通知的关键属性，作为属性记录
	if len(notifications) > 0 {
		span.SetAttributes(
			attribute.String("notification.bizId", strconv.FormatInt(notifications[0].BizID, 10)),
			attribute.String("notification.keys", strings.Join(slice.Map(notifications, func(_ int, src domain.Notification) string {
				return src.Key
			}), ",")),
		)
	}

	responses, err := s.sender.BatchSend(ctx, notifications)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		// 记录成功和失败的数量
		var succeeded, failed int
		for _, resp := range responses {
			if resp.Status == domain.SendStatusSucceeded {
				succeeded++
			} else {
				failed++
			}
		}
		span.SetAttributes(
			attribute.Int("notification.succeeded", succeeded),
			attribute.Int("notification.failed", failed),
		)
	}

	return responses, err
}

// NewTracingSender 创建一个新的带有链路追踪的发送器
func NewTracingSender(sender NotificationSender) *TracingSender {
	return &TracingSender{
		sender: sender,
		tracer: otel.Tracer("notification-platform/sender"),
	}
}
