//go:build e2e

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	notificationevt "notification-platform/internal/event/notification"
	batchmocks "notification-platform/internal/pkg/batchsize/mocks"
	"notification-platform/internal/pkg/mqx"
	limitmocks "notification-platform/internal/pkg/ratelimit/mocks"
	notificationmocks "notification-platform/internal/service/notification/mocks"
)

func TestNotificationEventSuite(t *testing.T) {
	t.Parallel()
	t.Skip()
	suite.Run(t, new(NotificationEventTestSuite))
}

type NotificationEventTestSuite struct {
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
func (s *NotificationEventTestSuite) SetupTest() {
	// 生成唯一的测试标识
	v4, err := uuid.DefaultGenerator.NewV4()
	s.NoError(err)

	s.testID = fmt.Sprintf("test-%s", v4.String())
	// 生成唯一的测试topic名称
	s.testTopic = fmt.Sprintf("notification_events-%s", s.testID)
	s.kafkaAddr = defaultTestKafkaAddr

	// 创建AdminClient
	s.adminClient, err = kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": s.kafkaAddr,
	})
	s.NoError(err)

	// 创建测试topic
	s.createTestTopic(s.testTopic)
}

// TearDownTest 在每个测试方法执行后运行
func (s *NotificationEventTestSuite) TearDownTest() {
	// 删除测试topic
	if s.adminClient != nil {
		s.deleteTestTopic(s.testTopic)
		s.adminClient.Close()
	}
}

// createTestTopic 创建测试topic
func (s *NotificationEventTestSuite) createTestTopic(topic string) {
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
func (s *NotificationEventTestSuite) deleteTestTopic(topic string) {
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
func (s *NotificationEventTestSuite) newTestComponent(ctrl *gomock.Controller) (eventProducer *mqx.GeneralProducer[notificationevt.Event], eventConsumer *notificationevt.EventConsumer, mockLimiter *limitmocks.MockLimiter, mockService *notificationmocks.MockSendService, mockBatchSizeAdjuster *batchmocks.MockAdjuster) {
	mockService = notificationmocks.NewMockSendService(ctrl)
	mockLimiter = limitmocks.NewMockLimiter(ctrl)
	mockBatchSizeAdjuster = batchmocks.NewMockAdjuster(ctrl)

	// 为当前测试创建唯一的consumer group
	groupID := fmt.Sprintf("mock-notification-group-%s", s.testID)

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

	eventProducer, err = mqx.NewGeneralProducer[notificationevt.Event](producer, s.testTopic)
	s.NoError(err)

	// 使用较短的批次超时时间，以便测试能快速运行
	eventConsumer, err = notificationevt.NewEventConsumerWithTopic(
		mockService,
		consumer,
		fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID),
		mockLimiter,
		20*time.Second,
		200*time.Millisecond,
		10,
		10*time.Second,
		mockBatchSizeAdjuster,
		s.testTopic,
	)
	s.NoError(err)

	return eventProducer, eventConsumer, mockLimiter, mockService, mockBatchSizeAdjuster
}

// createTestNotification 创建测试用的通知对象
func (s *NotificationEventTestSuite) createTestNotification(id uint64, bizID int64) domain.Notification {
	now := time.Now()
	return domain.Notification{
		ID:        id,
		BizID:     bizID,
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

// 向Kafka发送一条无效消息
func (s *NotificationEventTestSuite) sendInvalidMessage(topic string) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": s.kafkaAddr,
	})
	if err != nil {
		return err
	}
	defer producer.Close()

	// 发送一条格式错误的消息
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: []byte("invalid json"),
	}, nil)
	if err != nil {
		return err
	}

	producer.Flush(15 * 1000)
	return nil
}

// 创建一个自定义匹配器来匹配特定bizID的通知
type notificationsBizIDMatcher struct {
	bizID int64
	t     *testing.T
}

