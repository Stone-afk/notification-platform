//go:build e2e

package integration

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"

	"gorm.io/gorm"
	"notification-platform/internal/event/failover"
	monMocks "notification-platform/internal/pkg/gormx/monitor/mocks"
	"notification-platform/internal/test/ioc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type User0 struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
}

type ConsumerTestSuite struct {
	suite.Suite
	mockCtrl    *gomock.Controller
	mockMonitor *monMocks.MockDBMonitor
	gormDB      *gorm.DB
	producer    failover.ConnPoolEventProducer
	consumer    *failover.ConnPoolEventConsumer
}

func (s *ConsumerTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockMonitor = monMocks.NewMockDBMonitor(s.mockCtrl)

	// 初始化生产者
	ioc.InitTopic()
	pro := ioc.InitProducer("failover")
	s.producer = failover.NewProducer(pro)

	// 初始化
	dsn := "root:root@tcp(localhost:13316)/notification?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s&multiStatements=true&interpolateParams=true"
	ioc.WaitForDBSetup(dsn)
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	s.gormDB = gormDB
	s.gormDB.AutoMigrate(&User0{})
	config := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "failOverGroup",
		"auto.offset.reset": "earliest",
	}
	con, err := kafka.NewConsumer(config)
	require.NoError(s.T(), err)
	err = con.SubscribeTopics([]string{failover.FailoverTopic}, nil)
	require.NoError(s.T(), err)
	// 触发分区
	s.initializeConsumer(con)
	// 创建消费者实例
	s.consumer = failover.NewConnPoolEventConsumer(con, s.gormDB, s.mockMonitor)
}

func (c *ConsumerTestSuite) initializeConsumer(con *kafka.Consumer) {
	// 设置超时时间，避免永久阻塞
	timeout := time.Now().Add(30 * time.Second)
	for time.Now().Before(timeout) {
		// 进行一次Poll操作，触发分区分配
		ev := con.Poll(1000)
		if ev == nil {
			continue
		}
		return
	}
}

func (s *ConsumerTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
	s.gormDB.Exec("truncate table `user0`;")
}

func (s *ConsumerTestSuite) TestConsumerBehaviorWithMonitor() {
	t := s.T()
	t.Skip()
	start := time.Now().Unix()
	// 第一阶段：监控状态为false时的消费行为
	s.mockMonitor.EXPECT().Health().DoAndReturn(func() bool {
		end := time.Now().Unix()
		if end-start > 3 {
			return true
		}
		return false
	}).AnyTimes()

	// 推送三条测试消息
	events := []failover.ConnPoolEvent{
		{SQL: "INSERT INTO user0 (`id`,`name`) VALUES (?,?)", Args: []any{1, "user1"}},
		{SQL: "INSERT INTO user0 (`id`,`name`) VALUES (?,?)", Args: []any{2, "user2"}},
		{SQL: "INSERT INTO user0 (`id`,`name`) VALUES (?,?)", Args: []any{3, "user3"}},
	}
	for _, event := range events {
		err := s.producer.Produce(t.Context(), event)
		require.NoError(t, err)
	}

	// 消费者启动
	go s.consumer.Start(t.Context())
	time.Sleep(1 * time.Second)

	// 验证2秒内无数据写入
	assert.Eventually(t, func() bool {
		var count int64
		s.gormDB.Model(&User0{}).Count(&count)
		return count == 0
	}, 2*time.Second, 100*time.Millisecond, "数据库在监控不可用时不应有数据")

	// 第二阶段：切换监控状态为true
	time.Sleep(3 * time.Second)

	// 验证数据正确写入
	var userList []User0
	err := s.gormDB.Model(&User0{}).
		Order("id asc").
		Find(&userList).Error
	require.NoError(t, err)

	assert.Equal(t, []User0{
		{
			ID:   1,
			Name: "user1",
		},
		{
			ID:   2,
			Name: "user2",
		},
		{
			ID:   3,
			Name: "user3",
		},
	}, userList)
}

func TestConsumerSuite(t *testing.T) {
	suite.Run(t, new(ConsumerTestSuite))
}
