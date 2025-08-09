//go:build e2e

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ego-component/eetcd/registry"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/client/egrpc/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	grpc "google.golang.org/grpc"
	"gorm.io/gorm"
	clientv1 "notification-platform/api/proto/gen/client/v1"
	txnotificationv1 "notification-platform/api/proto/gen/client/v1"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/retry"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/service/config"
	configmocks "notification-platform/internal/service/config/mocks"
	"notification-platform/internal/service/notification"
	"notification-platform/internal/service/sender"
	sendermocks "notification-platform/internal/service/sender/mocks"
	"notification-platform/internal/test/integration/ioc/tx_notification"
	"notification-platform/internal/test/integration/testgrpc"
	testioc "notification-platform/internal/test/ioc"
)

type TxNotificationServiceTestSuite struct {
	suite.Suite
	db *egorm.Component
}

func (s *TxNotificationServiceTestSuite) SetupSuite() {
	s.db = testioc.InitDBAndTables()
}

func (s *TxNotificationServiceTestSuite) TearDownTest() {
	s.db.Exec("TRUNCATE TABLE `tx_notifications`")
	s.db.Exec("TRUNCATE TABLE `notifications`")
}

func (s *TxNotificationServiceTestSuite) TestPrepare() {
	testcases := []struct {
		name      string
		input     domain.Notification
		configSvc func(t *testing.T, ctrl *gomock.Controller) config.BusinessConfigService
		after     func(t *testing.T, now int64, id uint64)
	}{
		{
			name: "正常响应",
			input: domain.Notification{
				BizID:     3,
				Key:       "case_01",
				Receivers: []string{"18248842099"},
				Channel:   "SMS",
				Template: domain.Template{
					ID:        1,
					VersionID: 10,
					Params: map[string]string{
						"key": "value",
					},
				},
			},
			configSvc: func(_ *testing.T, ctrl *gomock.Controller) config.BusinessConfigService {
				mockConfigServices := configmocks.NewMockBusinessConfigService(ctrl)
				mockConfigServices.EXPECT().
					GetByID(gomock.Any(), int64(3)).
					Return(domain.BusinessConfig{
						ID: 3,
						TxnConfig: &domain.TxnConfig{
							ServiceName:  "order.notification.callback.service",
							InitialDelay: 10,
							RetryPolicy: &retry.Config{
								Type: "fixed",
								FixedInterval: &retry.FixedIntervalConfig{
									Interval:   30000,
									MaxRetries: 3,
								},
							},
						},
					}, nil)
				return mockConfigServices
			},
			after: func(t *testing.T, now int64, id uint64) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				var actual dao.TxNotification
				err := s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", 3, "case_01").First(&actual).Error
				require.NoError(s.T(), err)
				require.True(t, actual.Ctime > 0)
				require.True(t, actual.Utime > 0)
				nextCheckTime := now + 10000
				assert.Greater(t, actual.NextCheckTime, nextCheckTime)
				actual.Utime = 0
				actual.Ctime = 0
				actual.NextCheckTime = 0
				assert.True(t, actual.TxID > 0)
				actual.TxID = 0
				assert.Equal(t, dao.TxNotification{
					NotificationID: id,
					Key:            "case_01",
					BizID:          3,
					CheckCount:     1,
					Status:         domain.TxNotificationStatusPrepare.String(),
				}, actual)

				var actualNotification dao.Notification
				err = s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", 3, "case_01").First(&actualNotification).Error
				require.NoError(s.T(), err)
				require.True(t, actualNotification.Ctime > 0)
				require.True(t, actualNotification.Utime > 0)
				actualNotification.Utime = 0
				actualNotification.Ctime = 0
				actualNotification.ScheduledSTime = 0
				actualNotification.ScheduledETime = 0
				assert.Equal(t, dao.Notification{
					ID:                id,
					BizID:             3,
					Key:               "case_01",
					Status:            string(domain.SendStatusPrepare),
					Receivers:         `["18248842099"]`,
					Channel:           "SMS",
					TemplateID:        1,
					TemplateVersionID: 10,
					TemplateParams:    `{"key":"value"}`,
					Version:           1,
				}, actualNotification)
			},
		},
		{
			name: "配置没找到，下一次更新时间为0",
			input: domain.Notification{
				BizID:     22,
				Key:       "case_02",
				Receivers: []string{"18248842099"},
				Channel:   "SMS",
				Template: domain.Template{
					ID:        1,
					VersionID: 10,
					Params: map[string]string{
						"key": "value",
					},
				},
			},
			configSvc: func(_ *testing.T, ctrl *gomock.Controller) config.BusinessConfigService {
				mockConfigServices := configmocks.NewMockBusinessConfigService(ctrl)
				mockConfigServices.EXPECT().
					GetByID(gomock.Any(), int64(22)).
					Return(domain.BusinessConfig{}, gorm.ErrRecordNotFound)
				return mockConfigServices
			},
			after: func(t *testing.T, _ int64, id uint64) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				var actual dao.TxNotification
				err := s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", 22, "case_02").First(&actual).Error
				require.NoError(s.T(), err)
				require.True(t, actual.Ctime > 0)
				require.True(t, actual.Utime > 0)
				actual.Utime = 0
				actual.Ctime = 0
				assert.True(t, actual.TxID > 0)
				actual.TxID = 0
				assert.Equal(t, dao.TxNotification{
					NotificationID: id,
					BizID:          22,
					Key:            "case_02",
					CheckCount:     1,
					Status:         domain.TxNotificationStatusPrepare.String(),
				}, actual)

				var actualNotification dao.Notification
				err = s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", 22, "case_02").First(&actualNotification).Error
				require.NoError(s.T(), err)
				require.True(t, actualNotification.Ctime > 0)
				require.True(t, actualNotification.Utime > 0)
				actualNotification.Utime = 0
				actualNotification.Ctime = 0
				actualNotification.ScheduledSTime = 0
				actualNotification.ScheduledETime = 0
				assert.Equal(t, dao.Notification{
					ID:                id,
					BizID:             22,
					Key:               "case_02",
					Status:            string(domain.SendStatusPrepare),
					Receivers:         `["18248842099"]`,
					Channel:           "SMS",
					TemplateID:        1,
					TemplateVersionID: 10,
					TemplateParams:    `{"key":"value"}`,
					Version:           1,
				}, actualNotification)
			},
		},
	}
	for idx := range testcases {
		tc := testcases[idx]
		s.T().Run(tc.name, func(t *testing.T) {
			now := time.Now().Unix()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			configSvc := tc.configSvc(t, ctrl)
			app := tx_notification.InitTxNotificationService(configSvc, nil)
			id, err := app.Svc.Prepare(ctx, tc.input)
			require.NoError(t, err)
			tc.after(t, now, id)
		})
	}
}

