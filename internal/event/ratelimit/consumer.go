package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/pkg/mqx"
	"notification-platform/internal/pkg/ratelimit"
	notificationsvc "notification-platform/internal/service/notification"
	"time"
)

const (
	defaultPollInterval = time.Second
)

type RequestRateLimitedEventConsumer struct {
	srv      notificationsvc.SendNotificationService
	consumer mqx.Consumer

	limiter          ratelimit.Limiter
	limitedKey       string
	lookbackDuration time.Duration
	sleepDuration    time.Duration

	logger *elog.Component
}

func (c *RequestRateLimitedEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费限流请求事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *RequestRateLimitedEventConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.ReadMessage(-1)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	// 等待限流期结束
	if err2 := c.waitUntilRateLimitExpires(ctx); err2 != nil {
		return err2
	}

	var evt RequestRateLimitedEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		c.logger.Warn("解析消息失败",
			elog.FieldErr(err),
			elog.Any("msg", msg))
		return err
	}

	// 不管通知的原始发送策略是什么，经过MQ转存后，一律强转为默认的截止日期前发送，等地异步任务调度并发送
	_, err = c.srv.BatchSendNotificationsAsync(ctx, evt.Notifications...)
	if err != nil {
		c.logger.Warn("处理限流请求事件失败",
			elog.FieldErr(err),
			elog.Any("evt", evt))
	}

	// 消费完成，提交消费进度
	_, err = c.consumer.CommitMessage(msg)
	if err != nil {
		c.logger.Warn("提交消息失败",
			elog.FieldErr(err),
			elog.Any("msg", msg))
		return err
	}
	return nil
}

func (c *RequestRateLimitedEventConsumer) waitUntilRateLimitExpires(ctx context.Context) error {
	for {
		// 是否发送过限流
		lastLimitTime, err1 := c.limiter.LastLimitTime(ctx, c.limitedKey)
		if err1 != nil {
			c.logger.Warn("获取限流状态失败",
				elog.FieldErr(err1),
				elog.Any("limitedKey", c.limitedKey))
			return err1
		}

		// 未发生限流，或者最近一次发生限流的时间不在预期时间段内
		if lastLimitTime.IsZero() || time.Since(lastLimitTime) > c.lookbackDuration {
			return nil
		}

		// 获取分配的分区
		partitions, err2 := c.consumer.Assignment()
		if err2 != nil {
			c.logger.Warn("获取消费者已分配的分区失败",
				elog.FieldErr(err2),
				elog.Any("partitions", partitions))
			return err2
		}

		// 发生过限流，睡眠一段时间，醒了继续判断是否被限流
		// 暂停分区消费
		err3 := c.consumer.Pause(partitions)
		if err3 != nil {
			c.logger.Warn("暂停分区失败",
				elog.FieldErr(err3),
				elog.Any("partitions", partitions))
			return err3
		}

		// 睡眠
		c.sleepAndPoll(c.sleepDuration)

		// 恢复分区消费
		err4 := c.consumer.Resume(partitions)
		if err4 != nil {
			c.logger.Warn("恢复分区失败",
				elog.FieldErr(err4),
				elog.Any("partitions", partitions))
			return err4
		}
	}
}

// 这是一个 “在一定时间内持续轮询消费消息”的函数。它的核心目的是在 subTime（订阅/等待时间）内，以固定间隔不断调用 Kafka 的 Poll 方法。
/*
背景用途（典型场景）
	这个逻辑常用于 限流处理期间，仍保持消费 Kafka 消息避免 backlog 堆积，但处理流程暂时 sleep：
	比如你有个消费逻辑做了限速（例如超过 1000 QPS 暂停处理），
	但你不能让 Kafka 消费端完全 idle（否则会导致分区 rebalance 或 backlog 积压），
	就用这个 sleepAndPoll() 方法在这段期间继续 “消费 + 丢弃” 或 “仅轮询”，确保不会超时。
*/
func (c *RequestRateLimitedEventConsumer) sleepAndPoll(subTime time.Duration) {
	const defaultPollDuration = 100
	/*
		ticker：每隔一段时间（应该是 defaultPollInterval，触发一次，用于定期调用 Poll。
		timer：用来控制整个循环的运行时长，超时后退出循环。
	*/
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()
	// defaultPollInterval := 200 * time.Millisecond， subTime := 2 * time.Second
	// 则意味着：在 2 秒内，每隔 200ms 调一次 Poll(100)。
	timer := time.NewTimer(subTime)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return
		case <-ticker.C:
			// Kafka 的 Poll 接口需要一个 timeout，单位是 毫秒。
			// 每次轮询 Kafka 消息时最多阻塞 100ms。
			c.consumer.Poll(defaultPollDuration)
		}
	}
}

func NewRequestLimitedEventConsumerWithTopic(
	srv notificationsvc.SendNotificationService,
	consumer *kafka.Consumer,
	limitedKey string,
	limiter ratelimit.Limiter,
	lookbackDuration time.Duration,
	sleepDuration time.Duration,
	topic string,
) (*RequestRateLimitedEventConsumer, error) {
	err := consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		return nil, err
	}

	return &RequestRateLimitedEventConsumer{
		srv:              srv,
		consumer:         consumer,
		limitedKey:       limitedKey,
		limiter:          limiter,
		lookbackDuration: lookbackDuration,
		sleepDuration:    sleepDuration,
		logger:           elog.DefaultLogger,
	}, nil
}

func NewRequestLimitedEventConsumer(
	srv notificationsvc.SendNotificationService,
	consumer *kafka.Consumer,
	limitedKey string,
	limiter ratelimit.Limiter,
	lookbackDuration time.Duration,
	sleepDuration time.Duration,
) (*RequestRateLimitedEventConsumer, error) {
	return NewRequestLimitedEventConsumerWithTopic(
		srv,
		consumer,
		limitedKey,
		limiter,
		lookbackDuration,
		sleepDuration,
		eventName,
	)
}
