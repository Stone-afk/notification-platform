//go:build e2e

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"notification-platform/internal/domain"
	ratelimitevt "notification-platform/internal/event/ratelimit"
	limitmocks "notification-platform/internal/pkg/ratelimit/mocks"
	notificationmocks "notification-platform/internal/service/notification/mocks"
)

const (
	defaultTestKafkaAddr = "localhost:9092"
)

func TestRateLimitEventSuite(t *testing.T) {
	t.Parallel()
	t.Skip()
	suite.Run(t, new(RateLimitEventTestSuite))
}

type RateLimitEventTestSuite struct {
	suite.Suite

	// 每个测试方法使用的唯一标识
	testID string
	// 每个测试方法使用的topic名称
	testTopic string
	// Kafka地址
	kafkaAddr string
	// Kafka AdminClient
	adminClient *kafka.AdminClient
}

// SetupTest 在每个测试方法执行前运行
func (s *RateLimitEventTestSuite) SetupTest() {
	// 生成唯一的测试标识
	s.testID = fmt.Sprintf("test-%d", time.Now().UnixNano())
	// 生成唯一的测试topic名称
	s.testTopic = fmt.Sprintf("request_rate_limited_events-%s", s.testID)
	s.kafkaAddr = defaultTestKafkaAddr

	// 创建AdminClient
	var err error
	s.adminClient, err = kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": s.kafkaAddr,
	})
	s.NoError(err)

	// 创建测试topic
	s.createTestTopic(s.testTopic)
}

// TearDownTest 在每个测试方法执行后运行
func (s *RateLimitEventTestSuite) TearDownTest() {
	// 删除测试topic
	if s.adminClient != nil {
		s.deleteTestTopic(s.testTopic)
		s.adminClient.Close()
	}
}

// createTestTopic 创建测试topic
func (s *RateLimitEventTestSuite) createTestTopic(topic string) {
	numPartitions := 1
	replicationFactor := 1

	// 创建主题
	results, err := s.adminClient.CreateTopics(
		context.Background(),
		[]kafka.TopicSpecification{
			{
				Topic:             topic,
				NumPartitions:     numPartitions,
				ReplicationFactor: replicationFactor,
			},
		},
	)
	s.NoError(err)

	// 处理创建主题的结果
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrTopicAlreadyExists {
			s.Fail(fmt.Sprintf("创建topic失败 %s: %v", result.Topic, result.Error))
		} else {
			fmt.Printf("Topic %s 创建成功\n", result.Topic)
		}
	}

	// 等待topic创建完成
	time.Sleep(500 * time.Millisecond)
}

// deleteTestTopic 删除测试topic
func (s *RateLimitEventTestSuite) deleteTestTopic(topic string) {
	// 删除主题
	results, err := s.adminClient.DeleteTopics(
		context.Background(),
		[]string{topic},
	)
	s.NoError(err)

	// 处理删除主题的结果
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError {
			fmt.Printf("删除topic失败 %s: %v\n", result.Topic, result.Error)
		} else {
			fmt.Printf("Topic %s 删除成功\n", result.Topic)
		}
	}
}

// newTestComponent 创建测试组件，使用当前测试的唯一topic
func (s *RateLimitEventTestSuite) newTestComponent(ctrl *gomock.Controller) (eventProducer ratelimitevt.RequestRateLimitedEventProducer, eventConsumer *ratelimitevt.RequestRateLimitedEventConsumer, mockLimiter *limitmocks.MockLimiter, mockService *notificationmocks.MockSendService) {
	mockService = notificationmocks.NewMockSendService(ctrl)
	mockLimiter = limitmocks.NewMockLimiter(ctrl)

	// 为当前测试创建唯一的consumer group
	groupID := fmt.Sprintf("mock-ratelimit-group-%s", s.testID)

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": s.kafkaAddr,
	})
	s.NoError(err)

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  s.kafkaAddr,
		"auto.offset.reset":  "earliest",
		"group.id":           groupID,
		"enable.auto.commit": "false",
	})
	s.NoError(err)

	eventProducer, err = ratelimitevt.NewRequestRateLimitedEventProducerWithTopic(producer, s.testTopic)
	s.NoError(err)

	eventConsumer, err = ratelimitevt.NewRequestLimitedEventConsumerWithTopic(
		mockService,
		consumer,
		fmt.Sprintf("mock-ratelimit-limitKey-%s", s.testID),
		mockLimiter,
		5*time.Second, 200*time.Millisecond,
		s.testTopic,
	)
	s.NoError(err)

	return eventProducer, eventConsumer, mockLimiter, mockService
}