func (s *TxNotificationServiceTestSuite) TestCommit() {
	testcases := []struct {
		name      string
		configSvc func(t *testing.T, ctrl *gomock.Controller) config.BusinessConfigService
		sender    func(t *testing.T, ctrl *gomock.Controller) sender.NotificationSender
		after     func(t *testing.T, bizId int64, key string)
		before    func(t *testing.T) (int64, string)
		checkErr  func(t *testing.T, err error) bool
	}{
		{
			name: "正常提交",
			configSvc: func(_ *testing.T, ctrl *gomock.Controller) config.BusinessConfigService {
				mockConfigServices := configmocks.NewMockBusinessConfigService(ctrl)
				return mockConfigServices
			},
			sender: func(t *testing.T, ctrl *gomock.Controller) sender.NotificationSender {
				mockSender := sendermocks.NewMockNotificationSender(ctrl)
				return mockSender
			},
			before: func(t *testing.T) (int64, string) {
				t.Helper()
				now := time.Now().UnixMilli()
				txn := dao.TxNotification{
					TxID:           201,
					Key:            "case_02",
					NotificationID: 10123,
					BizID:          3,
					Status:         domain.TxNotificationStatusPrepare.String(),
					CheckCount:     0,
					NextCheckTime:  now + 10000,
					Ctime:          now,
					Utime:          now,
				}
				err := s.db.Create(&txn).Error
				require.NoError(t, err)
				noti := dao.Notification{
					ID:      10123,
					Key:     "case_02",
					Channel: "SMS",
					BizID:   3,
					Status:  domain.TxNotificationStatusPrepare.String(),
					Ctime:   now,
					Utime:   now,
				}
				err = s.db.Create(&noti).Error
				require.NoError(t, err)
				return txn.BizID, txn.Key
			},
			after: func(t *testing.T, bizId int64, key string) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				var actual dao.TxNotification
				err := s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", bizId, key).First(&actual).Error
				require.NoError(t, err)
				assert.Equal(t, domain.TxNotificationStatusCommit.String(), actual.Status)
				var actualNoti dao.Notification
				err = s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", bizId, key).First(&actualNoti).Error
				require.NoError(t, err)
				assert.Equal(t, string(domain.SendStatusPending), actualNoti.Status)
			},
			checkErr: func(t *testing.T, err error) bool {
				t.Helper()
				require.NoError(t, err)
				return false
			},
		},
	}

	for idx := range testcases {
		tc := testcases[idx]
		s.T().Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			bizID, key := tc.before(t)

			configSvc := tc.configSvc(t, ctrl)
			senderSvc := tc.sender(t, ctrl)
			svc := tx_notification.InitTxNotificationService(configSvc, senderSvc)
			err := svc.Svc.Commit(ctx, bizID, key)
			hasError := tc.checkErr(t, err)
			if !hasError {
				tc.after(t, bizID, key)
			}
		})
	}
}

