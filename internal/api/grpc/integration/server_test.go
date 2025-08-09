package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ego-component/eetcd/registry"
	"github.com/ego-component/egorm"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/client/egrpc"
	"github.com/gotomicro/ego/client/egrpc/resolver"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server"
	"github.com/gotomicro/ego/server/egovernor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v2"
	clientv1 "notification-platform/api/proto/gen/client/v1"
	notificationv1 "notification-platform/api/proto/gen/notification/v1"
	"notification-platform/internal/domain"
	prodioc "notification-platform/internal/ioc"
	"notification-platform/internal/pkg/retry"
	"notification-platform/internal/service/provider/sms/client"
	smsmocks "notification-platform/internal/service/provider/sms/client/mocks"
	platformioc "notification-platform/internal/test/integration/ioc/platform"
	"notification-platform/internal/test/integration/testgrpc"
	testioc "notification-platform/internal/test/ioc"

	jwtpkg "notification-platform/internal/api/grpc/interceptor/jwt"
)

const (
	batchSizeLimit            = 100
	callbackServerServiceName = "client.notification.callback.service"
)

func TestGRPCServerWithSuccessMock(t *testing.T) {
	t.Skip()
	t.Parallel()
	suite.Run(t, new(GRPCServerTestSuite))
}

type GRPCServerTestSuite struct {
	BaseGRPCServerTestSuite
}

func (s *GRPCServerTestSuite) SetupSuite() {
	// 创建成功响应的模拟客户端
	ctrl := gomock.NewController(s.T())
	mockClient := smsmocks.NewMockClient(ctrl)

	// 模拟Send方法
	mockClient.EXPECT().Send(gomock.Any()).Return(client.SendResp{
		RequestID: "mock-req-id",
		PhoneNumbers: map[string]client.SendRespStatus{
			"13800138000": {
				Code:    "OK",
				Message: "发送成功",
			},
		},
	}, nil).AnyTimes()

	// 模拟 CreateTemplate 方法
	mockClient.EXPECT().CreateTemplate(gomock.Any()).Return(client.CreateTemplateResp{
		RequestID:  "mock-req-id",
		TemplateID: "prov-tpl-001",
	}, nil).AnyTimes()

	// 模拟 BatchQueryTemplateStatus 方法
	mockClient.EXPECT().BatchQueryTemplateStatus(gomock.Any()).Return(client.BatchQueryTemplateStatusResp{
		Results: map[string]client.QueryTemplateStatusResp{
			"prov-tpl-001": {
				RequestID:   "mock-req-id",
				TemplateID:  "prov-tpl-001",
				AuditStatus: client.AuditStatusApproved,
				Reason:      "",
			},
		},
	}, nil).AnyTimes()

	// 配置参数 - 确保与providers表中的name字段匹配
	clients := map[string]client.Client{"mock-provider-1": mockClient}
	log.Printf("设置成功测试套件 Mock客户端: %+v", clients)

	serverProt := 9004
	clientAddr := fmt.Sprintf("127.0.0.1:%d", serverProt)
	s.BaseGRPCServerTestSuite.SetupTestSuite(serverProt, clientAddr, clients)
}

func (s *GRPCServerTestSuite) TearDownSuite() {
	s.BaseGRPCServerTestSuite.TearDownTestSuite()
}

// 基础测试套件
type BaseGRPCServerTestSuite struct {
	suite.Suite
	db     *egorm.Component
	server *ego.Ego
	app    *testioc.App

	clientGRPCServer *testgrpc.Server[clientv1.CallbackServiceServer]

	client      notificationv1.NotificationServiceClient
	queryClient notificationv1.NotificationQueryServiceClient

	mockClients map[string]client.Client
	serverAddr  string
	clientAddr  string
}

// 设置测试环境
func (s *BaseGRPCServerTestSuite) SetupTestSuite(serverPort int, clientAddr string, mockClients map[string]client.Client) {
	// 初始化数据库
	s.db = testioc.InitDBAndTables()
	time.Sleep(15 * time.Second)

	serverAddr := fmt.Sprintf("0.0.0.0:%d", serverPort)
	log.Printf("启动测试套件，服务器地址：%s, 客户端地址：%s\n", serverAddr, clientAddr)

	s.serverAddr = serverAddr
	s.clientAddr = clientAddr
	s.mockClients = mockClients

	// 加载配置
	dir, err := os.Getwd()
	s.Require().NoError(err)
	f, err := os.Open(dir + "/../../../../config/config.yaml")
	s.Require().NoError(err)
	err = econf.LoadFromReader(f, yaml.Unmarshal)
	s.Require().NoError(err)

	// 设置客户端配置
	econf.Set("server.grpc.port", serverPort)
	econf.Set("client", map[string]any{
		"addr":  clientAddr,
		"debug": true,
	})

	// 初始化注册中心
	etcdClient := testioc.InitEtcdClient()
	reg := registry.Load("").Build(registry.WithClientEtcd(etcdClient))
	go func() {
		s.clientGRPCServer = testgrpc.NewServer[clientv1.CallbackServiceServer](callbackServerServiceName, reg, &MockClientGRPCServer{}, func(s grpc.ServiceRegistrar, srv clientv1.CallbackServiceServer) {
			clientv1.RegisterCallbackServiceServer(s, srv)
		})
		err = s.clientGRPCServer.Start("127.0.0.1:30002")
		s.NoError(err)
	}()
	// 等待启动完成
	time.Sleep(1 * time.Second)
	resolver.Register("etcd", reg)

	// 创建服务器
	s.server = ego.New()
	setupCtx, setupCancelFunc := context.WithCancel(context.Background())

	go func() {
		// 使用指定的mock客户端创建应用
		log.Printf("开始创建应用，传入MockClients: %+v\n", s.mockClients)
		s.app = platformioc.InitGrpcServer(s.mockClients)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.app.StartTasks(ctx)

		// 初始化trace
		tp := prodioc.InitZipkinTracer()
		defer func(tp *trace.TracerProvider, ctx context.Context) {
			err := tp.Shutdown(ctx)
			if err != nil {
				elog.Error("Shutdown zipkinTracer", elog.FieldErr(err))
			}
		}(tp, ctx)

		// 设置服务器地址
		econf.Set("server.grpc", map[string]any{
			"addr": serverPort,
		})

		// 启动服务
		if err := s.server.Serve(
			egovernor.Load("server.governor").Build(),
			func() server.Server {
				setupCancelFunc()
				return s.app.GrpcServer
			}(),
		).Cron(s.app.Crons...).Run(); err != nil {
			elog.Panic("startup", elog.FieldErr(err))
		}
	}()

	// 等待服务启动
	log.Printf("等待服务启动...\n")
	select {
	case <-setupCtx.Done():
		time.Sleep(1 * time.Second)
	case <-time.After(10 * time.Second):
		s.Fail("服务启动超时")
	}

	// 创建客户端
	conn := egrpc.Load("client").Build()
	s.client = notificationv1.NewNotificationServiceClient(conn)
	s.queryClient = notificationv1.NewNotificationQueryServiceClient(conn)
}

