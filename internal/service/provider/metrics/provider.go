package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
	"time"
)

// 定义Prometheus指标配置常量
const (
	// 摘要指标的分位数配置
	median = 0.5
	p90    = 0.9
	p95    = 0.95
	p99    = 0.99

	medianError = 0.05
	p90Error    = 0.01
	p95Error    = 0.005
	p99Error    = 0.001

	// 摘要指标的最大保留时间
	maxAgeDuration = 5 * time.Minute
)

// Provider 为供应商实现添加指标收集的装饰器
type Provider struct {
	provider            provider.Provider
	sendDurationSummary *prometheus.SummaryVec
	sendCounter         *prometheus.CounterVec
	sendStatusCounter   *prometheus.CounterVec
	name                string
}

// Send 发送通知并记录指标
func (p *Provider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	// 开始计时
	startTime := time.Now()

	// 累加发送计数
	p.sendCounter.WithLabelValues(
		p.name,
		string(notification.Channel),
	).Inc()

	// 调用底层供应商发送通知
	response, err := p.provider.Send(ctx, notification)

	// 计算耗时
	duration := time.Since(startTime).Seconds()

	// 记录发送状态
	p.sendStatusCounter.WithLabelValues(
		p.name,
		string(notification.Channel),
		string(response.Status),
	).Inc()

	// 记录耗时
	p.sendDurationSummary.WithLabelValues(
		p.name,
		string(notification.Channel),
		string(response.Status),
	).Observe(duration)

	return response, err
}

// NewProvider 创建一个新的带有指标收集的供应商
func NewProvider(name string, p provider.Provider) *Provider {
	sendDurationSummary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "provider_send_duration_seconds",
			Help: "供应商发送通知耗时统计（秒）",
			Objectives: map[float64]float64{
				median: medianError,
				p90:    p90Error,
				p95:    p95Error,
				p99:    p99Error,
			},
			MaxAge: maxAgeDuration,
		},
		[]string{"provider", "channel", "status"},
	)

	sendCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "provider_send_total",
			Help: "供应商发送通知总数",
		},
		[]string{"provider", "channel"},
	)

	sendStatusCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "provider_send_status_total",
			Help: "供应商发送通知状态统计",
		},
		[]string{"provider", "channel", "status"},
	)

	// 注册指标
	prometheus.MustRegister(sendDurationSummary, sendCounter, sendStatusCounter)

	return &Provider{
		provider:            p,
		sendDurationSummary: sendDurationSummary,
		sendCounter:         sendCounter,
		sendStatusCounter:   sendStatusCounter,
		name:                name,
	}
}
