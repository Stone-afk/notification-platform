package testgrpc

import (
	"context"
	"log"
	"net"

	"github.com/ego-component/eetcd/registry"
	"github.com/gotomicro/ego/core/constant"
	"github.com/gotomicro/ego/server"
	"google.golang.org/grpc"
)

type Server[T any] struct {
	name string
	*grpc.Server
	grpcServer   T
	reg          *registry.Component
	registerFunc func(s grpc.ServiceRegistrar, srv T)
}

func NewServer[T any](name string, reg *registry.Component, grpcServer T, registerFunc func(s grpc.ServiceRegistrar, srv T)) *Server[T] {
	return &Server[T]{
		name:         name,
		reg:          reg,
		Server:       grpc.NewServer(),
		grpcServer:   grpcServer,
		registerFunc: registerFunc,
	}
}

func (s *Server[T]) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	err = s.reg.RegisterService(context.Background(), &server.ServiceInfo{
		Name:    s.name,
		Address: listener.Addr().String(),
		Scheme:  "grpc",
		Kind:    constant.ServiceProvider,
	})
	if err != nil {
		return err
	}
	s.registerFunc(s.Server, s.grpcServer)
	log.Printf("client grpc server %s = %s register success", s.name, addr)
	return s.Serve(listener)
}