type MockClientGRPCServer struct {
	clientv1.UnsafeCallbackServiceServer
}

func (m *MockClientGRPCServer) HandleNotificationResult(_ context.Context, _ *clientv1.HandleNotificationResultRequest) (*clientv1.HandleNotificationResultResponse, error) {
	return &clientv1.HandleNotificationResultResponse{Success: true}, nil
}

// 关闭测试环境
func (s *BaseGRPCServerTestSuite) TearDownTestSuite() {
	log.Printf("关闭测试套件，服务器地址：%s\n", s.serverAddr)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	s.NoError(s.server.Stop(ctx, false))

	s.clientGRPCServer.Stop()

	// 测试完成后清空表数据
	s.db.Exec("DELETE TABLE `callback_logs`")
	s.db.Exec("DELETE TABLE `business_configs`")
	s.db.Exec("DELETE TABLE `notifications`")
	s.db.Exec("DELETE TABLE `providers`")
	s.db.Exec("DELETE TABLE `quotas`")
	s.db.Exec("DELETE TABLE `channel_templates`")
	s.db.Exec("DELETE TABLE `channel_template_versions`")
	s.db.Exec("DELETE TABLE `channel_template_providers`")
	s.db.Exec("DELETE TABLE `tx_notifications`")
}

func (s *BaseGRPCServerTestSuite) TearDownTest() {
	// 每个测试后清空表数据
	s.db.Exec("TRUNCATE TABLE `callback_logs`")
	s.db.Exec("TRUNCATE TABLE `business_configs`")
	s.db.Exec("TRUNCATE TABLE `notifications`")
	s.db.Exec("TRUNCATE TABLE `providers`")
	s.db.Exec("TRUNCATE TABLE `quotas`")
	s.db.Exec("TRUNCATE TABLE `channel_templates`")
	s.db.Exec("TRUNCATE TABLE `channel_template_versions`")
	s.db.Exec("TRUNCATE TABLE `channel_template_providers`")
	s.db.Exec("TRUNCATE TABLE `tx_notifications`")
}

// 准备测试数据
func (s *BaseGRPCServerTestSuite) prepareTemplateData() int64 {
	templateID := time.Now().UnixNano()
	ctx := context.Background()

	// 创建SMS供应商
	provider := domain.Provider{
		ID:         1,
		Name:       "mock-provider-1",
		Channel:    domain.ChannelSMS,
		Endpoint:   "https://mock-sms-api.example.com",
		RegionID:   "cn-hangzhou",
		APIKey:     "mock-key",
		APISecret:  "mock-secret",
		Weight:     10,
		QPSLimit:   100,
		DailyLimit: 10000,
		Status:     "ACTIVE",
	}

	// 使用服务层创建供应商
	_, err := s.app.ProviderSvc.Create(ctx, provider)
	s.NoError(err)

	// 创建EMAIL供应商
	provider = domain.Provider{
		Name:       "mock-provider-1",
		Channel:    domain.ChannelEmail,
		Endpoint:   "https://mock-sms-api.example.com",
		RegionID:   "cn-hangzhou",
		APIKey:     "mock-key",
		APISecret:  "mock-secret",
		Weight:     10,
		QPSLimit:   100,
		DailyLimit: 10000,
		Status:     "ACTIVE",
	}

	// 使用服务层创建供应商
	_, err = s.app.ProviderSvc.Create(ctx, provider)
	s.NoError(err)

	// 3. 创建模板和相关记录
	s.createTemplate(ctx, templateID)

	// 设置业务配置
	const retryInterval = 10 * time.Second
	const maxRetries = 10
	config := domain.BusinessConfig{
		ID:        1,
		OwnerID:   1,
		OwnerType: "person",
		ChannelConfig: &domain.ChannelConfig{
			Channels: []domain.ChannelItem{
				{
					Channel:  "SMS",
					Priority: 1,
					Enabled:  true,
				},
			},
		},
		TxnConfig: &domain.TxnConfig{
			ServiceName:  "order.notification.callback.service",
			InitialDelay: int(time.Hour),
			RetryPolicy: &retry.Config{
				Type:          "fixed",
				FixedInterval: &retry.FixedIntervalConfig{Interval: retryInterval, MaxRetries: maxRetries},
			},
		},
		CallbackConfig: &domain.CallbackConfig{
			ServiceName: callbackServerServiceName,
			RetryPolicy: &retry.Config{
				Type:          "fixed",
				FixedInterval: &retry.FixedIntervalConfig{Interval: retryInterval, MaxRetries: maxRetries},
			},
		},
		RateLimit: 100,
		Quota: &domain.QuotaConfig{
			Monthly: domain.MonthlyConfig{
				SMS: 1000,
			},
		},
	}

	// 使用服务层设置配置
	err = s.app.ConfigSvc.SaveConfig(ctx, config)
	s.NoError(err)

	// 为其创建配额
	err = s.app.QuotaRepo.CreateOrUpdate(ctx, domain.Quota{
		BizID:   1,
		Quota:   10000,
		Channel: domain.ChannelSMS,
	})
	s.NoError(err)
	err = s.app.QuotaRepo.CreateOrUpdate(ctx, domain.Quota{
		BizID:   1,
		Quota:   10000,
		Channel: domain.ChannelEmail,
	})
	s.NoError(err)

	return templateID
}