func (s *TxNotificationServiceTestSuite) TestCancel() {
	testcases := []struct {
		name            string
		configSvc       func(t *testing.T, ctrl *gomock.Controller) config.BusinessConfigService
		notificationSvc func(t *testing.T, ctrl *gomock.Controller) notification.NotificationService
		after           func(t *testing.T, bizId int64, key string)
		before          func(t *testing.T) (int64, string)
		checkErr        func(t *testing.T, err error) bool
	}{
		{
			name: "正常取消",
			configSvc: func(_ *testing.T, ctrl *gomock.Controller) config.BusinessConfigService {
				mockConfigServices := configmocks.NewMockBusinessConfigService(ctrl)
				return mockConfigServices
			},
			before: func(t *testing.T) (int64, string) {
				t.Helper()
				now := time.Now().UnixMilli()
				txn := dao.TxNotification{
					TxID:           202,
					NotificationID: 10124,
					BizID:          5,
					Key:            "question_02",
					Status:         domain.TxNotificationStatusPrepare.String(),
					CheckCount:     0,
					NextCheckTime:  now + 10000,
					Ctime:          now,
					Utime:          now,
				}
				err := s.db.Create(&txn).Error
				require.NoError(t, err)
				noti := dao.Notification{
					ID:      10124,
					Channel: "SMS",
					BizID:   5,
					Key:     "question_02",
					Status:  domain.TxNotificationStatusPrepare.String(),
					Ctime:   now,
					Utime:   now,
				}
				err = s.db.Create(&noti).Error
				require.NoError(t, err)
				return txn.BizID, txn.Key
			},
			after: func(t *testing.T, bizId int64, key string) {
				t.Helper()
				ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
				defer cancel()
				var actual dao.TxNotification
				err := s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", bizId, key).First(&actual).Error
				require.NoError(t, err)
				assert.Equal(t, domain.TxNotificationStatusCancel.String(), actual.Status)

				var actualNoti dao.Notification
				err = s.db.WithContext(ctx).Where("biz_id = ? AND `key` = ?", bizId, key).First(&actualNoti).Error
				require.NoError(t, err)
				assert.Equal(t, string(domain.SendStatusCanceled), actualNoti.Status)
			},
			checkErr: func(t *testing.T, err error) bool {
				t.Helper()
				require.NoError(t, err)
				return false
			},
		},
	}

	for idx := range testcases {
		tc := testcases[idx]
		s.T().Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			bizID, key := tc.before(t)

			configSvc := tc.configSvc(t, ctrl)
			svc := tx_notification.InitTxNotificationService(configSvc, nil)
			err := svc.Svc.Cancel(ctx, bizID, key)
			hasError := tc.checkErr(t, err)
			if !hasError {
				tc.after(t, bizID, key)
			}
		})
	}
}