func (m *notificationsBizIDMatcher) Matches(x interface{}) bool {
	notifications, ok := x.([]domain.Notification)
	if !ok {
		return false
	}

	for _, n := range notifications {
		if n.BizID != m.bizID {
			m.t.Errorf("通知业务ID不匹配: 期望 %d, 实际 %d", m.bizID, n.BizID)
			return false
		}
	}
	return true
}

func (m *notificationsBizIDMatcher) String() string {
	return fmt.Sprintf("有业务ID为 %d 的通知", m.bizID)
}

// 创建一个匹配特定bizID的通知的匹配器
func matchNotificationsWithBizID(t *testing.T, bizID int64) gomock.Matcher {
	return &notificationsBizIDMatcher{bizID: bizID, t: t}
}

// TestConsumerWithoutRateLimit 测试消费者在非限流状态下成功消费事件
func (s *NotificationEventTestSuite) TestConsumerWithoutRateLimit() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService, mockBatchSizeAdjuster := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001, 123),
		s.createTestNotification(1002, 123),
	}
	evt := notificationevt.Event{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 模拟服务行为 - 成功处理通知
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{2001, 2002},
		}, nil)

	// 模拟批量大小调整器
	mockBatchSizeAdjuster.EXPECT().
		Adjust(gomock.Any(), gomock.Any()).
		Return(15, nil)

	err = consumer.Consume(ctx)
	assert.NoError(t, err)
}

// TestConsumerWithRateLimit 测试消费者在限流状态下等待直到限流结束
func (s *NotificationEventTestSuite) TestConsumerWithRateLimit() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService, mockBatchSizeAdjuster := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001, 123),
	}
	evt := notificationevt.Event{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 首次检查时处于限流期，第二次检查时限流结束
	limitTime := time.Now() // 刚刚发生的限流
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(limitTime, nil)

	// 由于我们无法直接访问或修改kafka.Consumer的内部属性，这里只能测试限流检测部分
	// 并跳过实际的Pause/Resume操作，因此我们需要第二次调用来模拟限流结束
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil) // 第二次检查，限流已结束

	// 模拟服务行为 - 成功处理通知
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{2001},
		}, nil)

	// 模拟批量大小调整器
	mockBatchSizeAdjuster.EXPECT().
		Adjust(gomock.Any(), gomock.Any()).
		Return(15, nil)

	err = consumer.Consume(ctx)
	assert.NoError(t, err)
}

// TestConsumerWithServiceError 测试消费者处理服务异常的情况
func (s *NotificationEventTestSuite) TestConsumerWithServiceError() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService, _ := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001, 123),
	}
	evt := notificationevt.Event{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 模拟服务行为 - 处理通知失败
	serviceErr := fmt.Errorf("模拟服务错误")
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{}, serviceErr)

	// 消费事件 - 应该返回服务错误
	err = consumer.Consume(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "模拟服务错误")
}

// TestConsumerWithLimiterError 测试消费者处理限流器异常的情况
func (s *NotificationEventTestSuite) TestConsumerWithLimiterError() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, _, _ := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001, 123),
	}
	evt := notificationevt.Event{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 获取限流状态失败
	limiterErr := fmt.Errorf("模拟限流器错误")
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, limiterErr)

	// 消费事件 - 应该返回限流器错误
	err = consumer.Consume(ctx)
	assert.Error(t, err)
	assert.Equal(t, limiterErr, err)
}

// TestConsumerWithInvalidMessage 测试消费者处理无效消息的情况
func (s *NotificationEventTestSuite) TestConsumerWithInvalidMessage() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, consumer, _, _, _ := s.newTestComponent(ctrl)

	// 向topic发送一条无效的消息
	err := s.sendInvalidMessage(s.testTopic)
	assert.NoError(t, err, "应该能够发送无效消息")

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 消费者应该能够跳过无效消息
	// 注意：由于批处理超时，这将会超时并返回nil，而不会返回错误
	err = consumer.Consume(ctx)
	assert.NoError(t, err, "消费者应该能够正常处理无效消息")
}

