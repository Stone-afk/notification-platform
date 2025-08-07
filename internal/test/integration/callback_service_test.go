package integration

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/ego-component/eetcd/registry"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/client/egrpc/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	clientv1 "notification-platform/api/proto/gen/client/v1"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/retry"
	configmocks "notification-platform/internal/service/config/mocks"
	callbackioc "notification-platform/internal/test/integration/ioc/callback"
	"notification-platform/internal/test/integration/testgrpc"
	testioc "notification-platform/internal/test/ioc"
)

const (
	callbackServerServiceName = "client2.notification.callback.service"
)

func TestNotificationCallbackServiceSuite(t *testing.T) {
	suite.Run(t, new(NotificationCallbackServiceTestSuite))
}

type NotificationCallbackServiceTestSuite struct {
	suite.Suite

	clientGRPCServer *testgrpc.Server[clientv1.CallbackServiceServer]
	db               *egorm.Component
}

func (s *NotificationCallbackServiceTestSuite) SetupSuite() {
	s.db = testioc.InitDBAndTables()

	// 初始化注册中心
	etcdClient := testioc.InitEtcdClient()
	reg := registry.Load("").Build(registry.WithClientEtcd(etcdClient))
	go func() {
		s.clientGRPCServer = testgrpc.NewServer[clientv1.CallbackServiceServer](callbackServerServiceName, reg, &MockClientGRPCServer{
			shouldSucceed: true,
		}, func(s grpc.ServiceRegistrar, srv clientv1.CallbackServiceServer) {
			clientv1.RegisterCallbackServiceServer(s, srv)
		})
		err := s.clientGRPCServer.Start("127.0.0.1:30012")
		s.NoError(err)
	}()
	// 等待启动完成
	time.Sleep(1 * time.Second)
	resolver.Register("etcd", reg)
}

type MockClientGRPCServer struct {
	clientv1.UnsafeCallbackServiceServer
	// 添加控制响应的字段
	shouldSucceed bool
}

func (m *MockClientGRPCServer) HandleNotificationResult(_ context.Context, _ *clientv1.HandleNotificationResultRequest) (*clientv1.HandleNotificationResultResponse, error) {
	return &clientv1.HandleNotificationResultResponse{Success: m.shouldSucceed}, nil
}

func (s *NotificationCallbackServiceTestSuite) TearDownTest() {
	// 每个测试后清空表数据
	s.db.Exec("TRUNCATE TABLE `notifications`")
	s.db.Exec("TRUNCATE TABLE `quotas`")
	s.db.Exec("TRUNCATE TABLE `callback_logs`")
}

func (s *NotificationCallbackServiceTestSuite) TearDownSuite() {
	s.clientGRPCServer.Stop()
}

// 创建测试用的通知对象
func (s *NotificationCallbackServiceTestSuite) createTestNotification(bizID int64) domain.Notification {
	now := time.Now()
	return domain.Notification{
		BizID:     bizID,
		Key:       fmt.Sprintf("callback-test-key-%d-%d", now.Unix(), rand.Int()),
		Receivers: []string{"13801138000"},
		Channel:   domain.ChannelSMS,
		Template: domain.Template{
			ID:        100,
			VersionID: 1,
			Params:    map[string]string{"code": "123456"},
		},
		ScheduledSTime: now,
		ScheduledETime: now.Add(-1 * time.Hour),
		Status:         domain.SendStatusPending,
	}
}

func (s *NotificationCallbackServiceTestSuite) newService(ctrl *gomock.Controller) (*callbackioc.Service, *configmocks.MockBusinessConfigService) {
	cfg := configmocks.NewMockBusinessConfigService(ctrl)
	return callbackioc.Init(cfg), cfg
}