func (s *TxNotificationServiceTestSuite) TestCheckBackTask() {
	t := s.T()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	configSvc := configmocks.NewMockBusinessConfigService(ctrl)
	mockConfigMap := s.mockConfigMap()

	configSvc.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, ids []int64) (map[int64]domain.BusinessConfig, error) {
		res := make(map[int64]domain.BusinessConfig, len(ids))
		for _, id := range ids {
			v, ok := mockConfigMap[id]
			if ok {
				res[id] = v
			}
		}
		return res, nil
	}).AnyTimes()

	// 测试逻辑比较，简单直接往数据库里插数据模拟各种情况

	now := time.Now().UnixMilli()
	// 事务1的会取出来，并且成功提交
	tx1 := s.MockDaoTxn(1, 1, now-(time.Second*11).Milliseconds())
	tx1.Status = domain.TxNotificationStatusPrepare.String()
	tx1.BizID = 1
	// 事务2取不出来，因为已经提交
	tx2 := s.MockDaoTxn(2, 2, 0)
	tx2.Status = domain.TxNotificationStatusCommit.String()
	tx2.BizID = 2
	// 事务3取不出来, 因为已经取消
	tx3 := s.MockDaoTxn(3, 3, 0)
	tx3.Status = domain.TxNotificationStatusCancel.String()
	tx3.BizID = 3
	// 事务4取不出来, 因为已经失败
	tx4 := s.MockDaoTxn(4, 4, 0)
	tx4.Status = domain.TxNotificationStatusFail.String()
	tx4.BizID = 4
	// 事务5取不出来，因为还没有到检查时间
	tx5 := s.MockDaoTxn(5, 5, now+(time.Second*30).Milliseconds())
	tx5.Status = domain.TxNotificationStatusPrepare.String()
	tx5.BizID = 5
	// 事务11会取出来，但是他只有最后一次回查机会了，发送会成功
	tx11 := s.MockDaoTxn(11, 11, now-(time.Second*11).Milliseconds())
	tx11.Status = domain.TxNotificationStatusPrepare.String()
	tx11.BizID = 1
	tx11.CheckCount = 2
	// 事务会取出来，但是他只有最后一次回查机会了，发送失败
	tx22 := s.MockDaoTxn(22, 22, now-(time.Second*11).Milliseconds())
	tx22.Status = domain.TxNotificationStatusPrepare.String()
	tx22.BizID = 1
	tx22.CheckCount = 3
	// 事务23会取出来，但是他还有好几次次回查机会了，发送失败 可以重试
	tx23 := s.MockDaoTxn(23, 23, now-(time.Second*11).Milliseconds())
	tx23.Status = domain.TxNotificationStatusPrepare.String()
	tx23.BizID = 1
	tx23.CheckCount = 1
	// 事务44h会取出来，回查一次然后取消
	tx44 := s.MockDaoTxn(44, 44, now-(time.Second*11).Milliseconds())
	tx44.Status = domain.TxNotificationStatusPrepare.String()
	tx44.BizID = 2
	tx44.CheckCount = 1
	txns := []dao.TxNotification{
		tx1,
		tx2,
		tx3,
		tx4,
		tx5,
		tx11,
		tx22,
		tx23,
		tx44,
	}
	err := s.db.WithContext(context.Background()).Create(txns).Error
	require.NoError(t, err)
	s.mockNotifications()

	txSvc := tx_notification.InitTxNotificationService(configSvc, nil)

	// 初始化注册中心
	etcdClient := testioc.InitEtcdClient()
	reg := registry.Load("").Build(registry.WithClientEtcd(etcdClient))
	go func() {
		server := testgrpc.NewServer[txnotificationv1.TransactionCheckServiceServer]("order.notification.callback.service", reg, &MockGrpcServer{}, func(s grpc.ServiceRegistrar, srv clientv1.TransactionCheckServiceServer) {
			txnotificationv1.RegisterTransactionCheckServiceServer(s, srv)
		})
		err = server.Start("127.0.0.1:30001")
		if err != nil {
			require.NoError(t, err)
		}
	}()
	// 等待启动完成
	time.Sleep(1 * time.Second)
	resolver.Register("etcd", reg)
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Second)
	defer cancel()

	txSvc.Task.Start(ctx)
	<-ctx.Done()
	tx1.CheckCount = 1
	tx1.NextCheckTime = 0
	tx1.Status = domain.TxNotificationStatusCommit.String()
	txns[0] = tx1
	tx11.NextCheckTime = 0
	tx11.CheckCount = 3
	tx11.Status = domain.TxNotificationStatusCommit.String()
	txns[5] = tx11
	tx22.NextCheckTime = 0
	tx22.CheckCount = 4
	tx22.Status = domain.TxNotificationStatusFail.String()
	txns[6] = tx22
	tx23.NextCheckTime = now + 30*1000
	fmt.Println("xxxxxx", tx23.NextCheckTime)
	tx23.CheckCount = 2
	tx23.Status = domain.TxNotificationStatusPrepare.String()
	txns[7] = tx23
	tx44.NextCheckTime = 0
	tx44.CheckCount = 2
	tx44.Status = domain.TxNotificationStatusCancel.String()
	txns[8] = tx44
	var notifications []dao.TxNotification
	nctx, ncancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ncancel()
	err = s.db.WithContext(nctx).Model(&dao.TxNotification{}).Find(&notifications).Error
	require.NoError(t, err)
	txnMap := make(map[int64]dao.TxNotification, len(notifications))
	for _, n := range notifications {
		txnMap[n.TxID] = n
	}
	for _, n := range txns {
		txn, ok := txnMap[n.TxID]
		require.True(t, ok)
		s.assertTxNotification(n, txn)
	}
	var list []dao.Notification
	notiMap := make(map[uint64]dao.Notification)
	s.db.WithContext(nctx).Model(&dao.Notification{}).Find(&list)
	for i := range list {
		notiMap[list[i].ID] = list[i]
	}

	for key, wantNotification := range s.wantNotifications() {
		actualNotification, ok := notiMap[key]
		require.True(t, ok)
		assert.Equal(t, wantNotification, actualNotification.Status)
	}
}

