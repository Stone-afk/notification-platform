package etcd

import (
	"context"
	"fmt"
	"github.com/ego-component/eetcd"
	clientv3 "go.etcd.io/etcd/client/v3"
	"notification-platform/internal/errs"
	"notification-platform/internal/pkg/configcenter/failover"
	"strings"
	"sync"
)

const (
	// failoverPrefix 是存储故障转移信息的前缀
	failoverPrefix = "/config/failover"
)

var _ failover.FailoverManager = (*Manager)(nil)

// Manager 实现基于etcd的故障转移管理
type Manager struct {
	client *eetcd.Component

	mutex       sync.RWMutex
	watchCancel []context.CancelFunc
}

// Failover 标记服务需要故障转移
func (m *Manager) Failover(ctx context.Context, serviceInstance failover.ServiceInstance) error {
	if err := serviceInstance.Validate(); err != nil {
		return err
	}

	key := m.buildKey(serviceInstance)
	_, err := m.client.Put(ctx, key, "")
	return err
}

// Recover 标记服务已恢复可用
func (m *Manager) Recover(ctx context.Context, serviceInstance failover.ServiceInstance) error {
	if err := serviceInstance.Validate(); err != nil {
		return err
	}

	key := m.buildKey(serviceInstance)
	_, err := m.client.Delete(ctx, key)
	return err
}

// WatchFailover 监听故障转移事件
func (m *Manager) WatchFailover(ctx context.Context) (<-chan failover.FailoverEvent, error) {
	// 创建带取消的上下文
	watchCtx, cancel := context.WithCancel(ctx)

	m.mutex.Lock()
	m.watchCancel = append(m.watchCancel, cancel)
	m.mutex.Unlock()

	// 监听前缀下的事件
	watchCh := m.client.Watch(watchCtx, failoverPrefix, clientv3.WithPrefix())
	eventCh := make(chan failover.FailoverEvent)

	go func() {
		defer close(eventCh)

		for {
			select {
			case resp := <-watchCh:
				if resp.Canceled {
					return
				}

				if err := resp.Err(); err != nil {
					continue
				}

				for _, event := range resp.Events {
					// 只关心新增事件且值为空的情况
					if event.Type == clientv3.EventTypePut && string(event.Kv.Value) == "" {
						key := string(event.Kv.Key)
						svc, err := m.parseKey(key)
						if err != nil {
							continue
						}

						select {
						case eventCh <- failover.FailoverEvent{
							ServiceInstance: failover.ServiceInstance{
								Group:   svc.Group,
								Name:    svc.Name,
								Address: svc.Address,
							},
						}:
						case <-watchCtx.Done():
							return
						}
					}
				}
			case <-watchCtx.Done():
				return
			}
		}
	}()

	return eventCh, nil
}

// parseKey 从键中解析出组和服务名
func (m *Manager) parseKey(key string) (*failover.ServiceInstance, error) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(key, failoverPrefix), "/"), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: key=%s", errs.ErrInvalidParameter, key)
	}
	return &failover.ServiceInstance{
		Name:    parts[0],
		Group:   parts[1],
		Address: parts[2],
	}, nil
}

// WatchRecovery 监听指定服务的恢复事件
func (m *Manager) WatchRecovery(ctx context.Context, serviceInstance failover.ServiceInstance) (<-chan struct{}, error) {
	if err := serviceInstance.Validate(); err != nil {
		return nil, err
	}

	key := m.buildKey(serviceInstance)
	// 创建带取消的上下文
	watchCtx, cancel := context.WithCancel(ctx)
	m.mutex.Lock()
	m.watchCancel = append(m.watchCancel, cancel)
	m.mutex.Unlock()
	// 监听指定键的删除事件
	watchCh := m.client.Watch(watchCtx, key)
	notifyCh := make(chan struct{})
	go func() {
		defer close(notifyCh)

		for {
			select {
			case resp := <-watchCh:

				if resp.Canceled {
					return
				}

				if err := resp.Err(); err != nil {
					continue
				}

				for _, event := range resp.Events {
					// 只关心删除事件
					if event.Type == clientv3.EventTypeDelete {
						select {
						case notifyCh <- struct{}{}:
							return
						case <-watchCtx.Done():
							return
						}
					}
				}
			case <-watchCtx.Done():
				return
			}
		}
	}()

	return notifyCh, nil
}

// TryTakeOver 尝试接管特定的资源不足服务
func (m *Manager) TryTakeOver(ctx context.Context, standbyService, targetService failover.ServiceInstance) (bool, error) {
	if err := standbyService.Validate(); err != nil {
		return false, err
	}

	if err := targetService.Validate(); err != nil {
		return false, err
	}

	key := m.buildKey(targetService)
	// 先检查目标服务是否需要接管
	resp, err := m.client.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf(": %w", err)
	}
	// 如果服务不存在或已被接管
	if len(resp.Kvs) == 0 {
		return false, fmt.Errorf("%w", errs.ErrNoAvailableFailoverService)
	}
	if string(resp.Kvs[0].Value) != "" {
		return false, fmt.Errorf("%w: 服务已被接管", errs.ErrNoAvailableFailoverService)
	}

	// 尝试原子性地更新，将空值更新为自己的名称
	txnResp, err := m.client.Txn(ctx).
		If(clientv3.Compare(clientv3.Value(key), "=", "")).
		Then(clientv3.OpPut(key, m.buildValue(standbyService))).
		Commit()
	if err != nil {
		return false, fmt.Errorf("%w: 接管故障服务过程出错", err)
	}

	// 返回是否成功接管
	return txnResp.Succeeded, nil
}

func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, cancel := range m.watchCancel {
		cancel()
	}
	m.watchCancel = nil
	return nil
}

// buildKey 构建故障键
func (m *Manager) buildKey(svc failover.ServiceInstance) string {
	return fmt.Sprintf("%s/%s/%s/%s", failoverPrefix, svc.Name, svc.Group, svc.Address)
}

func (m *Manager) buildValue(svc failover.ServiceInstance) string {
	return fmt.Sprintf("%s:%s:%s", svc.Group, svc.Name, svc.Address)
}

// NewFailoverManager 创建基于etcd的故障转移管理器
func NewFailoverManager(c *eetcd.Component) *Manager {
	return &Manager{
		client:      c,
		watchCancel: make([]context.CancelFunc, 0),
	}
}
