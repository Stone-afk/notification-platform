package ioc

import (
	"github.com/ego-component/eetcd"
	"github.com/ego-component/eetcd/registry"
	"github.com/gotomicro/ego/client/egrpc/resolver"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egrpc"
	"go.opentelemetry.io/otel"
	notificationv1 "notification-platform/api/proto/gen/notification/v1"
	grpcapi "notification-platform/internal/api/grpc"
	"notification-platform/internal/api/grpc/interceptor/jwt"
	"notification-platform/internal/api/grpc/interceptor/log"
	"notification-platform/internal/api/grpc/interceptor/metrics"
	"notification-platform/internal/api/grpc/interceptor/tracing"
)

const (
	// 用于OpenTelemetry跟踪的仪表名
	instrumentationName = "internal/api/grpc/interceptor/tracing"
)

func InitGrpc(noserver *grpcapi.NotificationServer, etcdClient *eetcd.Component) *egrpc.Component {
	// 注册全局的注册中心
	type Config struct {
		Key string `yaml:"key"`
	}
	var cfg Config

	err := econf.UnmarshalKey("jwt", &cfg)
	if err != nil {
		panic("config err:" + err.Error())
	}

	reg := registry.Load("").Build(registry.WithClientEtcd(etcdClient))
	resolver.Register("etcd", reg)

	// 创建observability拦截器
	metricsInterceptor := metrics.NewMetricsBuilder().Build()
	// 创建日志拦截器
	logInterceptor := log.NewLoggerBuilder().Build()

	traceInterceptor := tracing.NewTracingBuilder(otel.GetTracerProvider().Tracer(instrumentationName)).Build()
	jwtinterceter := jwt.NewJwtAuthInterceptorBuilder(cfg.Key).JwtAuthInterceptor()
	server := egrpc.Load("server.grpc").Build(
		egrpc.WithUnaryInterceptor(metricsInterceptor, logInterceptor, traceInterceptor, jwtinterceter),
	)

	notificationv1.RegisterNotificationServiceServer(server.Server, noserver)
	notificationv1.RegisterNotificationQueryServiceServer(server.Server, noserver)

	return server
}