func (s *TxNotificationServiceTestSuite) mockConfigMap() map[int64]domain.BusinessConfig {
	return map[int64]domain.BusinessConfig{
		1: {
			TxnConfig: &domain.TxnConfig{
				ServiceName:  "order.notification.callback.service",
				InitialDelay: 10,
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   30 * time.Second,
						MaxRetries: 3,
					},
				},
			},
		},
		2: {
			TxnConfig: &domain.TxnConfig{
				ServiceName:  "order.notification.callback.service",
				InitialDelay: 10,
				RetryPolicy: &retry.Config{
					Type: "fixed",
					FixedInterval: &retry.FixedIntervalConfig{
						Interval:   10 * time.Second,
						MaxRetries: 2,
					},
				},
			},
		},
	}
}

func TestTxNotificationServiceSuite(t *testing.T) {
	t.Skip()
	t.Parallel()
	suite.Run(t, new(TxNotificationServiceTestSuite))
}

func (s *TxNotificationServiceTestSuite) mockNotifications() {
	list := []dao.Notification{
		mockNotification(1, string(domain.SendStatusPrepare)),
		mockNotification(2, string(domain.SendStatusPending)),
		mockNotification(3, string(domain.SendStatusCanceled)),
		mockNotification(4, string(domain.SendStatusCanceled)),
		mockNotification(5, string(domain.SendStatusPrepare)),
		mockNotification(11, string(domain.SendStatusPrepare)),
		mockNotification(22, string(domain.SendStatusPrepare)),
		mockNotification(23, string(domain.SendStatusPrepare)),
		mockNotification(44, string(domain.SendStatusPrepare)),
	}
	err := s.db.WithContext(context.Background()).Create(&list).Error
	require.NoError(s.T(), err)
}

