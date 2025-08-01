package log

import (
	"context"
	"encoding/json"
	"github.com/gotomicro/ego/core/elog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

// Builder 日志拦截器构建器
type Builder struct {
	logger *elog.Component
}

// WithLogger 设置日志组件
func (b *Builder) WithLogger(logger *elog.Component) *Builder {
	b.logger = logger
	return b
}

// Build 构建 gRPC 一元拦截器
func (b *Builder) Build() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 记录开始时间
		startTime := time.Now()
		// 将请求对象转为 JSON 字符串进行记录
		reqJSON, _ := json.Marshal(req)
		b.logger.Info("gRPC request",
			elog.String("method", info.FullMethod),
			elog.String("request", string(reqJSON)),
			elog.Any("start_time", startTime))

		// 处理请求
		resp, err := handler(ctx, req)

		// 计算请求处理时间
		duration := time.Since(startTime)

		// 获取状态码
		st, _ := status.FromError(err)
		statusCode := st.Code()

		// 将响应对象转为 JSON 字符串进行记录
		respJSON, _ := json.Marshal(resp)

		if err != nil {
			// 如果有错误，记录错误日志
			b.logger.Error("gRPC response with error",
				elog.String("method", info.FullMethod),
				elog.String("status_code", statusCode.String()),
				elog.String("response", string(respJSON)),
				elog.Duration("duration", duration),
				elog.FieldErr(err))
		} else {
			// 记录成功响应日志
			b.logger.Info("gRPC response",
				elog.String("method", info.FullMethod),
				elog.String("status_code", codes.OK.String()),
				elog.String("response", string(respJSON)),
				elog.Duration("duration", duration))
		}
		return resp, err
	}
}

// NewLoggerBuilder 创建日志拦截器构建器
func NewLoggerBuilder() *Builder {
	return &Builder{
		logger: elog.DefaultLogger,
	}
}
