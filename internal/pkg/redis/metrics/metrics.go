package metrics

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"net"
	"time"
)

// Metric quantile constants
const (
	quantileP50 = 0.5
	quantileP90 = 0.9
	quantileP95 = 0.95
	quantileP99 = 0.99

	errorP50 = 0.05
	errorP90 = 0.01
	errorP95 = 0.005
	errorP99 = 0.001
)

// Status constants
const (
	statusSuccess = "success"
	statusError   = "error"
)

var (
	// Redis命令计数器
	commandCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_commands_total",
			Help: "Total number of Redis commands executed",
		},
		[]string{"command", "status"},
	)

	// Redis命令执行时间
	commandDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "redis_command_duration_seconds",
			Help:       "Redis command execution time in seconds",
			Objectives: map[float64]float64{quantileP50: errorP50, quantileP90: errorP90, quantileP95: errorP95, quantileP99: errorP99},
		},
		[]string{"command"},
	)

	// Redis管道命令计数器
	pipelineCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_pipeline_commands_total",
			Help: "Total number of Redis pipeline executions",
		},
		[]string{"status"},
	)

	// Redis管道命令总数
	pipelineCommandsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_pipeline_command_count_total",
			Help: "Total number of commands in Redis pipelines",
		},
	)

	// Redis管道执行时间
	pipelineDuration = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:       "redis_pipeline_duration_seconds",
			Help:       "Redis pipeline execution time in seconds",
			Objectives: map[float64]float64{quantileP50: errorP50, quantileP90: errorP90, quantileP95: errorP95, quantileP99: errorP99},
		},
	)

	// Redis连接计数器
	connectionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_connections_total",
			Help: "Total number of Redis connections created",
		},
		[]string{"status"},
	)
)

func init() {
	// 注册所有指标
	prometheus.MustRegister(
		commandCounter,
		commandDuration,
		pipelineCounter,
		pipelineCommandsCounter,
		pipelineDuration,
		connectionCounter,
	)
}

// Hook 实现了 redis.Hook 接口，为所有 Redis 操作添加指标收集
type Hook struct{}

// ProcessHook 处理Redis命令的指标收集
func (h *Hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		cmdName := cmd.Name()

		// 记录执行开始时间
		startTime := time.Now()

		// 执行Redis命令
		err := next(ctx, cmd)

		// 计算执行时间
		duration := time.Since(startTime)

		// 记录命令执行时间
		commandDuration.WithLabelValues(cmdName).Observe(duration.Seconds())

		// 记录命令执行状态
		status := statusSuccess
		if err != nil && !errors.Is(err, redis.Nil) {
			status = statusError
		}

		// 增加命令计数
		commandCounter.WithLabelValues(cmdName, status).Inc()

		return err
	}
}

// ProcessPipelineHook 处理Redis管道命令的指标收集
func (h *Hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if len(cmds) == 0 {
			return next(ctx, cmds)
		}

		// 记录执行开始时间
		startTime := time.Now()

		// 执行Redis管道命令
		err := next(ctx, cmds)

		// 计算执行时间
		duration := time.Since(startTime)

		// 记录管道执行时间
		pipelineDuration.Observe(duration.Seconds())

		// 记录管道命令数量
		pipelineCommandsCounter.Add(float64(len(cmds)))

		// 检查是否有错误
		status := statusSuccess
		for _, cmd := range cmds {
			if cmdErr := cmd.Err(); cmdErr != nil && !errors.Is(cmdErr, redis.Nil) {
				status = statusError
				break
			}
		}

		if status == statusSuccess && err != nil {
			status = statusError
		}

		// 增加管道计数
		pipelineCounter.WithLabelValues(status).Inc()

		return err
	}
}

// DialHook 处理Redis连接的指标收集
func (h *Hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// 执行连接操作
		conn, err := next(ctx, network, addr)

		// 记录连接状态
		status := statusSuccess
		if err != nil {
			status = statusError
		}

		// 增加连接计数
		connectionCounter.WithLabelValues(status).Inc()

		return conn, err
	}
}

// NewMetricsHook 创建一个新的 Redis 指标收集钩子
func NewMetricsHook() *Hook {
	return &Hook{}
}

// WithMetrics 为Redis客户端添加指标收集功能
func WithMetrics(client *redis.Client) *redis.Client {
	client.AddHook(NewMetricsHook())
	return client
}