func (s *TxNotificationServiceTestSuite) wantNotifications() map[uint64]string {
	return map[uint64]string{
		1:  string(domain.SendStatusPending),
		2:  string(domain.SendStatusPending),
		3:  string(domain.SendStatusCanceled),
		4:  string(domain.SendStatusCanceled),
		5:  string(domain.SendStatusPrepare),
		11: string(domain.SendStatusPending),
		22: string(domain.SendStatusFailed),
		23: string(domain.SendStatusPrepare),
		44: string(domain.SendStatusFailed),
	}
}

func mockNotification(id uint64, status string) dao.Notification {
	return dao.Notification{
		ID:      id,
		Channel: "SMS",
		BizID:   int64(id),
		Key:     fmt.Sprintf("%d", id),
		Status:  status,
	}
}

func (s *TxNotificationServiceTestSuite) MockDaoTxn(txid int64, nid uint64, nextTime int64) dao.TxNotification {
	now := time.Now().UnixMilli()
	return dao.TxNotification{
		TxID:           txid,
		Key:            fmt.Sprintf("%d", nid),
		NotificationID: nid,
		NextCheckTime:  nextTime,
		Ctime:          now,
		Utime:          now,
	}
}

func (s *TxNotificationServiceTestSuite) assertTxNotification(wantTxn, actualTxn dao.TxNotification) {
	assert.Equal(s.T(), wantTxn.TxID, actualTxn.TxID)
	assert.Equal(s.T(), wantTxn.Key, actualTxn.Key)
	assert.Equal(s.T(), wantTxn.NotificationID, actualTxn.NotificationID)
	assert.Equal(s.T(), wantTxn.BizID, actualTxn.BizID)
	assert.Equal(s.T(), wantTxn.Status, actualTxn.Status)
	if actualTxn.NextCheckTime == 0 {
		assert.Equal(s.T(), wantTxn.NextCheckTime, actualTxn.NextCheckTime)
	} else {
		require.LessOrEqual(s.T(), wantTxn.NextCheckTime, actualTxn.NextCheckTime)
	}
	require.True(s.T(), actualTxn.Ctime > 0)
	require.True(s.T(), actualTxn.Utime > 0)
}

type MockGrpcServer struct {
	clientv1.UnsafeTransactionCheckServiceServer
}

func (m *MockGrpcServer) Check(_ context.Context, request *clientv1.TransactionCheckServiceCheckRequest) (*clientv1.TransactionCheckServiceCheckResponse, error) {
	if strings.Contains(request.GetKey(), "1") {
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_COMMITTED,
		}, nil
	}
	if strings.Contains(request.GetKey(), "2") {
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_UNKNOWN,
		}, nil
	}
	if strings.Contains(request.GetKey(), "4") {
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_CANCEL,
		}, nil
	}
	return &clientv1.TransactionCheckServiceCheckResponse{
		Status: clientv1.TransactionCheckServiceCheckResponse_UNKNOWN,
	}, nil
}