// TestSendCallback 测试SendCallback方法
func (s *NotificationCallbackServiceTestSuite) TestSendCallback() {
	// 测试用例
	testCases := []struct {
		name          string
		setupMock     func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) ([]domain.CallbackLog, int64)
		startTime     int64
		batchSize     int64
		errAssertFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, logs []domain.CallbackLog, app *callbackioc.Service)
	}{
		{
			name: "成功发送回调-回调成功",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) ([]domain.CallbackLog, int64) {
				bizID := int64(1001)
				// 设置业务配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID: bizID,
					CallbackConfig: &domain.CallbackConfig{
						ServiceName: callbackServerServiceName,
						RetryPolicy: &retry.Config{
							Type: "fixed",
							FixedInterval: &retry.FixedIntervalConfig{
								MaxRetries: 3,
								Interval:   1000,
							},
						},
					},
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				// 创建通知记录
				notification := s.createTestNotification(bizID)

				// 使用应用服务创建通知和回调日志
				result, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
					BizID:     notification.BizID,
					Key:       notification.Key,
					Receivers: notification.Receivers,
					Channel:   notification.Channel,
					Template: domain.Template{
						ID:        notification.Template.ID,
						VersionID: notification.Template.VersionID,
						Params:    notification.Template.Params,
					},
					Status:         notification.Status,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				})
				assert.NoError(t, err)

				// 将通知标记为发送成功，这会将回调日志状态从INIT改为PENDING
				err = app.NotificationRepo.MarkSuccess(context.Background(), result)
				assert.NoError(t, err)

				// 查询回调日志
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{result.ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, logs, "回调日志创建失败，未找到记录") {
					log.Printf("notificationID = %d, callbackLogID = %d, log.Notification.ID = %d\n", result.ID, logs[0].ID, logs[0].Notification.ID)
					assert.Equal(t, domain.CallbackLogStatusPending, logs[0].Status, "回调日志状态未正确设置为PENDING")
				}

				return logs, time.Now().Add(time.Second).UnixMilli()
			},
			startTime:     0, // 这个值会被setupMock中返回的startTime替换
			batchSize:     10,
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, logs []domain.CallbackLog, app *callbackioc.Service) {
				assert.False(t, len(logs) == 0, "回调日志列表为空，无法进行验证")
				// 验证回调日志状态更新为成功
				updatedLogs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{logs[0].Notification.ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, updatedLogs, "更新后未找到回调日志") {
					assert.Equal(t, domain.CallbackLogStatusSuccess, updatedLogs[0].Status)
				}
			},
		},
		{
			name: "成功发送回调-无回调日志",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) ([]domain.CallbackLog, int64) {
				startTime := time.Now().Add(-24 * time.Hour).UnixMilli() // 使用24小时前的时间
				// 不创建任何回调日志
				return []domain.CallbackLog{}, startTime
			},
			startTime:     0, // 这个值会被setupMock中返回的startTime替换
			batchSize:     10,
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, logs []domain.CallbackLog, app *callbackioc.Service) {
				// 无需验证，因为没有回调日志
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建服务和设置mock
			svc, mockCfg := s.newService(ctrl)

			// 设置mock和准备数据
			logs, startTime := tc.setupMock(t, mockCfg, svc)

			// 执行测试
			err := svc.Svc.SendCallback(context.Background(), startTime, tc.batchSize)

			// 验证结果
			tc.errAssertFunc(t, err)
			if err != nil {
				return
			}
			tc.after(t, logs, svc)
		})
	}
}

