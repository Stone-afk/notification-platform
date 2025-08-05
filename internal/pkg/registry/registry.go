package registry

import (
	"context"
	"errors"
	"github.com/hashicorp/go-multierror"
	"sync/atomic"
)

// RegistryStatus 增加心跳方法，然后心跳不通，就把 active 设置为 false
type RegistryStatus struct {
	r      Registry
	active atomic.Bool
}

type MultipleRegistry struct {
	registries []Registry
	main       RegistryStatus
	back       RegistryStatus
}

func (m *MultipleRegistry) ListServicesV1(ctx context.Context, name string) ([]ServiceInstance, error) {
	if m.main.active.Load() {
		return m.main.r.ListServices(ctx, name)
	}
	return m.back.r.ListServices(ctx, name)
}

func (m *MultipleRegistry) ListServices(ctx context.Context, svcName string) ([]ServiceInstance, error) {
	for _, r := range m.registries {
		svcs, err := r.ListServices(ctx, svcName)
		if err == nil {
			return svcs, nil
		}
	}
	return nil, errors.New("所有注册中心都失败了")
}

func (m *MultipleRegistry) Register(ctx context.Context, si ServiceInstance) error {
	for _, r := range m.registries {
		err := r.Register(ctx, si)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MultipleRegistry) UnRegister(ctx context.Context, si ServiceInstance) error {
	var err error
	for _, r := range m.registries {
		err = multierror.Append(err, r.UnRegister(ctx, si))
		// 你可以考虑记录日志，Deregister != nil 的时候
	}
	return err
}

func (m *MultipleRegistry) Subscribe(name string) <-chan Event {
	return m.registries[0].Subscribe(name)
}

func (m *MultipleRegistry) Close() error {
	var err error
	for _, r := range m.registries {
		err = r.Close()
	}
	return err
}
