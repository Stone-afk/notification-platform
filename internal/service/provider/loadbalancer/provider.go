package loadbalancer

import (
	"errors"
	"notification-platform/internal/service/provider"
	"sync"
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

// NewProvider 创建一个新的负载均衡Provider
// 参数:
//   - providers: 基础provider列表
//   - bufferLen: 健康状态监控的环形缓冲区长度，用于异常检测
func NewProvider(providers []provider.Provider, bufferLen int) *Provider {
	panic("not implemented") // TODO: Implement this function
}
