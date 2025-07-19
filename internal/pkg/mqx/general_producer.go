package mqx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"time"
)

var _ Producer[any] = (*GeneralProducer[any])(nil)

type GeneralProducer[T any] struct {
	producer *kafka.Producer
	topic    string
}

func (p *GeneralProducer[T]) Produce(ctx context.Context, evt T) error {
	data, err := json.Marshal(&evt)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	// 创建专用递送通道
	deliveryChan := make(chan kafka.Event, 1)

	// 发送消息，处理队列满的情况
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 继续执行
		}
		// kafka.PartitionAny 代表“由 Kafka 客户端决定分区”
		err = p.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
			Value:          data,
		}, deliveryChan)
		if err != nil {
			var kafkaErr kafka.Error
			ok := errors.As(err, &kafkaErr)
			if ok && kafkaErr.Code() == kafka.ErrQueueFull {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
					continue
				}
			}
			return fmt.Errorf("向topic=%s发送event失败: %w", p.topic, err)
		}

		for {
			// 刷新所有待发送消息并等待完成
			flushed := p.producer.Flush(int(time.Second))
			if flushed == 0 {
				break
			}
		}
		// 成功提交到队列，跳出循环
		break
	}
	// 等待递送报告，带上下文控制
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-deliveryChan:
		switch event := e.(type) {
		case *kafka.Message:
			if event.TopicPartition.Error != nil {
				return fmt.Errorf("消息递送失败: %w", event.TopicPartition.Error)
			}
			return nil
		case kafka.Error:
			return fmt.Errorf("kafka错误: %w", errors.New(event.Error()))
		default:
			return fmt.Errorf("未知递送事件类型: %w", errors.New(e.String()))
		}
	}
}

// Close 安全关闭生产者
func (p *GeneralProducer[T]) Close() {
	// 刷新所有待发送消息并等待完成
	for {
		flushed := p.producer.Flush(int(time.Second))
		if flushed == 0 {
			break
		}
	}
	// 关闭生产者
	p.producer.Close()
}

func NewGeneralProducer[T any](producer *kafka.Producer, topic string) (*GeneralProducer[T], error) {
	return &GeneralProducer[T]{
		producer: producer,
		topic:    topic,
	}, nil
}
