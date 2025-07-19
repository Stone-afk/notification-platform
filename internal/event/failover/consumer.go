package failover

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ecodeclub/mq-api"
	"github.com/gotomicro/ego/core/elog"
	"gorm.io/gorm"
	"notification-platform/internal/pkg/gormx/monitor"
	"time"
)

const (
	defaultSleepTime = 2 * time.Second
	poll             = 1000
)

type ConnPoolEventConsumer struct {
	consumer  *kafka.Consumer
	db        *gorm.DB
	dbMonitor monitor.DBMonitor
	logger    *elog.Component

	topic string
}

// Consume 处理单个消息
func (c *ConnPoolEventConsumer) Consume(ctx context.Context) error {
	// 检查数据库健康状态
	if !c.dbMonitor.Health() {
		// 获取当前消费者分配的分区
		assigned, err := c.consumer.Assignment()
		if err != nil {
			return fmt.Errorf("获取消费者已分配分区失败: %w", err)
		}
		// 如果有分配的分区，暂停它们
		if len(assigned) > 0 {
			if err := c.consumer.Pause(assigned); err != nil {
				return fmt.Errorf("暂停分区失败: %w", err)
			}
			c.logger.Info("已暂停消费者分配的分区", elog.Int("分区数量", len(assigned)))

			// 等待2秒
			time.Sleep(defaultSleepTime)

			// 恢复分区，即使数据库仍然不健康也恢复分区
			// 因为下一次消费循环会再次检查并暂停
			if err = c.consumer.Resume(assigned); err != nil {
				return fmt.Errorf("恢复分区失败: %w", err)
			}
			c.logger.Info("已恢复消费者分配的分区", elog.Int("分区数量", len(assigned)))
		}

		return nil
	}
	// 数据库健康，获取并处理消息
	ev := c.consumer.Poll(poll)
	if ev == nil {
		return nil // 没有可用消息，稍后重试
	}

	switch e := ev.(type) {
	case *kafka.Message:
		// 处理消息
		msg := &mq.Message{
			Topic:     *e.TopicPartition.Topic,
			Partition: int64(e.TopicPartition.Partition),
			Offset:    int64(e.TopicPartition.Offset),
			Key:       e.Key,
			Value:     e.Value,
		}

		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("处理消息失败",
				elog.FieldErr(err),
				elog.String("主题", msg.Topic),
				elog.String("分区", fmt.Sprintf("%d", msg.Partition)),
				elog.String("偏移量", fmt.Sprintf("%d", msg.Offset)))
			return err
		}

		// 提交消息
		if _, err := c.consumer.CommitMessage(e); err != nil {
			return fmt.Errorf("提交消息失败: %w", err)
		}

	case kafka.Error:
		return fmt.Errorf("kafka错误: %w", e)
	}

	return nil
}

// processMessage 处理单个ConnPoolEvent消息
func (c *ConnPoolEventConsumer) processMessage(ctx context.Context, msg *mq.Message) error {
	// 反序列化消息
	var event ConnPoolEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("反序列化消息失败: %w", err)
	}

	c.logger.Info("正在处理ConnPoolEvent",
		elog.String("sql", event.SQL),
		elog.Any("参数", event.Args))

	// 在数据库上执行SQL
	_, err := c.db.ConnPool.ExecContext(ctx, event.SQL, event.Args...)
	if err != nil {
		return fmt.Errorf("执行事件中的SQL失败: %w", err)
	}

	return nil
}

// NewConnPoolEventConsumer 创建一个新的连接池事件消费者
func NewConnPoolEventConsumer(consumer *kafka.Consumer, db *gorm.DB, dbMonitor monitor.DBMonitor) *ConnPoolEventConsumer {
	return &ConnPoolEventConsumer{
		consumer:  consumer,
		db:        db,
		dbMonitor: dbMonitor,
		logger:    elog.DefaultLogger,
		topic:     FailoverTopic,
	}
}
