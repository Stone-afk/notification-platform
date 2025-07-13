package loadbalancer

import (
	"context"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
	"sync"
	"sync/atomic"
)

type Selector struct {
	providers []*mprovider  // 被封装的provider列表
	count     int64         // 轮询计数器
	mu        *sync.RWMutex // 保护providers的并发访问
}

func (s *Selector) Next(_ context.Context, _ domain.Notification) (provider.Provider, error) {
	s.mu.RLock()
	providers := s.providers
	s.mu.RUnlock()
	providerLen := len(providers)
	if providerLen == 0 {
		return nil, ErrNoProvidersAvailable
	}
	current := atomic.AddInt64(&s.count, 1)
	for i := 0; i < providerLen; i++ {
		idx := (int(current) + i) % providerLen
		pro := providers[idx]
		if pro != nil && pro.isHealthy() {
			return pro, nil
		}
	}
	return nil, ErrNoHealthyProvider
}

func NewSelector(providers []provider.Provider, bufferLen int) *Selector {
	if bufferLen <= 0 {
		bufferLen = 10 // 默认缓冲区长度
	}

	// 预分配足够的容量避免扩容
	mproviders := make([]*mprovider, 0, len(providers))
	for _, p := range providers {
		mp := newMprovider(p, bufferLen)
		mproviders = append(mproviders, &mp)
	}

	return &Selector{
		providers: mproviders,
		count:     0,
		mu:        &sync.RWMutex{},
	}
}