// TestConsumerWithMultipleBizIDs 测试消费者处理多个业务ID的通知
func (s *NotificationEventTestSuite) TestConsumerWithMultipleBizIDs() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService, mockBatchSizeAdjuster := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 第一个业务ID
	bizID1 := int64(123)
	evt1 := notificationevt.Event{
		Notifications: []domain.Notification{
			s.createTestNotification(1001, bizID1),
			s.createTestNotification(1002, bizID1),
		},
	}
	err := producer.Produce(ctx, evt1)
	assert.NoError(t, err)

	// 第二个业务ID
	bizID2 := int64(456)
	evt2 := notificationevt.Event{
		Notifications: []domain.Notification{
			s.createTestNotification(2001, bizID2),
			s.createTestNotification(2002, bizID2),
		},
	}
	err = producer.Produce(ctx, evt2)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 模拟服务行为 - 成功处理不同业务ID的通知
	// 第一个业务ID的通知处理
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), matchNotificationsWithBizID(t, bizID1)).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{3001, 3002},
		}, nil)

	// 第二个业务ID的通知处理
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), matchNotificationsWithBizID(t, bizID2)).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{4001, 4002},
		}, nil)

	// 模拟批量大小调整器
	mockBatchSizeAdjuster.EXPECT().
		Adjust(gomock.Any(), gomock.Any()).
		Return(15, nil)

	// 消费事件
	err = consumer.Consume(ctx)
	assert.NoError(t, err)
}

// TestConsumerWithDuplicateMessage 测试消费者处理重复消息的情况
func (s *NotificationEventTestSuite) TestConsumerWithDuplicateMessage() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService, mockBatchSizeAdjuster := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001, 123),
	}
	evt := notificationevt.Event{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 模拟服务行为 - 处理通知时返回重复错误
	duplicateErr := errs.ErrNotificationDuplicate
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{}, duplicateErr)

	// 模拟批量大小调整器
	mockBatchSizeAdjuster.EXPECT().
		Adjust(gomock.Any(), gomock.Any()).
		Return(15, nil)

	// 消费事件 - 应该能够处理重复错误并继续
	err = consumer.Consume(ctx)
	assert.NoError(t, err, "当遇到重复通知错误时，消费者应该继续处理")
}

// TestConsumerBatchSizeAdjusterError 测试批量大小调整器出错的情况
func (s *NotificationEventTestSuite) TestConsumerBatchSizeAdjusterError() {
	t := s.T()
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	producer, consumer, mockLimiter, mockService, mockBatchSizeAdjuster := s.newTestComponent(ctrl)

	// 使用测试ID生成唯一的限流键
	limitKey := fmt.Sprintf("mock-notifiction-limitKey-%s", s.testID)

	// 创建并发送测试事件
	notifications := []domain.Notification{
		s.createTestNotification(1001, 123),
	}
	evt := notificationevt.Event{
		Notifications: notifications,
	}

	// 发送事件
	err := producer.Produce(ctx, evt)
	assert.NoError(t, err)

	// 确保消息已发送
	time.Sleep(500 * time.Millisecond)

	// 模拟限流器行为 - 没有限流发生
	mockLimiter.EXPECT().
		LastLimitTime(gomock.Any(), limitKey).
		Return(time.Time{}, nil)

	// 模拟服务行为 - 成功处理通知
	mockService.EXPECT().
		BatchSendNotificationsAsync(gomock.Any(), gomock.Any()).
		Return(domain.BatchSendAsyncResponse{
			NotificationIDs: []uint64{2001},
		}, nil)

	// 模拟批量大小调整器出错
	adjusterErr := fmt.Errorf("批量大小调整器错误")
	mockBatchSizeAdjuster.EXPECT().
		Adjust(gomock.Any(), gomock.Any()).
		Return(0, adjusterErr)

	// 消费事件 - 应该能够处理调整器错误并继续，但不更新批量大小
	err = consumer.Consume(ctx)
	assert.NoError(t, err, "当批量大小调整器出错时，消费者应该继续处理")
}
