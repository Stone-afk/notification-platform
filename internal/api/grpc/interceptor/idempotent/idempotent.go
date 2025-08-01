package idempotent

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	notificationv1 "notification-platform/api/proto/gen/notification/v1"
	"notification-platform/internal/api/grpc/interceptor/jwt"
	"notification-platform/internal/pkg/idempotent"
)

type Builder struct {
	svc idempotent.IdempotencyService
}

func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		bizID, err := jwt.GetBizIDFromContext(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		v, ok := req.(notificationv1.IdempotencyCarrier)
		if !ok {
			return handler(ctx, req)
		}
		// 需要幂等检测
		keys := v.GetIdempotencyKeys()
		keys = slice.Map(keys, func(idx int, src string) string {
			return fmt.Sprintf("%d-%s", bizID, src)
		})
		exists, err := b.svc.MExists(ctx, keys...)
		if err != nil {
			return nil, fmt.Errorf("进行幂等检测失败")
		}
		for idx := range exists {
			if exists[idx] {
				return nil, status.Errorf(codes.InvalidArgument, "%v", fmt.Errorf("幂等检测没通过，有重复请求"))
			}
		}
		return handler(ctx, req)
	}
}

func NewIdempotentBuilder(svc idempotent.IdempotencyService) *Builder {
	return &Builder{svc: svc}
}
