package grpcx

import (
	"context"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"notification-platform/internal/pkg/registry"
	"time"
)

type CtxKey string

const (
	readWeightStr         = "read_weight"
	writeWeightStr        = "write_weight"
	groupStr              = "group"
	nodeStr               = "nodeStr"
	RequestType    CtxKey = "requestType"
)

type grpcResolverBuilder struct {
	r       registry.Registry
	timeout time.Duration
}

func NewResolverBuilder(r registry.Registry, timeout time.Duration) resolver.Builder {
	return &grpcResolverBuilder{
		r:       r,
		timeout: timeout,
	}
}

func (b *grpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	res := &grpcResolver{
		target:   target,
		cc:       cc,
		registry: b.r,
		close:    make(chan struct{}, 1),
		timeout:  b.timeout,
	}
	res.resolve()
	go res.watch()
	return res, nil
}

func (b *grpcResolverBuilder) Scheme() string {
	return "registry"
}

type grpcResolver struct {
	target   resolver.Target
	cc       resolver.ClientConn
	registry registry.Registry
	close    chan struct{}
	timeout  time.Duration
}

func (r *grpcResolver) ResolveNow(_ resolver.ResolveNowOptions) {
	// 重新获取一下所有服务
	r.resolve()
}

func (r *grpcResolver) Close() {
	r.close <- struct{}{}
}

func (r *grpcResolver) watch() {
	events := r.registry.Subscribe(r.target.Endpoint())
	for {
		select {
		case <-events:
			r.resolve()
		case <-r.close:
			return
		}
	}
}

func (r *grpcResolver) resolve() {
	serviceName := r.target.Endpoint()
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	instances, err := r.registry.ListServices(ctx, serviceName)
	cancel()
	if err != nil {
		r.cc.ReportError(err)
	}
	address := make([]resolver.Address, 0, len(instances))
	for _, ins := range instances {
		address = append(address, resolver.Address{
			Addr:       ins.Address,
			ServerName: ins.Name,
			Attributes: attributes.New(readWeightStr, ins.ReadWeight).
				WithValue(writeWeightStr, ins.WriteWeight).
				WithValue(groupStr, ins.Group).
				WithValue(nodeStr, ins.Name),
		})
	}
	err = r.cc.UpdateState(resolver.State{
		Addresses: address,
	})
	if err != nil {
		r.cc.ReportError(err)
	}
}
