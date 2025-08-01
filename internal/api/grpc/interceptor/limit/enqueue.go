package limit

import (
	"context"
	"github.com/gotomicro/ego/core/elog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	notificationv1 "notification-platform/api/proto/gen/notification/v1"
	"notification-platform/internal/api/grpc/interceptor/jwt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	ratelimitevt "notification-platform/internal/event/ratelimit"
	"notification-platform/internal/pkg/ratelimit"
)

// EnqueueRateLimitedRequestBuilder 限流请求入队拦截器构建器
type EnqueueRateLimitedRequestBuilder struct {
	limitedKey string
	limiter    ratelimit.Limiter
	producer   ratelimitevt.RequestRateLimitedEventProducer
	logger     *elog.Component
}

func (b *EnqueueRateLimitedRequestBuilder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		limited, err := b.limiter.Limit(ctx, b.limitedKey)
		// 未限流，继续流程
		if err == nil && !limited {
			return handler(ctx, req)
		}
		// 你可以考虑记录一下 err
		// 已限流，判断是否为需要转存的请求 —— 通知写请求
		notificationCarrier, ok := req.(notificationv1.NotificationCarrier)
		if !ok {
			return nil, status.Errorf(codes.ResourceExhausted, "%s", errs.ErrRateLimited)
		}
		bizID, err2 := jwt.GetBizIDFromContext(ctx)
		if err != nil {
			b.logger.Warn("获取BizID失败",
				elog.FieldErr(err2),
				elog.Any("req", req),
				elog.Any("info", info))
			return nil, status.Errorf(codes.ResourceExhausted, "%s", errs.ErrRateLimited)
		}
		ns := notificationCarrier.GetNotifications()
		domainNotifications := make([]domain.Notification, len(ns))
		for i := range ns {
			n, err3 := domain.NewNotificationFromAPI(ns[i])
			if err3 != nil {
				b.logger.Warn("转换为domain.NotificationB失败",
					elog.FieldErr(err3),
					elog.Any("req", req),
					elog.Any("info", info))
				return nil, status.Errorf(codes.ResourceExhausted, "%s", errs.ErrRateLimited)
			}
			// 设置业务ID
			n.BizID = bizID
			domainNotifications[i] = n
		}
		// 转存MQ
		err4 := b.producer.Produce(ctx, ratelimitevt.RequestRateLimitedEvent{
			Notifications: domainNotifications,
		})
		if err4 != nil {
			// 只记录日志
			b.logger.Warn("转存限流请求失败",
				elog.FieldErr(err4),
				elog.Any("req", req),
				elog.Any("info", info))
		}
		const accepted = 101
		return nil, status.Errorf(codes.Code(accepted), "转异步处理")
	}
}

func NewEnqueueRateLimitedRequestBuilder(
	limitedKey string,
	limiter ratelimit.Limiter,
	producer ratelimitevt.RequestRateLimitedEventProducer,
) *EnqueueRateLimitedRequestBuilder {
	return &EnqueueRateLimitedRequestBuilder{
		limitedKey: limitedKey,
		limiter:    limiter,
		producer:   producer,
		logger:     elog.DefaultLogger,
	}
}
