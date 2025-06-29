package provider

import (
	"context"
	"notification-platform/internal/domain"
	"sync/atomic"
)

type MockProvider struct {
	count int64
}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) Send(_ context.Context, _ domain.Notification) (domain.SendResponse, error) {
	v := atomic.AddInt64(&m.count, 1)
	// 随机睡眠1-2秒
	// sleepTime := 1 + rand.Float64() // 生成1到2之间的随机数
	// time.Sleep(time.Duration(sleepTime * float64(time.Second)))

	return domain.SendResponse{
		Status:         domain.SendStatusSucceeded,
		NotificationID: uint64(v),
	}, nil
}
