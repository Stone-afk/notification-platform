//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	helloworldv1 "notification-platform/api/proto/gen/helloworld/v1"
	"notification-platform/internal/api/grpc/interceptor/timeout"
	"notification-platform/internal/test/ioc"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestTimeoutSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TimeoutTestSuite))
}

type timeoutServer struct {
	helloworldv1.UnimplementedGreeterServiceServer
	cache redis.Cmdable
}

// 实现SayHello方法
func (g *timeoutServer) SayHello(ctx context.Context, _ *helloworldv1.SayHelloRequest) (*helloworldv1.SayHelloResponse, error) {
	err := g.f1(ctx)
	if err != nil {
		return nil, err
	}
	err = g.f2(ctx)
	if err != nil {
		return nil, err
	}
	//
	err = g.f3(ctx)
	if err != nil {
		return nil, err
	}
	return &helloworldv1.SayHelloResponse{}, nil
}

func (g *timeoutServer) f1(ctx context.Context) error {
	time.Sleep(1 * time.Second)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	g.cache.LPush(ctx, "timeout:test", "f1")
	return nil
}

func (g *timeoutServer) f2(ctx context.Context) error {
	time.Sleep(2 * time.Second)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	g.cache.LPush(ctx, "timeout:test", "f2")
	return nil
}

func (g *timeoutServer) f3(ctx context.Context) error {
	time.Sleep(6 * time.Second) // 超过5秒超时时间
	if ctx.Err() != nil {
		return ctx.Err()
	}
	fmt.Println("f3xxxxxxx")
	g.cache.LPush(ctx, "timeout:test", "f3")
	return nil
}

type TimeoutTestSuite struct {
	suite.Suite
	cache redis.Cmdable
}

func (s *TimeoutTestSuite) SetupSuite() {
	s.cache = ioc.InitRedis()
}

// 测试链路超时
// 测试逻辑： 超时时间为5秒，会在调用f3方法的时候超时退出
func (s *TimeoutTestSuite) TestTimeout() {
	t := s.T()

	// 创建gRPC服务端
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	srv := grpc.NewServer(
		grpc.ConnectionTimeout(10*time.Second),
		grpc.ChainUnaryInterceptor(
			timeout.ServerTimeoutInterceptor(),
		),
	)
	helloworldv1.RegisterGreeterServiceServer(srv, &timeoutServer{
		cache: s.cache,
	})
	go func() {
		// 启动服务端
		err := srv.Serve(lis)
		require.NoError(t, err)
	}()
	defer srv.Stop()

	// 创建gRPC客户端
	conn, _ := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			// 注入客户端超时拦截器
			timeout.InjectorTimeoutInterceptor(),
		),
	)
	defer func() {
		err := conn.Close()
		require.NoError(t, err)
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_, err := helloworldv1.NewGreeterServiceClient(conn).SayHello(ctx, &helloworldv1.SayHelloRequest{})
	require.Error(t, err)

	// 检验缓存中的数据
	elements, err := s.cache.LRange(t.Context(), "timeout:test", 0, -1).Result()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, []string{
		"f2",
		"f1",
	}, elements)
	s.cache.Del(t.Context(), "timeout:test")
}
