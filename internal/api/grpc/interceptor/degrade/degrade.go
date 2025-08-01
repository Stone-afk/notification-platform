package degrade

import (
	"context"
	"github.com/go-kratos/aegis/circuitbreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Builder struct {
	breaker circuitbreaker.CircuitBreaker
	// limiter ratelimit.Limiter
}

func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// 要判定要不要降级非核心业务
		err = b.breaker.Allow()
		if err != nil {
			// 要降级
			b.breaker.MarkFailed()
			// 我要判定是不是核心业务
			// 在这边，你可以用本地缓存，缓存业务的 ID
			// req.(User).ID => 判定这是不是一个核心用户（活跃用户，SVIP）用户

			// 为了保证高性能，不是从 BizConfig 里面去读的
			if ctx.Value("Priority") != "high" {
				return nil, status.Error(codes.Unavailable, "降级非核心业务")
			}
		}
		resp, err = handler(ctx, req)
		if err != nil {
			b.breaker.MarkFailed()
			return resp, err
		}
		b.breaker.MarkSuccess()
		return resp, err
	}
}