func (s *BaseGRPCServerTestSuite) createTemplate(ctx context.Context, templateID int64) {
	now := time.Now().Unix()
	versionID := templateID + 1

	// 1. 创建渠道模板记录
	err := s.db.WithContext(ctx).Exec(`
		INSERT INTO channel_templates 
		(id, owner_id, owner_type, name, description, channel, business_type, active_version_id, ctime, utime) 
		VALUES 
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		templateID, 1, "person", "Test Template", "Template for test", "SMS", 3, versionID, now, now,
	).Error
	s.NoError(err)

	// 2. 创建模板版本记录
	err = s.db.WithContext(ctx).Exec(`
		INSERT INTO channel_template_versions 
		(id, channel_template_id, name, signature, content, remark, audit_status, ctime, utime) 
		VALUES 
		(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		versionID, templateID, "v1.0.0", "Test", "您的验证码是：${code}", "测试用的验证码模板", "APPROVED", now, now,
	).Error
	s.NoError(err)

	// 3. 创建模板版本与供应商的关联
	err = s.db.WithContext(ctx).Exec(`
		INSERT INTO channel_template_providers 
		(template_id, template_version_id, provider_id, provider_name, provider_channel, provider_template_id, audit_status, ctime, utime) 
		VALUES 
		(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		templateID, versionID, 1, "mock-provider-1", "SMS", "prov-tpl-001", "APPROVED", now, now,
	).Error
	s.NoError(err)
}

// 添加JWT认证到context
func (s *BaseGRPCServerTestSuite) contextWithJWT(ctx context.Context) context.Context {
	// 使用项目已有的JWT包创建令牌
	jwtAuth := jwtpkg.NewJwtAuthInterceptorBuilder("test_key")

	// 创建包含业务ID的声明
	claims := jwt.MapClaims{
		"biz_id": float64(1),
	}

	// 使用JWT认证包的Encode方法生成令牌
	tokenString, _ := jwtAuth.Encode(claims)

	// 创建带有授权信息的元数据
	md := metadata.New(map[string]string{
		"Authorization": "Bearer " + tokenString,
	})
	return metadata.NewOutgoingContext(ctx, md)
}

// SendNotification测试
func (s *GRPCServerTestSuite) TestSendNotification() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.SendNotificationRequest
		setupContext  func(context.Context) context.Context
		wantResp      *notificationv1.SendNotificationResponse
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "成功发送短信通知",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-1",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:         notificationv1.SendStatus_SUCCEEDED,
				NotificationId: 1, // 实际任何非零值都可接受
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用截止日期策略发送短信通知",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-deadline",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Deadline{
							Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
								Deadline: timestamppb.New(time.Now().Add(1 * time.Hour)),
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:         notificationv1.SendStatus_PENDING,
				NotificationId: 1, // 实际任何非零值都可接受
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用延迟发送策略发送短信通知",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-delayed",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Delayed{
							Delayed: &notificationv1.SendStrategy_DelayedStrategy{
								DelaySeconds: 60, // 延迟60秒
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:         notificationv1.SendStatus_PENDING,
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用定时发送策略发送短信通知",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-scheduled",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Scheduled{
							Scheduled: &notificationv1.SendStrategy_ScheduledStrategy{
								SendTime: timestamppb.New(time.Now().Add(30 * time.Minute)),
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:         notificationv1.SendStatus_PENDING,
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用时间窗口策略发送短信通知",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-timewindow",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_TimeWindow{
							TimeWindow: &notificationv1.SendStrategy_TimeWindowStrategy{
								StartTimeMilliseconds: time.Now().Unix() * 1000,
								EndTimeMilliseconds:   time.Now().Add(2*time.Hour).Unix() * 1000,
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:         notificationv1.SendStatus_PENDING,
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用立即发送策略发送短信通知",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-immediate",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Immediate{
							Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:         notificationv1.SendStatus_SUCCEEDED,
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "JWT认证失败",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-2",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				// 不添加JWT认证信息
				return ctx
			},
			wantResp:      &notificationv1.SendNotificationResponse{},
			errAssertFunc: assert.Error, // 业务错误通过响应返回，而不是通过错误
		},
		{
			name: "JWT认证信息错误",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-jwt-invalid",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				md := metadata.New(map[string]string{
					"Authorization": "Bearer invalid-token",
				})
				return metadata.NewOutgoingContext(ctx, md)
			},
			wantResp:      &notificationv1.SendNotificationResponse{},
			errAssertFunc: assert.Error,
		},
		{
			name: "空notification参数",
			req: &notificationv1.SendNotificationRequest{
				Notification: nil,
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:       notificationv1.SendStatus_FAILED,
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: "参数错误: 通知信息不能为空",
			},
			errAssertFunc: assert.NoError, // 业务错误通过响应返回，而不是通过错误
		},
		{
			name: "无效的模板ID",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-3",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: "invalid-id", // 使用非数字模板ID
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:       notificationv1.SendStatus_FAILED,
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: "参数错误: 模板ID: invalid-id",
			},
			errAssertFunc: assert.NoError, // 业务错误通过响应返回，而不是通过错误
		},
		{
			name: "未知的渠道类型",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-4",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_CHANNEL_UNSPECIFIED, // 使用未指定渠道
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:       notificationv1.SendStatus_FAILED,
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: "未知渠道类型",
			},
			errAssertFunc: assert.NoError, // 业务错误通过响应返回，而不是通过错误
		},
		{
			name: "接收者为空",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-empty-receivers",
					Receivers:  []string{},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:    notificationv1.SendStatus_FAILED,
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "缺少必要的模板参数",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:            "test-key-missing-params",
					Receivers:      []string{"13800138000"},
					Channel:        notificationv1.Channel_SMS,
					TemplateId:     strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						// 缺少"code"参数
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:       notificationv1.SendStatus_FAILED,
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: "模板参数不完整",
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "模板ID不存在",
			req: &notificationv1.SendNotificationRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-key-nonexistent-template",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: "abc",
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationResponse{
				Status:    notificationv1.SendStatus_FAILED,
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 添加JWT认证或其他上下文设置
			ctx := tt.setupContext(t.Context())

			// 调用服务
			log.Printf("发送请求: %+v", tt.req)
			resp, err := s.client.SendNotification(ctx, tt.req)

			if err != nil {
				log.Printf("发送请求失败: %v", err)
			} else {
				log.Printf("接收响应: %+v", resp)
			}

			tt.errAssertFunc(t, err)

			if err != nil {
				return
			}

			if tt.wantResp.Status == notificationv1.SendStatus_SUCCEEDED ||
				tt.wantResp.Status == notificationv1.SendStatus_PENDING {
				assert.Greater(t, resp.NotificationId, uint64(0), "成功响应应返回非零通知ID")
			} else {
				assert.Equal(t, tt.wantResp.NotificationId, resp.NotificationId, "通知ID应匹配")
			}
			assert.Equal(t, tt.wantResp.Status, resp.Status, "响应状态应匹配")
			assert.Equal(t, tt.wantResp.ErrorCode, resp.ErrorCode, "错误码应匹配")
			if tt.wantResp.ErrorCode != notificationv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
				assert.NotEmpty(t, resp.ErrorMessage, "错误消息不能为空")
			}
		})
	}
}

// SendNotificationAsync测试
func (s *GRPCServerTestSuite) TestSendNotificationAsync() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.SendNotificationAsyncRequest
		setupContext  func(context.Context) context.Context
		wantResp      *notificationv1.SendNotificationAsyncResponse
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "使用截止日期策略成功发送异步短信通知",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-deadline-key",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Deadline{
							Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
								Deadline: timestamppb.New(time.Now().Add(1 * time.Hour)),
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				NotificationId: 1, // 实际任何非零值都可接受
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用延迟发送策略成功发送异步短信通知",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-delayed-key",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Delayed{
							Delayed: &notificationv1.SendStrategy_DelayedStrategy{
								DelaySeconds: 60, // 延迟60秒
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用定时发送策略成功发送异步短信通知",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-scheduled-key",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Scheduled{
							Scheduled: &notificationv1.SendStrategy_ScheduledStrategy{
								SendTime: timestamppb.New(time.Now().Add(30 * time.Minute)),
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用时间窗口策略成功发送异步短信通知",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-timewindow-key",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_TimeWindow{
							TimeWindow: &notificationv1.SendStrategy_TimeWindowStrategy{
								StartTimeMilliseconds: time.Now().Unix() * 1000,
								EndTimeMilliseconds:   time.Now().Add(2*time.Hour).Unix() * 1000,
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用立即发送策略成功发送异步短信通知",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-immediate-key",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Immediate{
							Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				NotificationId: 1,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "成功发送异步邮件通知",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-email-1",
					Receivers:  []string{"test@example.com"},
					Channel:    notificationv1.Channel_EMAIL,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				NotificationId: 1, // 实际任何非零值都可接受
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "JWT认证失败",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-key-2",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				// 不添加JWT认证信息
				return ctx
			},
			wantResp:      &notificationv1.SendNotificationAsyncResponse{},
			errAssertFunc: assert.Error,
		},
		{
			name: "JWT认证信息错误",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-key-jwt-invalid",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				md := metadata.New(map[string]string{
					"Authorization": "Bearer invalid-token",
				})
				return metadata.NewOutgoingContext(ctx, md)
			},
			wantResp:      &notificationv1.SendNotificationAsyncResponse{},
			errAssertFunc: assert.Error,
		},
		{
			name: "空notification参数",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: nil,
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "无效的模板ID",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-key-3",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: "invalid-id", // 使用非数字模板ID
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "未知的渠道类型",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-key-4",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_CHANNEL_UNSPECIFIED, // 使用未指定渠道
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "接收者为空",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:        "test-async-key-empty-receivers",
					Receivers:  []string{},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "缺少必要的模板参数",
			req: &notificationv1.SendNotificationAsyncRequest{
				Notification: &notificationv1.Notification{
					Key:            "test-async-key-missing-params",
					Receivers:      []string{"13800138000"},
					Channel:        notificationv1.Channel_SMS,
					TemplateId:     strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						// 缺少"code"参数
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.SendNotificationAsyncResponse{
				ErrorCode: notificationv1.ErrorCode_INVALID_PARAMETER,
			},
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 添加JWT认证或其他上下文设置
			ctx := tt.setupContext(t.Context())
			// 调用服务
			log.Printf("发送异步请求: %+v", tt.req)
			resp, err := s.client.SendNotificationAsync(ctx, tt.req)

			if err != nil {
				log.Printf("发送异步请求失败: %v", err)
			} else {
				log.Printf("接收异步响应: %+v", resp)
			}

			tt.errAssertFunc(t, err)

			if err != nil {
				return
			}

			if tt.wantResp.ErrorCode == notificationv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
				assert.Greater(t, resp.NotificationId, uint64(0), "成功响应应返回非零通知ID")
			} else {
				assert.Equal(t, tt.wantResp.NotificationId, resp.NotificationId, "通知ID应匹配")
			}
			assert.Equal(t, tt.wantResp.ErrorCode, resp.ErrorCode, "错误码应匹配")
			if tt.wantResp.ErrorCode != notificationv1.ErrorCode_ERROR_CODE_UNSPECIFIED {
				assert.NotEmpty(t, resp.ErrorMessage, "错误消息不能为空")
			}
		})
	}
}

// BatchSendNotifications测试
func (s *GRPCServerTestSuite) TestBatchSendNotifications() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.BatchSendNotificationsRequest
		setupContext  func(context.Context) context.Context
		wantResp      *notificationv1.BatchSendNotificationsResponse
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "成功批量发送短信通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-key-1",
						Receivers:  []string{"13800138001"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "111111",
						},
					},
					{
						Key:        "batch-key-2",
						Receivers:  []string{"13800138002"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "222222",
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{
					{
						Status: notificationv1.SendStatus_SUCCEEDED,
					},
					{
						Status: notificationv1.SendStatus_SUCCEEDED,
					},
				},
				TotalCount:   2,
				SuccessCount: 2,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "空通知列表",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   0,
				SuccessCount: 0,
				Results:      []*notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "混合有效和无效通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-valid-key",
						Receivers:  []string{"13800138003"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "333333",
						},
					},
					{
						Key:        "batch-invalid-key",
						Receivers:  []string{"13800138004"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: "invalid-id", // 无效模板ID
						TemplateParams: map[string]string{
							"code": "444444",
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 0, // 因为有无效通知，所以成功计数为0
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "JWT认证失败",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-jwt-key",
						Receivers:  []string{"13800138005"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "555555",
						},
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				// 不添加JWT认证信息
				return ctx
			},
			wantResp:      nil,
			errAssertFunc: assert.Error, // 认证失败返回错误
		},
		{
			name: "超过批量限制",
			req: func() *notificationv1.BatchSendNotificationsRequest {
				// 创建超过限制的通知列表
				notifications := make([]*notificationv1.Notification, 101) // 假设批量限制为100
				for i := 0; i < 101; i++ {
					notifications[i] = &notificationv1.Notification{
						Key:        fmt.Sprintf("batch-ratelimit-key-%d", i),
						Receivers:  []string{fmt.Sprintf("1380013%04d", i)},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": fmt.Sprintf("%06d", i),
						},
					}
				}
				return &notificationv1.BatchSendNotificationsRequest{
					Notifications: notifications,
				}
			}(),
			setupContext:  s.contextWithJWT,
			wantResp:      nil,
			errAssertFunc: assert.Error, // 超过限制返回错误
		},
		{
			name: "使用立即发送策略批量发送通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-immediate-key-1",
						Receivers:  []string{"13800138010"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "101010",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Immediate{
								Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
							},
						},
					},
					{
						Key:        "batch-immediate-key-2",
						Receivers:  []string{"13800138011"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "111111",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Immediate{
								Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 2,
				Results: []*notificationv1.SendNotificationResponse{
					{
						Status: notificationv1.SendStatus_SUCCEEDED,
					},
					{
						Status: notificationv1.SendStatus_SUCCEEDED,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用截止日期策略批量发送通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-deadline-key-1",
						Receivers:  []string{"13800138012"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "121212",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Deadline{
								Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
									Deadline: timestamppb.New(time.Now().Add(1 * time.Hour)),
								},
							},
						},
					},
					{
						Key:        "batch-deadline-key-2",
						Receivers:  []string{"13800138013"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "131313",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Deadline{
								Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
									Deadline: timestamppb.New(time.Now().Add(2 * time.Hour)),
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 2,
				Results: []*notificationv1.SendNotificationResponse{
					{
						Status: notificationv1.SendStatus_PENDING,
					},
					{
						Status: notificationv1.SendStatus_PENDING,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用延迟发送策略批量发送通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-delayed-key-1",
						Receivers:  []string{"13800138014"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "141414",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Delayed{
								Delayed: &notificationv1.SendStrategy_DelayedStrategy{
									DelaySeconds: 60, // 延迟60秒
								},
							},
						},
					},
					{
						Key:        "batch-delayed-key-2",
						Receivers:  []string{"13800138015"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "151515",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Delayed{
								Delayed: &notificationv1.SendStrategy_DelayedStrategy{
									DelaySeconds: 120, // 延迟120秒
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 2,
				Results: []*notificationv1.SendNotificationResponse{
					{
						Status: notificationv1.SendStatus_PENDING,
					},
					{
						Status: notificationv1.SendStatus_PENDING,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用定时发送策略批量发送通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-scheduled-key-1",
						Receivers:  []string{"13800138016"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "161616",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Scheduled{
								Scheduled: &notificationv1.SendStrategy_ScheduledStrategy{
									SendTime: timestamppb.New(time.Now().Add(30 * time.Minute)),
								},
							},
						},
					},
					{
						Key:        "batch-scheduled-key-2",
						Receivers:  []string{"13800138017"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "171717",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Scheduled{
								Scheduled: &notificationv1.SendStrategy_ScheduledStrategy{
									SendTime: timestamppb.New(time.Now().Add(45 * time.Minute)),
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 2,
				Results: []*notificationv1.SendNotificationResponse{
					{
						Status: notificationv1.SendStatus_PENDING,
					},
					{
						Status: notificationv1.SendStatus_PENDING,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用时间窗口策略批量发送通知",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-timewindow-key-1",
						Receivers:  []string{"13800138018"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "181818",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_TimeWindow{
								TimeWindow: &notificationv1.SendStrategy_TimeWindowStrategy{
									StartTimeMilliseconds: time.Now().Unix() * 1000,
									EndTimeMilliseconds:   time.Now().Add(2*time.Hour).Unix() * 1000,
								},
							},
						},
					},
					{
						Key:        "batch-timewindow-key-2",
						Receivers:  []string{"13800138019"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "191919",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_TimeWindow{
								TimeWindow: &notificationv1.SendStrategy_TimeWindowStrategy{
									StartTimeMilliseconds: time.Now().Unix() * 1000,
									EndTimeMilliseconds:   time.Now().Add(3*time.Hour).Unix() * 1000,
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 2,
				Results: []*notificationv1.SendNotificationResponse{
					{
						Status: notificationv1.SendStatus_PENDING,
					},
					{
						Status: notificationv1.SendStatus_PENDING,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "不同发送策略混合发送",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-immediate-key",
						Receivers:  []string{"13800138007"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "777777",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Immediate{
								Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
							},
						},
					},
					{
						Key:        "batch-deadline-key",
						Receivers:  []string{"13800138008"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "888888",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Deadline{
								Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
									Deadline: timestamppb.New(time.Now().Add(1 * time.Hour)),
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   2,
				SuccessCount: 2,
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "五种发送策略完整混合发送",
			req: &notificationv1.BatchSendNotificationsRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-immediate-mix-key",
						Receivers:  []string{"13800138020"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "202020",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Immediate{
								Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
							},
						},
					},
					{
						Key:        "batch-deadline-mix-key",
						Receivers:  []string{"13800138021"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "212121",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Deadline{
								Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
									Deadline: timestamppb.New(time.Now().Add(1 * time.Hour)),
								},
							},
						},
					},
					{
						Key:        "batch-delayed-mix-key",
						Receivers:  []string{"13800138022"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "222222",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Delayed{
								Delayed: &notificationv1.SendStrategy_DelayedStrategy{
									DelaySeconds: 60,
								},
							},
						},
					},
					{
						Key:        "batch-scheduled-mix-key",
						Receivers:  []string{"13800138023"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "232323",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Scheduled{
								Scheduled: &notificationv1.SendStrategy_ScheduledStrategy{
									SendTime: timestamppb.New(time.Now().Add(30 * time.Minute)),
								},
							},
						},
					},
					{
						Key:        "batch-timewindow-mix-key",
						Receivers:  []string{"13800138024"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "242424",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_TimeWindow{
								TimeWindow: &notificationv1.SendStrategy_TimeWindowStrategy{
									StartTimeMilliseconds: time.Now().Unix() * 1000,
									EndTimeMilliseconds:   time.Now().Add(2*time.Hour).Unix() * 1000,
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   5,
				SuccessCount: 5,
			},
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 添加JWT认证或其他上下文设置
			ctx := tt.setupContext(t.Context())

			// 调用服务
			log.Printf("发送批量请求: %+v", tt.req)
			resp, err := s.client.BatchSendNotifications(ctx, tt.req)

			if err != nil {
				log.Printf("发送批量请求失败: %v", err)
			} else {
				log.Printf("接收批量响应: %#v", resp)
			}

			tt.errAssertFunc(t, err)

			if err != nil {
				return
			}

			// 验证响应
			assert.Equal(t, tt.wantResp.TotalCount, resp.TotalCount, "总数应匹配")
			assert.Equal(t, tt.wantResp.SuccessCount, resp.SuccessCount, "成功数应匹配")

			if tt.wantResp.TotalCount > 0 {
				assert.Equal(t, int(tt.wantResp.TotalCount), len(resp.Results), "结果数组长度应匹配总数")
			}

			// 检查成功的通知是否有非零ID
			successCount := int32(0)
			for _, result := range resp.Results {
				if result.Status == notificationv1.SendStatus_SUCCEEDED ||
					result.Status == notificationv1.SendStatus_PENDING {
					assert.Greater(t, result.NotificationId, uint64(0), "成功的通知应有非零ID")
					successCount++
				}
			}
			assert.Equal(t, resp.SuccessCount, successCount, "成功计数应与成功结果匹配")
		})
	}
}

// BatchSendNotificationsAsync测试
func (s *GRPCServerTestSuite) TestBatchSendNotificationsAsync() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.BatchSendNotificationsAsyncRequest
		setupContext  func(context.Context) context.Context
		wantResp      *notificationv1.BatchSendNotificationsAsyncResponse
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "成功批量异步发送短信通知",
			req: &notificationv1.BatchSendNotificationsAsyncRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-async-key-1",
						Receivers:  []string{"13800138001"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "111111",
						},
					},
					{
						Key:        "batch-async-key-2",
						Receivers:  []string{"13800138002"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "222222",
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsAsyncResponse{
				NotificationIds: []uint64{1, 2}, // 实际值不重要，只要是非空数组
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "空通知列表",
			req: &notificationv1.BatchSendNotificationsAsyncRequest{
				Notifications: []*notificationv1.Notification{},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsAsyncResponse{
				NotificationIds: []uint64{},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "JWT认证失败",
			req: &notificationv1.BatchSendNotificationsAsyncRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-async-jwt-key",
						Receivers:  []string{"13800138005"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "555555",
						},
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				// 不添加JWT认证信息
				return ctx
			},
			wantResp:      nil,
			errAssertFunc: assert.Error, // 认证失败返回错误
		},
		{
			name: "超过批量限制",
			req: func() *notificationv1.BatchSendNotificationsAsyncRequest {
				// 创建超过限制的通知列表
				notifications := make([]*notificationv1.Notification, 101) // 假设批量限制为100
				for i := 0; i < 101; i++ {
					notifications[i] = &notificationv1.Notification{
						Key:        fmt.Sprintf("batch-async-ratelimit-key-%d", i),
						Receivers:  []string{fmt.Sprintf("1380013%04d", i)},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": fmt.Sprintf("%06d", i),
						},
					}
				}
				return &notificationv1.BatchSendNotificationsAsyncRequest{
					Notifications: notifications,
				}
			}(),
			setupContext:  s.contextWithJWT,
			wantResp:      nil,
			errAssertFunc: assert.Error, // 超过限制返回错误
		},
		{
			name: "不同渠道混合异步发送",
			req: &notificationv1.BatchSendNotificationsAsyncRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-async-sms-key",
						Receivers:  []string{"13800138006"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "666666",
						},
					},
					{
						Key:        "batch-async-email-key",
						Receivers:  []string{"test@example.com"},
						Channel:    notificationv1.Channel_EMAIL,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "777777",
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsAsyncResponse{
				NotificationIds: []uint64{1, 2}, // 实际值不重要，只要是非空数组
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "不同发送策略混合异步发送",
			req: &notificationv1.BatchSendNotificationsAsyncRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-async-immediate-key",
						Receivers:  []string{"13800138007"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "777777",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Immediate{
								Immediate: &notificationv1.SendStrategy_ImmediateStrategy{},
							},
						},
					},
					{
						Key:        "batch-async-deadline-key",
						Receivers:  []string{"13800138008"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "888888",
						},
						Strategy: &notificationv1.SendStrategy{
							StrategyType: &notificationv1.SendStrategy_Deadline{
								Deadline: &notificationv1.SendStrategy_DeadlineStrategy{
									Deadline: timestamppb.New(time.Now().Add(1 * time.Hour)),
								},
							},
						},
					},
				},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchSendNotificationsAsyncResponse{
				NotificationIds: []uint64{1, 2}, // 实际值不重要，只要是非空数组
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "异步请求中含部分无效通知",
			req: &notificationv1.BatchSendNotificationsAsyncRequest{
				Notifications: []*notificationv1.Notification{
					{
						Key:        "batch-async-valid-key",
						Receivers:  []string{"13800138003"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: strconv.FormatInt(templateID, 10),
						TemplateParams: map[string]string{
							"code": "333333",
						},
					},
					{
						Key:        "batch-async-invalid-key",
						Receivers:  []string{"13800138004"},
						Channel:    notificationv1.Channel_SMS,
						TemplateId: "invalid-id", // 无效模板ID
						TemplateParams: map[string]string{
							"code": "444444",
						},
					},
				},
			},
			setupContext:  s.contextWithJWT,
			wantResp:      nil,
			errAssertFunc: assert.Error, // 由于一个无效通知，整个请求失败
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 添加JWT认证或其他上下文设置
			ctx := tt.setupContext(t.Context())

			// 调用服务
			log.Printf("发送批量异步请求: %+v", tt.req)
			resp, err := s.client.BatchSendNotificationsAsync(ctx, tt.req)

			if err != nil {
				log.Printf("发送批量异步请求失败: %v", err)
			} else {
				log.Printf("接收批量异步响应: %+v", resp)
			}

			tt.errAssertFunc(t, err)

			if err != nil {
				return
			}

			// 验证响应
			if len(tt.wantResp.NotificationIds) > 0 {
				assert.Greater(t, len(resp.NotificationIds), 0, "通知ID数组应非空")
				for _, id := range resp.NotificationIds {
					assert.Greater(t, id, uint64(0), "通知ID应大于0")
				}
			} else {
				assert.Empty(t, resp.NotificationIds, "通知ID数组应为空")
			}
		})
	}
}

// QueryNotification测试
func (s *GRPCServerTestSuite) TestQueryNotification() {
	// 准备测试数据
	templateID := s.prepareTemplateData()
	ctx := s.contextWithJWT(context.Background())

	// 发送通知以创建记录
	sendReq := &notificationv1.SendNotificationRequest{
		Notification: &notificationv1.Notification{
			Key:        "query-notification-key-x",
			Receivers:  []string{"13800138000"},
			Channel:    notificationv1.Channel_SMS,
			TemplateId: strconv.FormatInt(templateID, 10),
			TemplateParams: map[string]string{
				"code": "123456",
			},
		},
	}

	// 发送通知
	sendResp, err := s.client.SendNotification(ctx, sendReq)
	s.NoError(err)
	s.NotZero(sendResp.NotificationId)
	s.Equal(notificationv1.SendStatus_SUCCEEDED, sendResp.Status)

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.QueryNotificationRequest
		setupContext  func(context.Context) context.Context
		wantResp      *notificationv1.QueryNotificationResponse
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "成功查询存在的通知",
			req: &notificationv1.QueryNotificationRequest{
				Key: "query-notification-key-x",
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.QueryNotificationResponse{
				Result: &notificationv1.SendNotificationResponse{
					NotificationId: sendResp.NotificationId,
					Status:         notificationv1.SendStatus_SUCCEEDED,
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "查询不存在的通知",
			req: &notificationv1.QueryNotificationRequest{
				Key: "non-existent-notification-key",
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.QueryNotificationResponse{
				Result: &notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "查询空Key",
			req: &notificationv1.QueryNotificationRequest{
				Key: "",
			},
			setupContext:  s.contextWithJWT,
			wantResp:      nil,
			errAssertFunc: assert.Error,
		},
		{
			name: "JWT认证失败",
			req: &notificationv1.QueryNotificationRequest{
				Key: "query-notification-key",
			},
			setupContext: func(ctx context.Context) context.Context {
				// 不添加JWT认证
				return ctx
			},
			wantResp:      nil,
			errAssertFunc: assert.Error,
		},
		{
			name:          "请求对象为nil",
			req:           nil,
			setupContext:  s.contextWithJWT,
			wantResp:      nil,
			errAssertFunc: assert.Error,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 设置上下文
			ctx := tt.setupContext(t.Context())

			// 调用服务
			log.Printf("查询通知请求: %+v", tt.req)
			resp, err := s.queryClient.QueryNotification(ctx, tt.req)
			log.Printf("查询通知响应: %+v, 错误: %v", resp, err)

			// 验证结果
			tt.errAssertFunc(t, err)
			if err != nil {
				return
			}

			assert.NotNil(t, resp, "响应不应为nil")
			assert.NotNil(t, resp.Result, "结果不应为nil")

			if tt.wantResp.Result.NotificationId > 0 {
				assert.Equal(t, tt.wantResp.Result.NotificationId, resp.Result.NotificationId)
				assert.Equal(t, tt.wantResp.Result.Status, resp.Result.Status)
			} else {
				// 空结果或不存在的Key
				assert.Zero(t, resp.Result.NotificationId)
			}
		})
	}
}

// BatchQueryNotifications测试
func (s *GRPCServerTestSuite) TestBatchQueryNotifications() {
	// 准备测试数据
	templateID := s.prepareTemplateData()
	ctx := s.contextWithJWT(context.Background())

	// 发送多条通知以创建记录
	keys := []string{"batch-query-key-x", "batch-query-key-y", "batch-query-key-z"}
	notificationIDs := make([]uint64, len(keys))

	for i, key := range keys {
		sendReq := &notificationv1.SendNotificationRequest{
			Notification: &notificationv1.Notification{
				Key:        key,
				Receivers:  []string{"1394601380" + strconv.Itoa(i)},
				Channel:    notificationv1.Channel_SMS,
				TemplateId: strconv.FormatInt(templateID, 10),
				TemplateParams: map[string]string{
					"code": "12345" + strconv.Itoa(i),
				},
			},
		}

		// 发送通知
		sendResp, err := s.client.SendNotification(ctx, sendReq)
		s.NoError(err)
		s.NotZero(sendResp.NotificationId)
		s.Equal(notificationv1.SendStatus_SUCCEEDED, sendResp.Status)
		notificationIDs[i] = sendResp.NotificationId
	}

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.BatchQueryNotificationsRequest
		setupContext  func(context.Context) context.Context
		wantResp      *notificationv1.BatchQueryNotificationsResponse
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "成功批量查询存在的通知",
			req: &notificationv1.BatchQueryNotificationsRequest{
				Keys: keys,
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{
					{
						NotificationId: notificationIDs[0],
						Status:         notificationv1.SendStatus_SUCCEEDED,
					},
					{
						NotificationId: notificationIDs[1],
						Status:         notificationv1.SendStatus_SUCCEEDED,
					},
					{
						NotificationId: notificationIDs[2],
						Status:         notificationv1.SendStatus_SUCCEEDED,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "查询单条通知",
			req: &notificationv1.BatchQueryNotificationsRequest{
				Keys: []string{keys[0]},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{
					{
						NotificationId: notificationIDs[0],
						Status:         notificationv1.SendStatus_SUCCEEDED,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "查询空列表",
			req: &notificationv1.BatchQueryNotificationsRequest{
				Keys: []string{},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "部分存在部分不存在的Key",
			req: &notificationv1.BatchQueryNotificationsRequest{
				Keys: []string{keys[0], "non-existent-key-1", keys[2], "non-existent-key-2"},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{
					{
						NotificationId: notificationIDs[0],
						Status:         notificationv1.SendStatus_SUCCEEDED,
					},
					{
						NotificationId: notificationIDs[2],
						Status:         notificationv1.SendStatus_SUCCEEDED,
					},
				},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "全部不存在的Key",
			req: &notificationv1.BatchQueryNotificationsRequest{
				Keys: []string{"non-key-1", "non-key-2"},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "查询接近批量限制的数量",
			req: func() *notificationv1.BatchQueryNotificationsRequest {
				closeToLimitKeys := make([]string, batchSizeLimit-1)
				for i := 0; i < batchSizeLimit-1; i++ {
					closeToLimitKeys[i] = fmt.Sprintf("close-to-ratelimit-key-%d", i)
				}
				return &notificationv1.BatchQueryNotificationsRequest{
					Keys: closeToLimitKeys,
				}
			}(),
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "查询超过批量限制的数量",
			req: func() *notificationv1.BatchQueryNotificationsRequest {
				n := batchSizeLimit + 1
				closeToLimitKeys := make([]string, n)
				for i := 0; i < n; i++ {
					closeToLimitKeys[i] = fmt.Sprintf("over-ratelimit-key-%d", i)
				}
				return &notificationv1.BatchQueryNotificationsRequest{
					Keys: closeToLimitKeys,
				}
			}(),
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.Error,
		},
		{
			name: "查询包含非ASCII字符的Key",
			req: &notificationv1.BatchQueryNotificationsRequest{
				Keys: []string{"中文key-1", "中文key-2", "英文key-3"},
			},
			setupContext: s.contextWithJWT,
			wantResp: &notificationv1.BatchQueryNotificationsResponse{
				Results: []*notificationv1.SendNotificationResponse{},
			},
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 设置上下文
			ctx := tt.setupContext(t.Context())

			// 调用服务
			log.Printf("批量查询通知请求: %+v", tt.req)
			resp, err := s.queryClient.BatchQueryNotifications(ctx, tt.req)
			log.Printf("批量查询通知响应: %+v, 错误: %v", resp, err)

			// 验证结果
			tt.errAssertFunc(t, err)
			if err != nil {
				return
			}

			assert.NotNil(t, resp, "响应不应为nil")

			// 验证结果数量
			if len(tt.wantResp.Results) > 0 {
				// 对于存在的情况，检查返回的结果是否与预期一致
				assert.Equal(t, len(tt.wantResp.Results), len(resp.Results), "结果数量应一致")
				for i, result := range tt.wantResp.Results {
					assert.Equal(t, result.NotificationId, resp.Results[i].NotificationId, "通知ID应一致")
					assert.Equal(t, result.Status, resp.Results[i].Status, "状态应一致")
				}
			} else {
				// 空请求
				assert.Empty(t, resp.Results, "空请求的结果应为空")
			}
		})
	}
}

// TxPrepare测试
func (s *GRPCServerTestSuite) TestTxPrepare() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		req           *notificationv1.TxPrepareRequest
		setupContext  func(context.Context) context.Context
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "成功准备短信通知事务",
			req: &notificationv1.TxPrepareRequest{
				Notification: &notificationv1.Notification{
					Key:        "tx-prepare-sms-key",
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name: "成功准备邮件通知事务",
			req: &notificationv1.TxPrepareRequest{
				Notification: &notificationv1.Notification{
					Key:        "tx-prepare-email-key",
					Receivers:  []string{"test@example.com"},
					Channel:    notificationv1.Channel_EMAIL,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用延迟发送策略",
			req: &notificationv1.TxPrepareRequest{
				Notification: &notificationv1.Notification{
					Key:        "tx-prepare-delayed-key",
					Receivers:  []string{"13800138001"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Delayed{
							Delayed: &notificationv1.SendStrategy_DelayedStrategy{
								DelaySeconds: 60,
							},
						},
					},
				},
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name: "使用计划发送策略",
			req: &notificationv1.TxPrepareRequest{
				Notification: &notificationv1.Notification{
					Key:        "tx-prepare-scheduled-key",
					Receivers:  []string{"13800138002"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
					Strategy: &notificationv1.SendStrategy{
						StrategyType: &notificationv1.SendStrategy_Scheduled{
							Scheduled: &notificationv1.SendStrategy_ScheduledStrategy{
								SendTime: timestamppb.New(time.Now().Add(30 * time.Minute)),
							},
						},
					},
				},
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			// 设置上下文
			ctx := tt.setupContext(t.Context())

			// 调用服务
			log.Printf("TxPrepare请求: %+v", tt.req)
			resp, err := s.client.TxPrepare(ctx, tt.req)
			log.Printf("TxPrepare响应: %+v, 错误: %v", resp, err)

			// 验证结果
			tt.errAssertFunc(t, err)
			if err == nil {
				assert.NotNil(t, resp, "响应不应为nil")
			}
		})
	}
}

// TxCommit测试
func (s *GRPCServerTestSuite) TestTxCommit() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		prepareKey    string
		commitReq     *notificationv1.TxCommitRequest
		setupContext  func(context.Context) context.Context
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name:       "成功提交短信通知事务",
			prepareKey: "tx-commit-sms-key",
			commitReq: &notificationv1.TxCommitRequest{
				Key: "tx-commit-sms-key",
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name:       "成功提交邮件通知事务",
			prepareKey: "tx-commit-email-key",
			commitReq: &notificationv1.TxCommitRequest{
				Key: "tx-commit-email-key",
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name:       "成功提交带延迟策略的通知事务",
			prepareKey: "tx-commit-delayed-key",
			commitReq: &notificationv1.TxCommitRequest{
				Key: "tx-commit-delayed-key",
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext(t.Context())

			// 先准备一个事务
			prepareReq := &notificationv1.TxPrepareRequest{
				Notification: &notificationv1.Notification{
					Key:        tt.prepareKey,
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			}

			// 如果是延迟策略的测试
			if tt.name == "成功提交带延迟策略的通知事务" {
				prepareReq.Notification.Strategy = &notificationv1.SendStrategy{
					StrategyType: &notificationv1.SendStrategy_Delayed{
						Delayed: &notificationv1.SendStrategy_DelayedStrategy{
							DelaySeconds: 60,
						},
					},
				}
			} else if tt.name == "成功提交邮件通知事务" {
				prepareReq.Notification.Channel = notificationv1.Channel_EMAIL
				prepareReq.Notification.Receivers = []string{"test@example.com"}
			}

			// 先执行准备
			_, err := s.client.TxPrepare(ctx, prepareReq)
			assert.NoError(t, err, "准备事务应成功")

			// 调用TxCommit
			log.Printf("TxCommit请求: %+v", tt.commitReq)
			resp, err := s.client.TxCommit(ctx, tt.commitReq)
			log.Printf("TxCommit响应: %+v, 错误: %v", resp, err)

			// 验证结果
			tt.errAssertFunc(t, err)
			if err == nil {
				assert.NotNil(t, resp, "响应不应为nil")
			}
		})
	}
}

// TxCancel测试
func (s *GRPCServerTestSuite) TestTxCancel() {
	// 准备测试数据
	templateID := s.prepareTemplateData()

	// 测试用例
	testCases := []struct {
		name          string
		prepareKey    string
		cancelReq     *notificationv1.TxCancelRequest
		setupContext  func(context.Context) context.Context
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name:       "成功取消短信通知事务",
			prepareKey: "tx-cancel-sms-key",
			cancelReq: &notificationv1.TxCancelRequest{
				Key: "tx-cancel-sms-key",
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name:       "成功取消邮件通知事务",
			prepareKey: "tx-cancel-email-key",
			cancelReq: &notificationv1.TxCancelRequest{
				Key: "tx-cancel-email-key",
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
		{
			name:       "成功取消带时间窗口策略的通知事务",
			prepareKey: "tx-cancel-window-key",
			cancelReq: &notificationv1.TxCancelRequest{
				Key: "tx-cancel-window-key",
			},
			setupContext:  s.contextWithJWT,
			errAssertFunc: assert.NoError,
		},
	}

	for _, tt := range testCases {
		s.T().Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext(t.Context())

			// 先准备一个事务
			prepareReq := &notificationv1.TxPrepareRequest{
				Notification: &notificationv1.Notification{
					Key:        tt.prepareKey,
					Receivers:  []string{"13800138000"},
					Channel:    notificationv1.Channel_SMS,
					TemplateId: strconv.FormatInt(templateID, 10),
					TemplateParams: map[string]string{
						"code": "123456",
					},
				},
			}

			// 根据测试用例设置不同的通知类型和策略
			if tt.name == "成功取消带时间窗口策略的通知事务" {
				prepareReq.Notification.Strategy = &notificationv1.SendStrategy{
					StrategyType: &notificationv1.SendStrategy_TimeWindow{
						TimeWindow: &notificationv1.SendStrategy_TimeWindowStrategy{
							StartTimeMilliseconds: time.Now().Unix(),
							EndTimeMilliseconds:   time.Now().Add(2 * time.Hour).Unix(),
						},
					},
				}
			} else if tt.name == "成功取消邮件通知事务" {
				prepareReq.Notification.Channel = notificationv1.Channel_EMAIL
				prepareReq.Notification.Receivers = []string{"test@example.com"}
			}

			// 先执行准备
			_, err := s.client.TxPrepare(ctx, prepareReq)
			assert.NoError(t, err, "准备事务应成功")

			// 调用TxCancel
			log.Printf("TxCancel请求: %+v", tt.cancelReq)
			resp, err := s.client.TxCancel(ctx, tt.cancelReq)
			log.Printf("TxCancel响应: %+v, 错误: %v", resp, err)

			// 验证结果
			tt.errAssertFunc(t, err)
			if err == nil {
				assert.NotNil(t, resp, "响应不应为nil")
			}
		})
	}
}
