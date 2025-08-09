package integration

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	helloworldv1 "notification-platform/api/proto/gen/helloworld/v1"
	"notification-platform/internal/api/grpc/interceptor/idempotent"
	"notification-platform/internal/api/grpc/interceptor/jwt"
	pkgIdempotent "notification-platform/internal/pkg/idempotent"
	"notification-platform/internal/test/ioc"
)

// 使用适配器模式包装SayHelloRequest
type SayHelloRequestAdapter struct {
	*helloworldv1.SayHelloRequest
}

// 实现IdempotencyCarrier接口
func (s *SayHelloRequestAdapter) GetIdempotencyKeys() []string {
	return s.GetKeys()
}

type IdempotentTestSuite struct {
	suite.Suite
	cache         redis.Cmdable
	idempotentSvc pkgIdempotent.IdempotencyService
}

func TestIdempotentSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IdempotentTestSuite))
}

func (s *IdempotentTestSuite) SetupSuite() {
	s.cache = ioc.InitRedis()
	s.idempotentSvc = pkgIdempotent.NewRedisIdempotencyService(s.cache, 60*time.Second)
}

// GreeterService实现
type greeterServiceServer struct {
	helloworldv1.UnimplementedGreeterServiceServer
}

// SayHello方法实现
func (s *greeterServiceServer) SayHello(_ context.Context, req *helloworldv1.SayHelloRequest) (*helloworldv1.SayHelloResponse, error) {
	return &helloworldv1.SayHelloResponse{
		Message: "Hello, " + req.GetName(),
	}, nil
}

// 测试幂等拦截器
// 测试逻辑：
// 1. 第一次调用GreeterService的SayHello方法成功
// 2. 使用相同的幂等键再次调用，应返回幂等错误
// 3. 使用不同的幂等键调用，应成功
func (s *IdempotentTestSuite) TestIdempotentWithGreeterService() {
	t := s.T()

	// 创建一个gRPC服务器
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// 使用我们的幂等服务
	idempotentService := s.idempotentSvc

	// 添加JWT和幂等拦截器
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// JWT拦截器模拟 - 简化测试，提供biz-id
			func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				md, ok := metadata.FromIncomingContext(ctx)
				if !ok {
					return nil, status.Error(codes.InvalidArgument, "没有metadata")
				}
				bizIDs := md.Get("biz-id")
				if len(bizIDs) == 0 {
					return nil, status.Error(codes.InvalidArgument, "没有biz-id")
				}
				// 设置bizID到context中
				ctx = context.WithValue(ctx, jwt.BizIDName, int64(1001))
				return handler(ctx, req)
			},
			// 幂等拦截器，这里创建一个闭包函数，内部直接使用我们的idempotentService
			idempotent.NewIdempotentBuilder(idempotentService).Build(),
		),
	)

	// 注册GreeterService
	helloworldv1.RegisterGreeterServiceServer(srv, &greeterServiceServer{})

	go func() {
		err := srv.Serve(lis)
		require.NoError(t, err)
	}()
	defer srv.Stop()

	// 创建gRPC客户端
	conn, err := grpc.Dial(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// 创建GreeterService客户端
	client := helloworldv1.NewGreeterServiceClient(conn)

	// 测试场景1：第一次调用，应该成功
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("biz-id", "1001"))
	req := &helloworldv1.SayHelloRequest{
		Name: "World",
		Keys: []string{"test-key-1"}, // 用作幂等键
	}

	resp, err := client.SayHello(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World", resp.GetMessage())

	// 测试场景2：使用相同的幂等键再次调用，应该失败
	_, err = client.SayHello(ctx, req)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "幂等检测没通过"))

	// 测试场景3：使用不同的幂等键调用，应该成功
	req.Keys = []string{"test-key-2"}
	resp, err = client.SayHello(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World", resp.GetMessage())
}
