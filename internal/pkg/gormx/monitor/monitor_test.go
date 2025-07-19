package monitor

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"
)

func TestHeartbeat_Health(t *testing.T) {
	t.Skip()
	// 测试逻辑
	// 1. 使用sqlmock进行测试，刚开始调用Health()方法为true
	// 2. 持续五秒 sqlmock的pingctx返回报错, 三秒过后调用Health()方法为false
	// 3. 五秒过后pingctx会返回一直正确的，四秒过后Health()方法为true
	// 创建sql mock，启用ping监控
	t.Parallel()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	// 创建Heartbeat监控
	monitor := NewHeartbeatDBMonitor(db)

	go func() {
		monitor.healthCheck(t.Context())
	}()

	// 阶段1: 初始化状态健康应为true
	if !monitor.Health() {
		t.Errorf("Initial health status should be true")
	}

	// 阶段2: 让PingContext持续5秒返回错误
	dbError := errors.New("db connection error")

	// 设置多次Ping调用返回错误（至少3次以触发健康状态变更）
	mock.ExpectPing().WillReturnError(dbError)
	mock.ExpectPing().WillReturnError(dbError)
	mock.ExpectPing().WillReturnError(dbError)
	mock.ExpectPing().WillReturnError(dbError)

	// 等待3秒以确保健康检查有足够时间执行
	time.Sleep(4 * time.Second)

	// 验证健康状态为false
	if monitor.Health() {
		t.Errorf("Health status should be false after 3 consecutive failures")
	}

	// 阶段3: 让PingContext持续返回成功
	// 设置多次Ping调用成功（至少3次以触发健康状态恢复）
	mock.ExpectPing()
	mock.ExpectPing()
	mock.ExpectPing()

	// 等待4秒，健康状态应该恢复为true
	time.Sleep(4 * time.Second)

	if !monitor.Health() {
		t.Errorf("Health status should be true after 3 consecutive successes")
	}

	// 验证所有预期的调用都已发生
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}
