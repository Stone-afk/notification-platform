package sender

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"notification-platform/internal/domain"
	"time"
)

const (
	// Prometheus metrics constants
	metricsMaxAge        = 5 * time.Minute
	metricsP50Percentile = 0.5
	metricsP50Error      = 0.05
	metricsP90Percentile = 0.9
	metricsP90Error      = 0.01
	metricsP95Percentile = 0.95
	metricsP95Error      = 0.005
	metricsP99Percentile = 0.99
	metricsP99Error      = 0.001

	// Special tags for metrics
	metricsBatchTag = "batch"
)

// MetricsSender 为通知发送添加指标收集的装饰器
type MetricsSender struct {
	sender                 NotificationSender
	sendDurationSummary    *prometheus.SummaryVec
	sendCounter            *prometheus.CounterVec
	batchSendCounter       *prometheus.CounterVec
	notificationSentStatus *prometheus.CounterVec
}

// Send 发送单条通知并记录指标
func (m *MetricsSender) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	// 开始计时
	startTime := time.Now()

	// 累加发送计数
	m.sendCounter.WithLabelValues(notification.Channel.String()).Inc()

	// 调用实际的发送方法
	response, err := m.sender.Send(ctx, notification)

	// 计算耗时
	duration := time.Since(startTime).Seconds()

	// 记录发送状态
	m.notificationSentStatus.WithLabelValues(
		notification.Channel.String(),
		response.Status.String(),
	).Inc()

	// 记录耗时
	m.sendDurationSummary.WithLabelValues(
		notification.Channel.String(),
		response.Status.String(),
	).Observe(duration)

	return response, err
}

// BatchSend 批量发送通知并记录指标
func (m *MetricsSender) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, nil
	}

	// 开始计时
	startTime := time.Now()

	// 获取通知的渠道（假设所有通知都是同一渠道）
	channel := notifications[0].Channel.String()

	// 累加批量发送计数
	m.batchSendCounter.WithLabelValues(channel).Inc()

	// 调用实际的批量发送方法
	responses, err := m.sender.BatchSend(ctx, notifications)

	// 记录各状态的数量
	if err == nil && len(responses) > 0 {
		var succeeded, failed int
		for _, resp := range responses {
			if resp.Status == domain.SendStatusSucceeded {
				succeeded++
			} else {
				failed++
			}
		}

		// 记录成功和失败的通知数量
		m.notificationSentStatus.WithLabelValues(channel, domain.SendStatusSucceeded.String()).Add(float64(succeeded))
		m.notificationSentStatus.WithLabelValues(channel, domain.SendStatusFailed.String()).Add(float64(failed))
	}

	// 计算耗时
	duration := time.Since(startTime).Seconds()

	// 记录平均耗时（每条通知）
	m.sendDurationSummary.WithLabelValues(
		channel,
		metricsBatchTag, // 使用特殊标签来标识批量发送的耗时
	).Observe(duration / float64(len(notifications)))

	return responses, err
}

// NewMetricsSender 创建一个新的带有指标收集的发送器
func NewMetricsSender(sender NotificationSender) *MetricsSender {
	sendDurationSummary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "notification_send_duration_seconds",
			Help:       "通知发送耗时统计（秒）",
			Objectives: map[float64]float64{metricsP50Percentile: metricsP50Error, metricsP90Percentile: metricsP90Error, metricsP95Percentile: metricsP95Error, metricsP99Percentile: metricsP99Error},
			MaxAge:     metricsMaxAge,
		},
		[]string{"channel", "status"},
	)

	sendCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_send_total",
			Help: "通知发送总数",
		},
		[]string{"channel"},
	)

	batchSendCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_batch_send_total",
			Help: "批量通知发送总数",
		},
		[]string{"channel"},
	)

	notificationSentStatus := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "notification_sent_status_total",
			Help: "通知发送状态统计",
		},
		[]string{"channel", "status"},
	)

	// 注册指标
	prometheus.MustRegister(sendDurationSummary, sendCounter, batchSendCounter, notificationSentStatus)

	return &MetricsSender{
		sender:                 sender,
		sendDurationSummary:    sendDurationSummary,
		sendCounter:            sendCounter,
		batchSendCounter:       batchSendCounter,
		notificationSentStatus: notificationSentStatus,
	}
}