// TestSendCallbackByNotification 测试SendCallbackByNotification方法
func (s *NotificationCallbackServiceTestSuite) TestSendCallbackByNotification() {
	// 测试用例
	testCases := []struct {
		name          string
		setupMock     func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) domain.Notification
		errAssertFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, notification domain.Notification, app *callbackioc.Service)
	}{
		{
			name: "成功发送单个通知回调",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) domain.Notification {
				bizID := int64(1002)

				// 设置业务配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID: bizID,
					CallbackConfig: &domain.CallbackConfig{
						ServiceName: callbackServerServiceName,
						RetryPolicy: &retry.Config{
							Type: "exponential",
							ExponentialBackoff: &retry.ExponentialBackoffConfig{
								InitialInterval: 1000 * time.Millisecond,
								MaxInterval:     10000 * time.Millisecond,
								MaxRetries:      3,
							},
						},
					},
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				// 创建通知记录和回调日志
				notification := s.createTestNotification(bizID)

				// 使用应用服务创建通知和回调日志
				result, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
					BizID:     notification.BizID,
					Key:       notification.Key,
					Receivers: notification.Receivers,
					Channel:   notification.Channel,
					Template: domain.Template{
						ID:        notification.Template.ID,
						VersionID: notification.Template.VersionID,
						Params:    notification.Template.Params,
					},
					Status:         notification.Status,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				})
				assert.NoError(t, err)

				// 将通知标记为发送成功，这会将回调日志状态从INIT改为PENDING
				err = app.NotificationRepo.MarkSuccess(context.Background(), result)
				assert.NoError(t, err)

				// 查询回调日志确认状态
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{result.ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, logs) {
					assert.Equal(t, domain.CallbackLogStatusPending, logs[0].Status, "回调日志状态未正确设置为PENDING")
				}

				// 获取完整的通知对象
				return domain.Notification{
					ID:             result.ID,
					BizID:          result.BizID,
					Key:            result.Key,
					Receivers:      notification.Receivers,
					Channel:        notification.Channel,
					Template:       notification.Template,
					Status:         domain.SendStatusSucceeded, // 状态已更新为成功
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				}
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notification domain.Notification, app *callbackioc.Service) {
				// 验证回调日志状态更新为成功
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, logs) {
					assert.Equal(t, domain.CallbackLogStatusSuccess, logs[0].Status)
				}
			},
		},
		{
			name: "找不到回调记录",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) domain.Notification {
				bizID := int64(1002)

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				// 创建通知记录，但不创建回调日志
				notification := s.createTestNotification(bizID)

				// 使用应用服务创建通知，但不创建回调日志
				result, err := app.NotificationRepo.Create(context.Background(), domain.Notification{
					BizID:     notification.BizID,
					Key:       notification.Key,
					Receivers: notification.Receivers,
					Channel:   notification.Channel,
					Template: domain.Template{
						ID:        notification.Template.ID,
						VersionID: notification.Template.VersionID,
						Params:    notification.Template.Params,
					},
					Status:         notification.Status,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				})
				assert.NoError(t, err)

				// 获取完整的通知对象
				return domain.Notification{
					ID:             result.ID,
					BizID:          result.BizID,
					Key:            result.Key,
					Receivers:      notification.Receivers,
					Channel:        notification.Channel,
					Template:       notification.Template,
					Status:         notification.Status,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				}
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notification domain.Notification, app *callbackioc.Service) {
				// 确认数据库中没有对应的回调记录
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
				assert.NoError(t, err)
				assert.Empty(t, logs)
			},
		},
		{
			name: "回调配置不存在",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) domain.Notification {
				bizID := int64(1102)

				// 设置业务配置但不包含回调配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID:             bizID,
					CallbackConfig: nil, // 没有回调配置
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				// 创建通知记录和回调日志
				notification := s.createTestNotification(bizID)

				// 使用应用服务创建通知和回调日志
				result, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
					BizID:     notification.BizID,
					Key:       notification.Key,
					Receivers: notification.Receivers,
					Channel:   notification.Channel,
					Template: domain.Template{
						ID:        notification.Template.ID,
						VersionID: notification.Template.VersionID,
						Params:    notification.Template.Params,
					},
					Status:         notification.Status,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				})
				assert.NoError(t, err)

				// 将通知标记为发送成功，这会将回调日志状态从INIT改为PENDING
				err = app.NotificationRepo.MarkSuccess(context.Background(), result)
				assert.NoError(t, err)

				// 获取完整的通知对象
				return domain.Notification{
					ID:             result.ID,
					BizID:          result.BizID,
					Key:            result.Key,
					Receivers:      notification.Receivers,
					Channel:        notification.Channel,
					Template:       notification.Template,
					Status:         domain.SendStatusSucceeded,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				}
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notification domain.Notification, app *callbackioc.Service) {
				// 回调状态应该保持不变
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, logs) {
					assert.Equal(t, domain.CallbackLogStatusPending, logs[0].Status)
				}
			},
		},
		{
			name: "获取业务配置失败",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) domain.Notification {
				bizID := int64(1202)

				// 模拟获取业务配置失败
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{}, fmt.Errorf("业务配置获取失败")).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				// 创建通知记录和回调日志
				notification := s.createTestNotification(bizID)
				notification.Status = domain.SendStatusPrepare

				// 使用应用服务创建通知和回调日志
				result, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
					BizID:     notification.BizID,
					Key:       notification.Key,
					Receivers: notification.Receivers,
					Channel:   notification.Channel,
					Template: domain.Template{
						ID:        notification.Template.ID,
						VersionID: notification.Template.VersionID,
						Params:    notification.Template.Params,
					},
					Status:         domain.SendStatusPrepare,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				})
				assert.NoError(t, err)

				// 将通知标记为发送成功，这会将回调日志状态从INIT改为PENDING
				err = app.NotificationRepo.MarkSuccess(context.Background(), result)
				assert.NoError(t, err)

				// 获取完整的通知对象
				return domain.Notification{
					ID:             result.ID,
					BizID:          result.BizID,
					Key:            result.Key,
					Receivers:      notification.Receivers,
					Channel:        notification.Channel,
					Template:       notification.Template,
					Status:         domain.SendStatusSucceeded,
					ScheduledSTime: notification.ScheduledSTime,
					ScheduledETime: notification.ScheduledETime,
				}
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notification domain.Notification, app *callbackioc.Service) {
				// 回调状态应该保持不变
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, logs) {
					assert.Equal(t, domain.CallbackLogStatusPending, logs[0].Status)
				}
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建服务
			svc, mockCfg := s.newService(ctrl)

			// 设置mock和准备数据
			notification := tc.setupMock(t, mockCfg, svc)

			// 执行测试
			err := svc.Svc.SendCallbackByNotification(context.Background(), notification)

			// 验证结果
			tc.errAssertFunc(t, err)
			if err != nil {
				return
			}
			tc.after(t, notification, svc)
		})
	}
}

