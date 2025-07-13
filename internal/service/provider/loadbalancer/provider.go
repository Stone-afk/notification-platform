package loadbalancer

import (
	"context"
	"errors"
	"notification-platform/internal/domain"
	"notification-platform/internal/service/provider"
	"sync"
	"sync/atomic"
)

// 定义包级别错误，提高错误处理的一致性
var (
	ErrNoProvidersAvailable = errors.New("no providers available")
	ErrNoHealthyProvider    = errors.New("no healthy provider available")
)

// Provider 实现了基于轮询的负载均衡通知发送
// 它会自动跳过不健康的provider，确保通知可靠发送
type Provider struct {
	providers []*mprovider  // 被封装的provider列表
	count     int64         // 轮询计数器
	mu        *sync.RWMutex // 保护providers的并发访问
}

// Send 轮询查找健康的provider来发送通知
// 如果所有provider都不健康，则返回错误
// 前提：p.providers的长度在使用过程中不会改变
func (p *Provider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	providers := p.providers
	providerCount := len(providers)
	if providerCount == 0 {
		return domain.SendResponse{}, ErrNoProvidersAvailable
	}

	// 原子操作获取并递增计数，确保均匀分配负载
	current := atomic.AddInt64(&p.count, 1) - 1

	// 轮询所有provider
	for i := 0; i < providerCount; i++ {
		// 计算当前要使用的provider索引
		idx := (int(current) + i) % providerCount

		// 由于providers长度不变，可以安全地直接访问
		pro := providers[idx]
		if pro != nil && pro.healthy.Load() {
			// 使用健康的provider发送通知
			resp, err := pro.Send(ctx, notification)
			if err == nil {
				return resp, nil
			}
		}
	}
	// 所有provider都不健康或发送失败
	return domain.SendResponse{}, ErrNoHealthyProvider
}

// NewProvider 创建一个新的负载均衡Provider
// 参数:
//   - providers: 基础provider列表
//   - bufferLen: 健康状态监控的环形缓冲区长度，用于异常检测
func NewProvider(providers []provider.Provider, bufferLen int) *Provider {
	if bufferLen <= 0 {
		bufferLen = 10 // 默认缓冲区长度
	}

	// 预分配足够的容量避免扩容
	mproviders := make([]*mprovider, 0, len(providers))
	for _, p := range providers {
		mp := newMprovider(p, bufferLen)
		mproviders = append(mproviders, &mp)
	}

	return &Provider{
		providers: mproviders,
		count:     0,
		mu:        &sync.RWMutex{},
	}
}
