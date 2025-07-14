package tracing

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"net"
	"strings"
	"time"
)

const (
	// 用于Redis追踪的仪器名称
	instrumentationName = "internal/pkg/redis/tracing"
)

// Hook 实现了 redis.Hook 接口，为所有 Redis 操作添加 OpenTelemetry 追踪
type Hook struct {
	// 可选的追踪器，如果为nil则使用全局追踪器
	tracer trace.Tracer
}

// NewTracingHook 创建一个新的 Redis 追踪钩子
func NewTracingHook() *Hook {
	return &Hook{
		tracer: otel.GetTracerProvider().Tracer(instrumentationName),
	}
}

// ProcessHook 处理Redis命令的追踪
func (h *Hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		cmdName := cmd.Name()
		spanName := fmt.Sprintf("Redis %s", cmdName)

		opCtx, span := h.tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindClient),
		)

		// 设置基本属性
		attributes := []attribute.KeyValue{
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", cmdName),
		}

		// 添加命令参数（可能需要限制长度或敏感信息）
		cmdArgs := cmd.Args()
		if len(cmdArgs) > 0 {
			// 将参数转为字符串
			args := make([]string, len(cmdArgs))
			for i, arg := range cmdArgs {
				if arg != nil {
					args[i] = fmt.Sprintf("%v", arg)
				} else {
					args[i] = "<nil>"
				}
			}
			attributes = append(attributes, attribute.String("db.statement", strings.Join(args, " ")))
		}

		span.SetAttributes(attributes...)

		// 执行Redis命令
		err := next(opCtx, cmd)

		// 处理错误
		if err != nil && !errors.Is(err, redis.Nil) {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}
		const lenResult = 100

		// 添加命令结果（可能需要限制长度或敏感信息）
		if err == nil {
			result := cmd.String()
			// 避免存储过大的结果
			if len(result) > lenResult {
				result = result[:100] + "... (truncated)"
			}
			span.SetAttributes(attribute.String("db.result", result))
		}

		span.End()
		return err
	}
}

// ProcessPipelineHook 处理Redis管道命令的追踪
func (h *Hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if len(cmds) == 0 {
			return next(ctx, cmds)
		}

		cmdNames := make([]string, len(cmds))
		for i, cmd := range cmds {
			cmdNames[i] = cmd.Name()
		}

		spanName := fmt.Sprintf("Redis Pipeline (%d commands)", len(cmds))
		opCtx, span := h.tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindClient),
		)

		// 设置基本属性
		attributes := []attribute.KeyValue{
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "pipeline"),
			attribute.Int("db.redis.num_commands", len(cmds)),
			attribute.String("db.redis.commands", strings.Join(cmdNames, ", ")),
		}

		span.SetAttributes(attributes...)

		startTime := time.Now()
		err := next(opCtx, cmds)
		duration := time.Since(startTime)

		// 处理错误
		var hasError bool
		for _, cmd := range cmds {
			if cmdErr := cmd.Err(); cmdErr != nil && !errors.Is(cmdErr, redis.Nil) {
				hasError = true
				span.RecordError(cmdErr)
			}
		}

		if hasError || err != nil {
			if err != nil {
				span.RecordError(err)
			}
			span.SetStatus(codes.Error, "Pipeline had errors")
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// 执行时长
		span.SetAttributes(attribute.Float64("db.redis.pipeline_duration_ms", float64(duration.Milliseconds())))

		span.End()
		return err
	}
}

// DialHook 处理Redis连接的追踪
func (h *Hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		spanName := fmt.Sprintf("Redis CONNECT %s", addr)
		opCtx, span := h.tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindClient),
		)

		// 设置基本属性
		span.SetAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "connect"),
			attribute.String("net.transport", "tcp"),
			attribute.String("net.peer.name", addr),
		)

		conn, err := next(opCtx, network, addr)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		span.End()
		return conn, err
	}
}

// WithTracing 为Redis客户端添加追踪功能
func WithTracing(client *redis.Client) *redis.Client {
	client.AddHook(NewTracingHook())
	return client
}