// TestSendCallbackByNotifications 测试SendCallbackByNotifications方法
func (s *NotificationCallbackServiceTestSuite) TestSendCallbackByNotifications() {
	// 测试用例
	testCases := []struct {
		name          string
		setupMock     func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) []domain.Notification
		errAssertFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, notifications []domain.Notification, app *callbackioc.Service)
	}{
		{
			name: "成功发送多个通知回调-全部有回调记录",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) []domain.Notification {
				bizID := int64(1003)

				// 设置业务配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID: bizID,
					CallbackConfig: &domain.CallbackConfig{
						ServiceName: callbackServerServiceName,
						RetryPolicy: &retry.Config{
							Type: "exponential",
							ExponentialBackoff: &retry.ExponentialBackoffConfig{
								InitialInterval: 1000 * time.Millisecond,
								MaxInterval:     10000 * time.Millisecond,
								MaxRetries:      3,
							},
						},
					},
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				var notifications []domain.Notification

				// 创建多个通知和回调记录
				for i := 0; i < 2; i++ {
					notification := s.createTestNotification(bizID)
					notification.Key = fmt.Sprintf("multi-test-key-%d", i+1)

					// 使用应用服务创建通知和回调日志
					result, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
						BizID:     notification.BizID,
						Key:       notification.Key,
						Receivers: notification.Receivers,
						Channel:   notification.Channel,
						Template: domain.Template{
							ID:        notification.Template.ID,
							VersionID: notification.Template.VersionID,
							Params:    notification.Template.Params,
						},
						Status:         notification.Status,
						ScheduledSTime: notification.ScheduledSTime,
						ScheduledETime: notification.ScheduledETime,
					})
					assert.NoError(t, err)

					// 将通知标记为发送成功，这会将回调日志状态从INIT改为PENDING
					err = app.NotificationRepo.MarkSuccess(context.Background(), result)
					assert.NoError(t, err)

					// 添加到通知列表
					notifications = append(notifications, domain.Notification{
						ID:             result.ID,
						BizID:          result.BizID,
						Key:            result.Key,
						Receivers:      notification.Receivers,
						Channel:        notification.Channel,
						Template:       notification.Template,
						Status:         domain.SendStatusSucceeded,
						ScheduledSTime: notification.ScheduledSTime,
						ScheduledETime: notification.ScheduledETime,
					})
				}

				return notifications
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notifications []domain.Notification, app *callbackioc.Service) {
				// 验证回调日志状态更新为成功
				for _, notification := range notifications {
					logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
					assert.NoError(t, err)
					if assert.NotEmpty(t, logs) {
						assert.Equal(t, domain.CallbackLogStatusSuccess, logs[0].Status)
					}
				}
			},
		},
		{
			name: "部分通知有回调记录",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) []domain.Notification {
				bizID := int64(1003)

				// 设置业务配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID: bizID,
					CallbackConfig: &domain.CallbackConfig{
						ServiceName: callbackServerServiceName,
						RetryPolicy: &retry.Config{
							Type: "exponential",
							ExponentialBackoff: &retry.ExponentialBackoffConfig{
								InitialInterval: 1000 * time.Millisecond,
								MaxInterval:     10000 * time.Millisecond,
								MaxRetries:      3,
							},
						},
					},
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				var notifications []domain.Notification

				// 第一个通知有回调记录
				notification1 := s.createTestNotification(bizID)
				notification1.Key = "partial-test-key-1"

				result1, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
					BizID:     notification1.BizID,
					Key:       notification1.Key,
					Receivers: notification1.Receivers,
					Channel:   notification1.Channel,
					Template: domain.Template{
						ID:        notification1.Template.ID,
						VersionID: notification1.Template.VersionID,
						Params:    notification1.Template.Params,
					},
					Status:         notification1.Status,
					ScheduledSTime: notification1.ScheduledSTime,
					ScheduledETime: notification1.ScheduledETime,
				})
				assert.NoError(t, err)

				// 将第一个通知标记为发送成功
				err = app.NotificationRepo.MarkSuccess(context.Background(), result1)
				assert.NoError(t, err)

				notifications = append(notifications, domain.Notification{
					ID:             result1.ID,
					BizID:          result1.BizID,
					Key:            result1.Key,
					Receivers:      result1.Receivers,
					Channel:        result1.Channel,
					Template:       result1.Template,
					Status:         domain.SendStatusSucceeded,
					ScheduledSTime: result1.ScheduledSTime,
					ScheduledETime: result1.ScheduledETime,
				})

				// 第二个通知没有回调记录
				notification2 := s.createTestNotification(bizID)
				notification2.Key = "partial-test-key-2"

				result2, err := app.NotificationRepo.Create(context.Background(), domain.Notification{
					BizID:     notification2.BizID,
					Key:       notification2.Key,
					Receivers: notification2.Receivers,
					Channel:   notification2.Channel,
					Template: domain.Template{
						ID:        notification2.Template.ID,
						VersionID: notification2.Template.VersionID,
						Params:    notification2.Template.Params,
					},
					Status:         notification2.Status,
					ScheduledSTime: notification2.ScheduledSTime,
					ScheduledETime: notification2.ScheduledETime,
				})
				assert.NoError(t, err)

				notifications = append(notifications, domain.Notification{
					ID:             result2.ID,
					BizID:          result2.BizID,
					Key:            result2.Key,
					Receivers:      result2.Receivers,
					Channel:        result2.Channel,
					Template:       result2.Template,
					Status:         result2.Status,
					ScheduledSTime: result2.ScheduledSTime,
					ScheduledETime: result2.ScheduledETime,
				})

				return notifications
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notifications []domain.Notification, app *callbackioc.Service) {
				// 验证有回调记录的通知状态更新为成功
				logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notifications[0].ID})
				assert.NoError(t, err)
				if assert.NotEmpty(t, logs) {
					assert.Equal(t, domain.CallbackLogStatusSuccess, logs[0].Status)
				}
				logs, err = app.Repo.FindByNotificationIDs(context.Background(), []uint64{notifications[1].ID})
				assert.NoError(t, err)
				assert.Empty(t, logs)
			},
		},
		{
			name: "全部通知没有回调记录",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) []domain.Notification {
				bizID := int64(1003)

				// 设置业务配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID: bizID,
					CallbackConfig: &domain.CallbackConfig{
						ServiceName: callbackServerServiceName,
						RetryPolicy: &retry.Config{
							Type: "exponential",
							ExponentialBackoff: &retry.ExponentialBackoffConfig{
								InitialInterval: 1000 * time.Millisecond,
								MaxInterval:     10000 * time.Millisecond,
								MaxRetries:      3,
							},
						},
					},
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				var notifications []domain.Notification

				// 两个通知都没有回调记录
				for i := 0; i < 2; i++ {
					notification := s.createTestNotification(bizID)
					notification.Key = fmt.Sprintf("no-callback-key-%d", i+1)

					result, err := app.NotificationRepo.Create(context.Background(), domain.Notification{
						BizID:     notification.BizID,
						Key:       notification.Key,
						Receivers: notification.Receivers,
						Channel:   notification.Channel,
						Template: domain.Template{
							ID:        notification.Template.ID,
							VersionID: notification.Template.VersionID,
							Params:    notification.Template.Params,
						},
						Status:         notification.Status,
						ScheduledSTime: notification.ScheduledSTime,
						ScheduledETime: notification.ScheduledETime,
					})
					assert.NoError(t, err)

					notifications = append(notifications, domain.Notification{
						ID:             result.ID,
						BizID:          result.BizID,
						Key:            result.Key,
						Receivers:      notification.Receivers,
						Channel:        notification.Channel,
						Template:       notification.Template,
						Status:         notification.Status,
						ScheduledSTime: notification.ScheduledSTime,
						ScheduledETime: notification.ScheduledETime,
					})
				}

				return notifications
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notifications []domain.Notification, app *callbackioc.Service) {
				// 验证仍然没有回调记录
				for _, notification := range notifications {
					logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
					assert.NoError(t, err)
					assert.Empty(t, logs)
				}
			},
		},
		{
			name: "回调配置不存在",
			setupMock: func(t *testing.T, mockCfg *configmocks.MockBusinessConfigService, app *callbackioc.Service) []domain.Notification {
				bizID := int64(1013)

				// 返回没有回调配置的业务配置
				mockCfg.EXPECT().GetByID(gomock.Any(), bizID).Return(domain.BusinessConfig{
					ID:             bizID,
					CallbackConfig: nil, // 没有回调配置
				}, nil).AnyTimes()

				// 创建配额
				err := app.QuotaRepo.CreateOrUpdate(context.Background(), domain.Quota{
					BizID:   bizID,
					Channel: domain.ChannelSMS,
					Quota:   int32(100),
				})
				assert.NoError(t, err)

				var notifications []domain.Notification

				// 创建多个通知和回调记录
				for i := 0; i < 2; i++ {
					notification := s.createTestNotification(bizID)
					notification.Key = fmt.Sprintf("no-config-key-%d", i+1)

					result, err := app.NotificationRepo.CreateWithCallbackLog(context.Background(), domain.Notification{
						BizID:     notification.BizID,
						Key:       notification.Key,
						Receivers: notification.Receivers,
						Channel:   notification.Channel,
						Template: domain.Template{
							ID:        notification.Template.ID,
							VersionID: notification.Template.VersionID,
							Params:    notification.Template.Params,
						},
						Status:         notification.Status,
						ScheduledSTime: notification.ScheduledSTime,
						ScheduledETime: notification.ScheduledETime,
					})
					assert.NoError(t, err)

					// 将通知标记为发送成功
					err = app.NotificationRepo.MarkSuccess(context.Background(), result)
					assert.NoError(t, err)

					notifications = append(notifications, domain.Notification{
						ID:             result.ID,
						BizID:          result.BizID,
						Key:            result.Key,
						Receivers:      notification.Receivers,
						Channel:        notification.Channel,
						Template:       notification.Template,
						Status:         domain.SendStatusSucceeded,
						ScheduledSTime: notification.ScheduledSTime,
						ScheduledETime: notification.ScheduledETime,
					})
				}

				return notifications
			},
			errAssertFunc: assert.NoError,
			after: func(t *testing.T, notifications []domain.Notification, app *callbackioc.Service) {
				// 回调状态应该更新为成功
				for _, notification := range notifications {
					logs, err := app.Repo.FindByNotificationIDs(context.Background(), []uint64{notification.ID})
					assert.NoError(t, err)
					if assert.NotEmpty(t, logs) {
						assert.Equal(t, domain.CallbackLogStatusPending, logs[0].Status)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// 创建服务
			svc, mockCfg := s.newService(ctrl)

			// 设置mock和准备数据
			notifications := tc.setupMock(t, mockCfg, svc)

			// 执行测试
			err := svc.Svc.SendCallbackByNotifications(context.Background(), notifications)

			// 验证结果
			tc.errAssertFunc(t, err)
			if err != nil {
				return
			}
			tc.after(t, notifications, svc)
		})
	}
}
