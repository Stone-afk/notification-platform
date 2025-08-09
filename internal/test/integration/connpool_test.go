////go:build e2e

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gorm.io/gorm"
	"notification-platform/internal/errs"
	"notification-platform/internal/event/failover"
	"notification-platform/internal/pkg/gormx/connpool"
	"notification-platform/internal/test/ioc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	porducerMock "notification-platform/internal/event/mocks"
	monitormocks "notification-platform/internal/pkg/gormx/monitor/mocks"
)

// User 用于测试的用户表结构
type User struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
}

// TableName 设置表名
func (User) TableName() string {
	return "users"
}

// ConnPoolSuite 是测试套件结构体
type ConnPoolSuite struct {
	suite.Suite
	mockCtrl     *gomock.Controller
	mockProducer *porducerMock.MockConnPoolEventProducer
	mockMonitor  *monitormocks.MockDBMonitor
	gormDB       *gorm.DB
	db           *sql.DB
}

// SetupSuite 在所有测试之前初始化测试环境
func (s *ConnPoolSuite) SetupSuite() {
	dsn := "root:root@tcp(localhost:13316)/notification?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s&multiStatements=true&interpolateParams=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		s.T().Fatal(err)
	}
	s.db = db
	ioc.WaitForDBSetup(dsn)

	// 确保用户表存在
	s.mockCtrl = gomock.NewController(s.T())
	s.mockMonitor = monitormocks.NewMockDBMonitor(s.mockCtrl)
	s.mockProducer = porducerMock.NewMockConnPoolEventProducer(s.mockCtrl)
	start := time.Now().Unix()
	// 设置默认的Health方法期望，确保在初始化期间被调用时不会出错
	s.mockMonitor.EXPECT().Health().DoAndReturn(func() bool {
		end := time.Now().Unix()
		if end-start > 1 {
			return false
		}
		return true
	}).AnyTimes()

	cp := connpool.NewDBWithFailOver(db, s.mockMonitor, s.mockProducer)
	s.gormDB = ioc.InitDBWithCustomConnPool(cp)
	err = s.gormDB.AutoMigrate(&User{})
	if err != nil {
		s.T().Fatal(err)
	}
}

// TearDownSuite 在所有测试之后清理资源
func (s *ConnPoolSuite) TearDownSuite() {
	// 清空表
	_, err := s.db.Exec("TRUNCATE TABLE users")
	if err != nil {
		s.T().Logf("清空表失败: %v", err)
	}

	// 关闭数据库连接
	if s.db != nil {
		err := s.db.Close()
		if err != nil {
			s.T().Logf("关闭数据库连接失败: %v", err)
		}
	}

	// 完成mock控制器
	if s.mockCtrl != nil {
		s.mockCtrl.Finish()
	}
}

// TestUnhealthyDatabaseFailover 测试数据库不健康时的故障转移流程
func (s *ConnPoolSuite) TestUnhealthyDatabaseFailover() {
	time.Sleep(2 * time.Second)

	// 设置Producer期望接收到正确的事件
	s.mockProducer.EXPECT().
		Produce(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, evt failover.ConnPoolEvent) error {
			// 验证SQL语句是否包含users表
			assert.Contains(s.T(), evt.SQL, "users")

			// 对于GORM生成的SQL，我们只能确认它包含表名，不能确切知道字段名在哪
			// 但可以验证参数
			userFound := false
			for _, arg := range evt.Args {
				if str, ok := arg.(string); ok && str == "user2" {
					userFound = true
					break
				}
			}
			assert.True(s.T(), userFound, "参数应该包含user2")

			return nil
		}).Times(1)

	// 尝试创建用户
	user := &User{
		Name: "user2",
	}

	// 预期创建会失败，但会发送到消息队列
	result := s.gormDB.WithContext(context.Background()).Create(user)
	assert.Equal(s.T(), errs.ErrToAsync, result.Error, "数据库故障时应该返回ErrToAsync错误")

	// 确认数据库中没有用户记录
	var count int64
	s.gormDB.Model(&User{}).Count(&count)
	assert.Equal(s.T(), int64(0), count, "数据库中应该没有用户记录")
}

// 运行测试套件
func TestConnPoolSuite(t *testing.T) {
	t.Skip()
	suite.Run(t, new(ConnPoolSuite))
}
