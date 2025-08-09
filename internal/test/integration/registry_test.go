//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net"
	"notification-platform/internal/pkg/grpcx"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ego-component/eetcd"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	helloworldv1 "notification-platform/api/proto/gen/helloworld/v1"
	"notification-platform/internal/pkg/grpcx/loadbalancer"
	"notification-platform/internal/pkg/registry"
	"notification-platform/internal/pkg/registry/etcd"
	testioc "notification-platform/internal/test/ioc"
)

func TestRegistrySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RegistryTestSuite))
}

type RegistryTestSuite struct {
	suite.Suite
	etcdClient *eetcd.Component
}

func (s *RegistryTestSuite) SetupSuite() {
	s.etcdClient = testioc.InitEtcdClient()
}

// 添加一个ServerInfo结构体用于安全传递服务器信息
type ServerInfo struct {
	addr string
	err  error
}

// TestGroup 测试基于分组的服务发现和请求路由功能
// 本测试验证：
// 1. 启动4个服务器实例：3个group=core，1个group=""
// 2. 客户端通过在Context中设置group="core"标签
// 3. 验证所有请求都路由到了group=core的服务器
func (s *RegistryTestSuite) TestGroup() {
	t := s.T()

	// 创建etcd注册中心实例
	etcdRegistry, err := etcd.NewRegistry(s.etcdClient)
	require.NoError(t, err)
	defer etcdRegistry.Close()

	// 定义超时时间
	timeout := time.Second * 10

	// 创建4个服务器实例，3个设置group=core，1个设置group=""
	var servers []*Server
	var coreServers []*greeterServer
	var defaultServers []*greeterServer

	// 使用通道来安全地获取服务器地址
	for i := 0; i < 3; i++ {
		gs := &greeterServer{group: "core"}
		name := fmt.Sprintf("greeter-%d", i)
		srv := NewServer(name,
			ServerWithRegistry(etcdRegistry),
			ServerWithTimeout(timeout),
			ServerWithGroup("core"))

		// 注册greeter服务
		helloworldv1.RegisterGreeterServiceServer(srv.Server, gs)

		// 使用通道获取服务器地址
		addrCh := make(chan ServerInfo, 1)

		// 使用随机端口启动服务
		go func() {
			// 先获取监听器但不启动服务
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				addrCh <- ServerInfo{err: err}
				return
			}

			// 安全地获取地址
			addr := listener.Addr().String()
			addrCh <- ServerInfo{addr: addr}

			// 设置监听器并启动服务
			srv.listener = listener

			// 必须调用Start方法以执行注册逻辑，使用空地址因为监听器已经设置
			_ = srv.Start("")
		}()

		// 等待地址准备好或出错
		serverInfo := <-addrCh
		require.NoError(t, serverInfo.err, "启动服务失败")

		// 现在可以安全地使用地址
		gs.addr = serverInfo.addr
		servers = append(servers, srv)
		coreServers = append(coreServers, gs)
	}

	// 启动1个group=""的默认服务
	gs := &greeterServer{group: ""}
	srv := NewServer("greeter",
		ServerWithRegistry(etcdRegistry),
		ServerWithTimeout(timeout),
		ServerWithGroup(""))

	helloworldv1.RegisterGreeterServiceServer(srv.Server, gs)

	// 使用通道获取服务器地址
	addrCh := make(chan ServerInfo, 1)

	go func() {
		// 先获取监听器但不启动服务
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			addrCh <- ServerInfo{err: err}
			return
		}

		// 安全地获取地址
		addr := listener.Addr().String()
		addrCh <- ServerInfo{addr: addr}

		// 设置监听器并启动服务
		srv.listener = listener

		// 必须调用Start方法以执行注册逻辑，使用空地址因为监听器已经设置
		_ = srv.Start("")
	}()

	// 等待地址准备好或出错
	serverInfo := <-addrCh
	require.NoError(t, serverInfo.err, "启动服务失败")

	// 现在可以安全地使用地址
	gs.addr = serverInfo.addr
	servers = append(servers, srv)
	defaultServers = append(defaultServers, gs)

	// 确保所有服务器在测试结束后关闭
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	// 等待服务注册完成
	time.Sleep(time.Second * 3)

	// 列出所有服务，确认注册成功
	services, err := etcdRegistry.ListServices(context.Background(), "greeter")
	require.NoError(t, err)
	require.Equal(t, 4, len(services), "应该有4个服务实例注册成功")

	// 创建客户端
	c := NewClient(
		ClientWithRegistry(etcdRegistry, timeout),
		ClientWithInsecure(),
		ClientWithPickerBuilder("weight", loadbalancer.NewWeightPickerBuilder()),
	)

	// 连接服务
	conn, err := c.Dial("greeter")
	require.NoError(t, err)
	defer conn.Close()

	// 创建客户端
	client := helloworldv1.NewGreeterServiceClient(conn)

	// 稍等片刻，确保连接建立
	time.Sleep(time.Second)

	// 发起5次调用，验证所有请求都路由到了group=core的服务器
	for i := 0; i < 5; i++ {
		reqCtx, reqCancel := context.WithTimeout(context.Background(), timeout)

		// 在context中设置group="core"，使请求被路由到core组的服务器
		reqCtx = loadbalancer.WithGroup(reqCtx, "core")

		// 发送请求
		resp, err := client.SayHello(
			reqCtx,
			&helloworldv1.SayHelloRequest{Name: fmt.Sprintf("test-%d", i)},
		)
		reqCancel()
		require.NoError(t, err)

		// 验证响应包含了服务器地址和组信息
		s.Contains(resp.Message, "group=core", "响应应该来自core组服务器")

		// 短暂等待，避免请求过于密集
		time.Sleep(time.Millisecond * 100)
	}

	// 等待一会确保所有请求都已完成处理
	time.Sleep(time.Second)

	// 验证请求分布情况
	var coreTotal int32
	for _, s := range coreServers {
		coreTotal += s.reqCnt.Load()
	}

	var defaultTotal int32
	for _, s := range defaultServers {
		defaultTotal += s.reqCnt.Load()
	}

	// 验证所有请求都发送到了group=core的服务器
	s.Equal(int32(5), coreTotal, "所有请求应该被发送到group=core的服务器")
	s.Equal(int32(0), defaultTotal, "group=''的服务器不应该收到请求")

	// 验证请求在core组服务器之间有分布
	var nonZeroServers int
	for _, server := range coreServers {
		if server.reqCnt.Load() > 0 {
			nonZeroServers++
		}
	}
	s.Greater(nonZeroServers, 0, "至少有一个core组服务器应该收到请求")
}

