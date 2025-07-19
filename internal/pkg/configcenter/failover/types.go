package failover

import (
	"context"
	"fmt"
	"notification-platform/internal/errs"
)

type ServiceInstance struct {
	Name    string
	Address string
	Group   string
}

func (svc *ServiceInstance) Validate() error {
	if svc.Name == "" {
		return fmt.Errorf("%w: 服务实例名称：%s", errs.ErrInvalidParameter, svc.Name)
	}

	if svc.Address == "" {
		return fmt.Errorf("%w: 服务实例地址：%s", errs.ErrInvalidParameter, svc.Address)
	}
	return nil
}

// FailoverEvent 描述故障转移事件
type FailoverEvent struct {
	ServiceInstance ServiceInstance
}

// FailoverManager 定义服务故障转移管理接口
type FailoverManager interface {
	// Failover 标记服务需要故障转移
	Failover(ctx context.Context, serviceInstance ServiceInstance) error

	// Recover 标记服务已恢复可用
	Recover(ctx context.Context, serviceInstance ServiceInstance) error

	// WatchFailover 监听故障转移事件
	WatchFailover(ctx context.Context) (<-chan FailoverEvent, error)

	// WatchRecovery 监听指定服务的恢复事件
	WatchRecovery(ctx context.Context, serviceInstance ServiceInstance) (<-chan struct{}, error)

	// TryTakeOver standbyServiceName 服务尝试接管特定的资源不足服务
	TryTakeOver(ctx context.Context, standbyService, targetService ServiceInstance) (bool, error)

	// Close 关闭管理器资源
	Close() error
}