// createTestNotification 创建测试用的通知对象
func (s *RateLimitEventTestSuite) createTestNotification(id uint64) domain.Notification {
	now := time.Now()
	return domain.Notification{
		ID:        id,
		BizID:     123,
		Key:       "test-key",
		Receivers: []string{"user1", "user2"},
		Channel:   domain.ChannelSMS,
		Template: domain.Template{
			ID:        456,
			VersionID: 789,
			Params:    map[string]string{"param1": "value1", "param2": "value2"},
		},
		Status:         domain.SendStatusPending,
		ScheduledSTime: now,
		ScheduledETime: now.Add(time.Hour),
		Version:        1,
		SendStrategyConfig: domain.SendStrategyConfig{
			Type:         domain.SendStrategyDeadline,
			DeadlineTime: now.Add(time.Hour),
		},
	}
}

// TestProducerSuccess 测试生产者成功发送事件
func (s *RateLimitEventTestSuite) TestProducerSuccess() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, _, _, _ := s.newTestComponent(ctrl)

	// 创建测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001),
		s.createTestNotification(1002),
	}
	evt := ratelimitevt.RequestRateLimitedEvent{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err, "生产者应该能够成功发送事件")
}

// TestConsumerWithoutRateLimit 测试消费者在非限流状态下成功消费事件
func (s *RateLimitEventTestSuite) TestConsumerWithoutRateLimit() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-ratelimit-limitKey-%s", s.testID)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001),
		s.createTestNotification(1002),
	}
	evt := ratelimitevt.RequestRateLimitedEvent{
		Notifications: notifications,
	}

	// 模拟服务行为 - 成功处理通知
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{2001, 2002},
		}, nil)

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	err = consumer.Consume(ctx)
	assert.NoError(t, err)
}

// TestConsumerWithRateLimit 测试消费者在限流状态下等待直到限流结束
func (s *RateLimitEventTestSuite) TestConsumerWithRateLimit() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-ratelimit-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001),
	}
	evt := ratelimitevt.RequestRateLimitedEvent{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 模拟限流器行为 - 首次检查时处于限流期，第二次检查时限流结束
	limitTime := time.Now() // 刚刚发生的限流
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(limitTime, nil)
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil) // 第二次检查，限流已结束

	// 模拟服务行为 - 成功处理通知
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{2001},
		}, nil)

	assert.NoError(t, consumer.Consume(ctx))
}

// TestConsumerWithServiceError 测试消费者处理服务异常的情况
func (s *RateLimitEventTestSuite) TestConsumerWithServiceError() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-ratelimit-limitKey-%s", s.testID)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001),
	}
	evt := ratelimitevt.RequestRateLimitedEvent{
		Notifications: notifications,
	}

	// 模拟服务行为 - 处理通知失败
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{}, fmt.Errorf("模拟服务错误"))

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	assert.NoError(t, consumer.Consume(ctx))
}

// TestConsumerWithLimiterError 测试消费者处理限流器异常的情况
func (s *RateLimitEventTestSuite) TestConsumerWithLimiterError() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, _ := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-ratelimit-limitKey-%s", s.testID)

	// 模拟限流器行为 - 获取限流状态失败
	wantErr := fmt.Errorf("模拟限流器错误")
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, wantErr)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001),
	}
	evt := ratelimitevt.RequestRateLimitedEvent{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	err = consumer.Consume(ctx)
	assert.Error(t, wantErr, "当限流器出错时，消费者应该返回错误")
	assert.Equal(t, wantErr, err)
}
