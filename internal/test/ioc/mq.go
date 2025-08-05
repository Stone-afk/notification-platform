package ioc

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/mq-api/memory"
	"notification-platform/internal/event/failover"
	"sync"
	"time"
)

var (
	q          mq.MQ
	mqInitOnce sync.Once
)

const (
	maxInterval = 10 * time.Second
	maxRetries  = 10
	number1     = 1
)

func InitMQ() mq.MQ {
	mqInitOnce.Do(func() {
		strategy, err := retry.NewExponentialBackoffRetryStrategy(time.Second, maxInterval, maxRetries)
		if err != nil {
			panic(err)
		}
		for {
			q, err = initMQ()
			if err == nil {
				break
			}
			next, ok := strategy.Next()
			if !ok {
				panic("InitMQ 重试失败......")
			}
			time.Sleep(next)
		}
	})
	return q
}

func initMQ() (mq.MQ, error) {
	type Topic struct {
		Name       string `yaml:"name"`
		Partitions int    `yaml:"partitions"`
	}

	topics := []Topic{
		{
			Name:       "test",
			Partitions: number1,
		},
		{
			Name:       "audit_result_events",
			Partitions: number1,
		},
	}
	// 替换用内存实现，方便测试
	qq := memory.NewMQ()
	for _, t := range topics {
		err := qq.CreateTopic(context.Background(), t.Name, t.Partitions)
		if err != nil {
			return nil, err
		}
	}
	return qq, nil
}

func InitTopic() {
	topics := []kafka.TopicSpecification{
		{
			Topic:         failover.FailoverTopic,
			NumPartitions: number1,
		},
	}
	initTopic(topics...)
}

func InitProducer(id string) *kafka.Producer {
	// 初始化生产者
	config := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         id,
	}

	// 2. 创建生产者实例
	producer, err := kafka.NewProducer(config)
	if err != nil {
		panic(fmt.Sprintf("创建生产者失败: %v", err))
	}
	return producer
}

func initTopic(topics ...kafka.TopicSpecification) {
	// 创建 AdminClient
	const kafkaAddr = "localhost:9092"
	const serverName = "bootstrap.servers"
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		serverName: kafkaAddr,
	})
	if err != nil {
		panic(fmt.Sprintf("创建kafka连接失败: %v", err))
	}
	defer adminClient.Close()
	// 设置要创建的主题的配置信息
	ctx, cancel := context.WithTimeout(context.Background(), maxInterval)
	defer cancel()
	// 创建主题
	results, err := adminClient.CreateTopics(
		ctx,
		topics,
	)
	if err != nil {
		panic(fmt.Sprintf("创建topic失败: %v", err))
	}

	// 处理创建主题的结果
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrTopicAlreadyExists {
			fmt.Printf("创建topic失败 %s: %v\n", result.Topic, result.Error)
		} else {
			fmt.Printf("topic %s 创建成功\n", result.Topic)
		}
	}
}