// 创建Greeter服务的实现
type greeterServer struct {
	helloworldv1.UnimplementedGreeterServiceServer
	addr   string
	group  string
	reqCnt atomic.Int32
}

// 实现SayHello方法
func (g *greeterServer) SayHello(_ context.Context, req *helloworldv1.SayHelloRequest) (*helloworldv1.SayHelloResponse, error) {
	g.reqCnt.Add(1)
	return &helloworldv1.SayHelloResponse{
		Message: fmt.Sprintf("Hello %s from server %s in group=%s", req.Name, g.addr, g.group),
	}, nil
}

type Server struct {
	name     string
	listener net.Listener

	si       registry.ServiceInstance
	registry registry.Registry
	// 单个操作的超时时间，一般用于和注册中心打交道
	timeout time.Duration
	*grpc.Server

	readWeight  int32
	writeWeight int32
	group       string
}

func NewServer(name string, opts ...ServerOption) *Server {
	res := &Server{
		name:        name,
		Server:      grpc.NewServer(),
		readWeight:  1,
		writeWeight: 1,
	}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func ServerWithGroup(group string) ServerOption {
	return func(server *Server) {
		server.group = group
	}
}

func ServerWithRegistry(r registry.Registry) ServerOption {
	return func(server *Server) {
		server.registry = r
	}
}

func ServerWithTimeout(timeout time.Duration) ServerOption {
	return func(server *Server) {
		server.timeout = timeout
	}
}

func ServerWithReadWeight(readWeight int32) ServerOption {
	return func(server *Server) {
		server.readWeight = readWeight
	}
}

func ServerWithWriteWeight(writeWeight int32) ServerOption {
	return func(server *Server) {
		server.writeWeight = writeWeight
	}
}

type ServerOption func(server *Server)

// 需要修改Server的Start方法，使其支持外部传入的listener
func (s *Server) Start(addr string) error {
	// 如果已经有listener，直接使用
	if s.listener == nil {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		s.listener = listener
	}

	// 用户决定使用注册中心
	if s.registry != nil {
		s.si = registry.ServiceInstance{
			Name:        s.name,
			Address:     s.listener.Addr().String(),
			ReadWeight:  s.readWeight,
			WriteWeight: s.writeWeight,
			Group:       s.group,
		}
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		err := s.registry.Register(ctx, s.si)
		cancel()
		if err != nil {
			return err
		}
	}
	return s.Server.Serve(s.listener)
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	if s.registry != nil {
		if err := s.registry.UnRegister(ctx, s.si); err != nil {
			return err
		}
	}
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

type ClientOption func(client *Client)

type Client struct {
	rb       resolver.Builder
	insecure bool
	balancer balancer.Builder
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *Client) Dial(service string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{grpc.WithResolvers(c.rb), grpc.WithNoProxy()}
	address := fmt.Sprintf("registry:///%s", service)
	if c.insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if c.balancer != nil {
		opts = append(opts, grpc.WithDefaultServiceConfig(
			fmt.Sprintf(`{"loadBalancingPolicy": %q}`,
				c.balancer.Name())))
	}
	return grpc.NewClient(address, opts...)
}

func ClientWithRegistry(r registry.Registry, timeout time.Duration) ClientOption {
	return func(client *Client) {
		client.rb = grpcx.NewResolverBuilder(r, timeout)
	}
}

func ClientWithInsecure() ClientOption {
	return func(client *Client) {
		client.insecure = true
	}
}

func ClientWithPickerBuilder(name string, b base.PickerBuilder) ClientOption {
	return func(client *Client) {
		builder := base.NewBalancerBuilder(name, b, base.Config{HealthCheck: true})
		balancer.Register(builder)
		client.balancer = builder
	}
}
