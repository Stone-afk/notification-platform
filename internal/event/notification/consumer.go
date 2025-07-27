package notification

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ecodeclub/ekit/mapx"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
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
	batchTimer := time.NewTimer(c.batchTimeout)
	defer batchTimer.Stop()

	processedMessages := c.collectOneBatch(ctx)
	groupedBatchNotifications := mapx.NewMultiBuiltinMap[int64, domain.Notification](c.batchSize)
	// 按分区ID将消息分组，存储每个分区的最后一条消息
	lastMessages := make(map[int32]*kafka.Message)
	for _, msg := range processedMessages {
		var evt Event
		if err := json.Unmarshal(msg.Value, &evt); err != nil {
			c.logger.Error("解析消息失败",
				elog.FieldErr(err),
				elog.Any("msg", msg))
			// 解析失败，跳过本条，继续下一轮
			continue
		}
		const first = 0
		// TODO 假设一个事件里一批次的通知的 BizID 是相同的
		notification := evt.Notifications[first]
		_ = groupedBatchNotifications.PutMany(notification.BizID, evt.Notifications...)
		lastMessages[msg.TopicPartition.Partition] = msg
	}
	// 若本批次没有任何数据直接返回
	if groupedBatchNotifications.Len() == 0 {
		return nil
	}

	// 执行落库操作
	if err := c.batchSendNotificationsAsync(ctx, groupedBatchNotifications); err != nil {
		return err
	}

	// 只提交每个分区的最后一条消息
	for _, lastMsg := range lastMessages {
		if _, err := c.consumer.CommitMessage(lastMsg); err != nil {
			c.logger.Warn("提交消息失败",
				elog.FieldErr(err),
				elog.Any("partition", lastMsg.TopicPartition.Partition),
				elog.Any("offset", lastMsg.TopicPartition.Offset))
			return err
		}
	}
	return nil
}

func (c *EventConsumer) batchSendNotificationsAsync(ctx context.Context, groupedBatchNotifications *mapx.MultiMap[int64, domain.Notification]) error {
	// 走异步批量发送逻辑落库
	start := time.Now()
	for _, bizID := range groupedBatchNotifications.Keys() {
		// 同一个 BizID 通知在一个批次
		ns, _ := groupedBatchNotifications.Get(bizID)
		if _, err := c.srv.BatchSendNotificationsAsync(ctx, ns...); err != nil {
			if errors.Is(err, errs.ErrNotificationDuplicate) {
				// 上次落库成功，但后续提交消费进度失败，此次重复消费导致的数据库层面的唯一索引冲突
				continue
			}
			// 其他类型的错误，记录日志
			c.logger.Warn("走异步批量发送逻辑落库失败",
				elog.FieldErr(err),
				elog.Int64("bizID", bizID),
				elog.Int("notifications_count", len(ns)))

			return err
		}
	}

	// 根据响应时间来计算新的batchSize
	newBatchSize, err := c.batchSizeAdjuster.Adjust(ctx, time.Since(start))
	if err == nil {
		c.batchSize = newBatchSize
	}
	return nil
}

func (c *EventConsumer) collectOneBatch(ctx context.Context) []*kafka.Message {
	ctx, cancel := context.WithTimeout(ctx, c.batchTimeout)
	defer cancel()
	// 我们认为大部分都是只有一条通知的
	processedMessages := make([]*kafka.Message, 0, c.batchSize)
	curBatchSize := 0
	for {
		select {
		case <-ctx.Done():
			return processedMessages
		default:
			// 达到批量大小
			if curBatchSize >= c.batchSize {
				return processedMessages
			}
		}
		// 剩余的超时时间，严格计算就是用剩余超时时间
		ddl, _ := ctx.Deadline()
		timeout := time.Until(ddl)
		msg, err := c.consumer.ReadMessage(timeout)
		if err != nil {
			var kErr kafka.Error
			if errors.As(err, &kErr) && kErr.Code() == kafka.ErrTimedOut {
				// 聚合当前批次已超时
				return processedMessages
			}
			c.logger.Error("从 consumer 中读取消息失败", elog.FieldErr(err))
			return processedMessages
		}
		curBatchSize++
		processedMessages = append(processedMessages, msg)
	}
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
