package notification

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/pkg/batchsize"
	"notification-platform/internal/pkg/mqx"
	"notification-platform/internal/pkg/ratelimit"
	notificationsvc "notification-platform/internal/service/notification"
	"time"
)

const (
	defaultPollInterval = time.Second
)

type EventConsumer struct {
	srv      notificationsvc.SendNotificationService
	consumer mqx.Consumer

	limiter          ratelimit.Limiter
	limitedKey       string
	lookbackDuration time.Duration
	sleepDuration    time.Duration

	batchSize         int
	batchTimeout      time.Duration
	batchSizeAdjuster batchsize.Adjuster

	logger *elog.Component
}

func (c *EventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			// 限流检测
			if err := c.waitUntilRateLimitExpires(ctx); err != nil {
				c.logger.Error("消费者限流", elog.FieldErr(err))
			}

			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费通知事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *EventConsumer) Consume(ctx context.Context) error {
	panic("not implemented") // TODO: Implement Consume method
}

func (c *EventConsumer) waitUntilRateLimitExpires(ctx context.Context) error {
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

func (c *EventConsumer) sleepAndPoll(subTime time.Duration) {
	const defaultPollDuration = 100
	ticker := time.NewTicker(defaultPollInterval)
	defer ticker.Stop()
	timer := time.NewTimer(subTime)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return
		case <-ticker.C:
			c.consumer.Poll(defaultPollDuration)
		}
	}
}

func NewEventConsumer(
	srv notificationsvc.SendNotificationService,
	consumer *kafka.Consumer,
	limitedKey string,
	limiter ratelimit.Limiter,
	lookbackDuration time.Duration,
	sleepDuration time.Duration,
	batchSize int,
	batchTimeout time.Duration,
	batchSizeAdjuster batchsize.Adjuster,
) (*EventConsumer, error) {
	return NewEventConsumerWithTopic(
		srv,
		consumer,
		limitedKey,
		limiter,
		lookbackDuration,
		sleepDuration,
		batchSize,
		batchTimeout,
		batchSizeAdjuster,
		EventName,
	)
}

func NewEventConsumerWithTopic(
	srv notificationsvc.SendNotificationService,
	consumer *kafka.Consumer,
	limitedKey string,
	limiter ratelimit.Limiter,
	lookbackDuration time.Duration,
	sleepDuration time.Duration,
	batchSize int,
	batchTimeout time.Duration,
	batchSizeAdjuster batchsize.Adjuster,
	topic string,
) (*EventConsumer, error) {
	err := consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		return nil, err
	}

	return &EventConsumer{
		srv:               srv,
		consumer:          consumer,
		limitedKey:        limitedKey,
		limiter:           limiter,
		lookbackDuration:  lookbackDuration,
		sleepDuration:     sleepDuration,
		batchSize:         batchSize,
		batchTimeout:      batchTimeout,
		batchSizeAdjuster: batchSizeAdjuster,
		logger:            elog.DefaultLogger,
	}, nil
}
