package loadbalancer

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockHealthAwareProvider 是一个模拟的HealthAwareProvider实现
type MockHealthAwareProvider struct {
	name           string
	failCount      atomic.Int32 // 用于控制连续失败次数
	mu             sync.RWMutex // 保护shouldFail和healthCheckErr
	shouldFail     bool         // 是否应该失败
	callCount      atomic.Int32 // 调用计数
	healthyStatus  atomic.Bool
	healthCheckErr error
}

func NewMockHealthAwareProvider(name string, shouldFail bool) *MockHealthAwareProvider {
	m := &MockHealthAwareProvider{
		name:           name,
		shouldFail:     shouldFail,
		healthCheckErr: nil,
	}
	m.healthyStatus.Store(true)
	return m
}

func (m *MockHealthAwareProvider) Send(_ context.Context, notification domain.Notification) (domain.SendResponse, error) {
	m.callCount.Add(1)

	m.mu.RLock()
	shouldFail := m.shouldFail
	m.mu.RUnlock()

	if shouldFail {
		m.failCount.Add(1)
		return domain.SendResponse{}, errors.New("mock provider sending error")
	}

	return domain.SendResponse{
		NotificationID: notification.ID,
		Status:         domain.SendStatusSucceeded,
	}, nil
}

// 将失败的提供者设置为恢复
func (m *MockHealthAwareProvider) MarkRecovered() {
	m.mu.Lock()
	m.shouldFail = false
	m.healthCheckErr = nil
	m.mu.Unlock()
}

// 获取调用计数，确保线程安全
func (m *MockHealthAwareProvider) GetCallCount() int32 {
	return m.callCount.Load()
}

// 重置调用计数，确保线程安全
func (m *MockHealthAwareProvider) ResetCallCount() {
	m.callCount.Store(0)
}

// TestProviderLoadBalancingAndHealthRecovery 测试负载均衡Provider的主流程
// 包括：
// 1. 正常的轮询发送
// 2. 当一个provider持续失败时，会被标记为不健康
// 3. 当不健康的provider恢复后，会被重新标记为健康
func TestProviderLoadBalancingAndHealthRecovery(t *testing.T) {
	t.Skip()
	t.Parallel()
	// 创建3个模拟的Provider，其中一个会持续失败
	provider1 := NewMockHealthAwareProvider("provider1", false)
	provider2 := NewMockHealthAwareProvider("provider2", true) // 这个会一直失败
	provider3 := NewMockHealthAwareProvider("provider3", false)

	providers := []provider.Provider{provider1, provider2, provider3}

	// 创建负载均衡Provider，设置缓冲区长度为10，这样失败率超过阈值后provider2会被标记为不健康
	lb := NewProvider(providers, 10)

	// 创建一个测试通知
	notification := domain.Notification{
		ID:        123,
		BizID:     456,
		Key:       "test-key",
		Channel:   domain.ChannelSMS,
		Receivers: []string{"13800138000"},
		Template: domain.Template{
			ID:        789,
			VersionID: 1,
			Params:    map[string]string{"code": "1234"},
		},
	}

	// 第1阶段：发送足够多的请求，使provider2达到失败阈值并被标记为不健康
	for i := 0; i < 2000; i++ {
		_, _ = lb.Send(t.Context(), notification)
	}

	// 等待provider2被标记为不健康
	time.Sleep(1 * time.Second)

	// 重置所有provider的计数
	provider1.ResetCallCount()
	provider2.ResetCallCount()
	provider3.ResetCallCount()

	// 第2阶段：连续发送5个请求，验证它们不会发送到不健康的provider2
	for i := 0; i < 5; i++ {
		resp, err := lb.Send(t.Context(), notification)
		assert.NoError(t, err)
		assert.Equal(t, notification.ID, resp.NotificationID)
	}

	// 验证provider2没有收到任何请求，而其他两个provider都收到了请求
	assert.Greater(t, provider1.GetCallCount(), int32(0))
	assert.Equal(t, int32(0), provider2.GetCallCount())
	assert.Greater(t, provider3.GetCallCount(), int32(0))

	// 将provider2的行为改为返回成功
	provider2.mu.Lock()
	provider2.shouldFail = false
	provider2.mu.Unlock()

	// 第3阶段：等待3秒钟，让健康检查机制自动恢复provider2
	time.Sleep(61 * time.Second)
	// 重置所有provider的计数
	provider1.ResetCallCount()
	provider2.ResetCallCount()
	provider3.ResetCallCount()

	// 第4阶段：再次发送请求，验证所有provider（包括恢复的provider2）都能收到请求
	for i := 0; i < 9; i++ {
		resp, err := lb.Send(t.Context(), notification)
		assert.NoError(t, err)
		assert.Equal(t, notification.ID, resp.NotificationID)
	}

	// 验证所有provider都收到了请求
	assert.Greater(t, provider1.GetCallCount(), int32(0))
	assert.Greater(t, provider2.GetCallCount(), int32(0))
	assert.Greater(t, provider3.GetCallCount(), int32(0))
}
