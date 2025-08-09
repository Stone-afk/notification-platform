//go:build e2e

package integration

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"notification-platform/internal/repository/cache"

	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	"notification-platform/internal/repository"
	testioc "notification-platform/internal/test/ioc"

	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	notificationsvc "notification-platform/internal/service/notification"
	notificationioc "notification-platform/internal/test/integration/ioc/notification"
)

func TestNotificationServiceSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceTestSuite))
}

type NotificationServiceTestSuite struct {
	suite.Suite
	db              *egorm.Component
	svc             notificationsvc.NotificationService
	repo            repository.NotificationRepository
	quotaRepo       repository.QuotaRepository
	callbackLogRepo repository.CallbackLogRepository
	quotaCache      cache.QuotaCache
}

func (s *NotificationServiceTestSuite) SetupSuite() {
	s.db = testioc.InitDBAndTables()
	svc := notificationioc.Init()
	s.svc, s.repo, s.quotaRepo, s.callbackLogRepo, s.quotaCache = svc.Svc, svc.Repo, svc.QuotaRepo, svc.CallbackLogRepo, svc.QuotaCache
}

func (s *NotificationServiceTestSuite) TearDownTest() {
	// 每个测试后清空表数据
	s.db.Exec("TRUNCATE TABLE `notifications`")
	s.db.Exec("TRUNCATE TABLE `quotas`")
	s.db.Exec("TRUNCATE TABLE `callback_logs`")
}

// 创建测试用的通知对象
func (s *NotificationServiceTestSuite) createTestNotification(bizID int64) domain.Notification {
	now := time.Now()
	return domain.Notification{
		BizID:     bizID,
		Key:       fmt.Sprintf("test-key-%d-%d", now.Unix(), rand.Int()),
		Receivers: []string{"13800138000"},
		Channel:   domain.ChannelSMS,
		Template: domain.Template{
			ID:        100,
			VersionID: 1,
			Params:    map[string]string{"code": "123456"},
		},
		ScheduledSTime: now,
		ScheduledETime: now.Add(1 * time.Hour),
		Status:         domain.SendStatusPending,
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryCreate() {
	t := s.T()

	bizID := int64(2)
	tests := []struct {
		name          string
		before        func(t *testing.T, notification domain.Notification)
		after         func(t *testing.T, expected, actual domain.Notification)
		notification  domain.Notification
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name: "创建成功",
			before: func(t *testing.T, notification domain.Notification) {
				t.Helper()
				_ = s.createTestQuota(t, notification)
			},
			notification: func() domain.Notification {
				return s.createTestNotification(1)
			}(),
			assertErrFunc: assert.NoError,
			after: func(t *testing.T, expected, actual domain.Notification) {
				t.Helper()
				s.assertNotification(t, expected, actual)
			},
		},
		{
			name: "BizID和Key组成的唯一索引冲突",
			before: func(t *testing.T, notification domain.Notification) {
				t.Helper()
				t.Helper()

				_ = s.createTestQuota(t, notification)

				_, err := s.repo.Create(t.Context(), notification)
				assert.NoError(t, err)
			},
			notification: func() domain.Notification {
				return s.createTestNotification(bizID)
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, errs.ErrNotificationDuplicate)
			},
			after: func(t *testing.T, expected, actual domain.Notification) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.before(t, tt.notification)

			created, err := s.repo.Create(t.Context(), tt.notification)
			tt.assertErrFunc(t, err)

			if err != nil {
				return
			}

			tt.after(t, tt.notification, created)
		})
	}
}

func (s *NotificationServiceTestSuite) assertNotification(t *testing.T, expected domain.Notification, actual domain.Notification) {
	assert.NotZero(t, actual.ID)
	assert.Equal(t, expected.BizID, actual.BizID)
	assert.Equal(t, expected.Key, actual.Key)
	assert.Equal(t, expected.Receivers, actual.Receivers)
	assert.Equal(t, expected.Channel, actual.Channel)
	assert.Equal(t, expected.Template, actual.Template)
	assert.Equal(t, expected.ScheduledSTime.UnixMilli(), actual.ScheduledSTime.UnixMilli())
	assert.Equal(t, expected.ScheduledETime.UnixMilli(), actual.ScheduledETime.UnixMilli())
	assert.Equal(t, expected.Status, actual.Status)
}

func (s *NotificationServiceTestSuite) TestRepositoryBatchCreate() {
	t := s.T()
	bizID := int64(4)
	s.createTestQuota(t, domain.Notification{
		BizID:   bizID,
		Channel: domain.ChannelSMS,
	})

	tests := []struct {
		name          string
		notifications []domain.Notification
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name: "创建成功",
			notifications: []domain.Notification{
				s.createTestNotification(bizID),
				s.createTestNotification(bizID),
				s.createTestNotification(bizID),
			},
			assertErrFunc: assert.NoError,
		},
		{
			name: "BizID和Key组成的唯一索引冲突",
			notifications: func() []domain.Notification {
				ns := make([]domain.Notification, 2)
				ns[0] = s.createTestNotification(bizID)
				ns[1] = ns[0]
				return ns
			}(),
			assertErrFunc: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, errs.ErrNotificationDuplicate)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := s.repo.BatchCreate(t.Context(), tt.notifications)
			tt.assertErrFunc(t, err)
			if err != nil {
				return
			}
			for i := range tt.notifications {
				s.assertNotification(t, tt.notifications[i], created[i])
			}
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryCASStatus() {
	t := s.T()

	bizID := int64(8)
	tests := []struct {
		name   string
		before func(t *testing.T) (uint64, int)
		after  func(t *testing.T, id uint64)

		requireFunc require.ErrorAssertionFunc
	}{
		{
			name: "id存在，更新成功",
			before: func(t *testing.T) (uint64, int) {
				t.Helper()

				notification := s.createTestNotification(bizID)

				_ = s.createTestQuota(t, notification)

				created, err := s.repo.Create(t.Context(), notification)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusPending, created.Status)
				return created.ID, created.Version
			},
			after: func(t *testing.T, id uint64) {
				t.Helper()
				// 验证状态已更新
				updated, err := s.repo.GetByID(t.Context(), id)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusSucceeded, updated.Status)
				assert.Equal(t, 2, updated.Version) // 版本号应该加1

				find, err := s.quotaRepo.Find(t.Context(), bizID, updated.Channel)
				require.NoError(t, err)
				assert.Equal(t, int32(99), find.Quota)
			},
			requireFunc: require.NoError,
		},
		{
			name: "id不存在",
			before: func(t *testing.T) (uint64, int) {
				t.Helper()
				return 999999, 1
			},
			after: func(t *testing.T, id uint64) {
				t.Helper()
			},
			requireFunc: require.Error, // 应该返回错误
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, version := tt.before(t)

			err := s.repo.CASStatus(t.Context(), domain.Notification{
				ID:      id,
				Status:  domain.SendStatusSucceeded,
				Version: version,
			})
			tt.requireFunc(t, err)

			tt.after(t, id)
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryUpdateStatus() {
	t := s.T()

	bizID := int64(8)
	tests := []struct {
		name   string
		before func(t *testing.T) (uint64, int) // 返回ID和Version
		after  func(t *testing.T, id uint64)

		requireFunc require.ErrorAssertionFunc
	}{
		{
			name: "id存在更新成功",
			before: func(t *testing.T) (uint64, int) {
				t.Helper()
				created, err := s.repo.Create(t.Context(), s.createTestNotification(bizID))
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusPending, created.Status)

				return created.ID, created.Version
			},
			after: func(t *testing.T, id uint64) {
				t.Helper()
				// 验证状态已更新
				updated, err := s.repo.GetByID(t.Context(), id)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusSucceeded, updated.Status)
				assert.Equal(t, 2, updated.Version) // 版本号应该加1
			},
			requireFunc: require.NoError,
		},
		{
			name: "id不存在",
			before: func(t *testing.T) (uint64, int) {
				t.Helper()
				return 999999, 1
			},
			after: func(t *testing.T, id uint64) {
				t.Helper()
			},
			requireFunc: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, version := tt.before(t)

			err := s.repo.UpdateStatus(t.Context(), domain.Notification{
				ID:      id,
				Status:  domain.SendStatusSucceeded,
				Version: version,
			})
			tt.requireFunc(t, err)

			tt.after(t, id)
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryBatchUpdateStatusSucceededOrFailed() {
	t := s.T()
	ctx := t.Context()

	// 准备测试数据 - 创建多条通知记录
	bizID := int64(10)
	s.createTestQuota(t, domain.Notification{
		BizID:   bizID,
		Channel: domain.ChannelSMS,
	})
	notifications := []domain.Notification{
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
	}

	// 批量创建通知记录
	createdNotifications, err := s.repo.BatchCreate(ctx, notifications)
	require.NoError(t, err)
	require.Len(t, createdNotifications, len(notifications))

	// 记录初始版本号
	initialVersions := make(map[uint64]int)
	for _, n := range createdNotifications {
		initialVersions[n.ID] = n.Version
	}

	// 定义测试场景
	tests := []struct {
		name                   string
		succeededNotifications []domain.Notification // 要更新为成功状态的通知
		failedNotifications    []domain.Notification // 要更新为失败状态的通知
		assertFunc             func(t *testing.T, err error, initialVersions map[uint64]int)
	}{
		{
			name: "仅更新成功状态但不更新重试次数",
			succeededNotifications: []domain.Notification{
				{
					ID:      createdNotifications[0].ID,
					Version: initialVersions[createdNotifications[0].ID],
				},
				{
					ID:      createdNotifications[1].ID,
					Version: initialVersions[createdNotifications[1].ID],
				},
			},
			failedNotifications: nil,
			assertFunc: func(t *testing.T, err error, initialVersions map[uint64]int) {
				require.NoError(t, err)

				// 验证成功状态更新
				for _, id := range []uint64{createdNotifications[0].ID, createdNotifications[1].ID} {
					updated, err := s.repo.GetByID(ctx, id)
					require.NoError(t, err)
					assert.Equal(t, domain.SendStatusSucceeded, updated.Status)
					assert.Greater(t, updated.Version, initialVersions[id])
				}

				// 验证其他通知状态未变
				unchanged, err := s.repo.GetByID(ctx, createdNotifications[2].ID)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusPending, unchanged.Status)
				assert.Equal(t, initialVersions[createdNotifications[2].ID], unchanged.Version)
			},
		},
		{
			name: "更新成功状态同时更新重试次数",
			succeededNotifications: []domain.Notification{
				{
					ID:      createdNotifications[6].ID,
					Version: initialVersions[createdNotifications[6].ID],
				},
			},
			failedNotifications: nil,
			assertFunc: func(t *testing.T, err error, initialVersions map[uint64]int) {
				require.NoError(t, err)

				// 验证成功状态和重试次数更新
				updated, err := s.repo.GetByID(ctx, createdNotifications[6].ID)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusSucceeded, updated.Status)
				assert.Greater(t, updated.Version, initialVersions[createdNotifications[6].ID])
			},
		},
		{
			name:                   "仅更新失败状态但不更新重试次数",
			succeededNotifications: nil,
			failedNotifications: []domain.Notification{
				{
					ID:      createdNotifications[2].ID,
					Version: initialVersions[createdNotifications[2].ID],
				},
			},
			assertFunc: func(t *testing.T, err error, initialVersions map[uint64]int) {
				require.NoError(t, err)

				// 验证失败状态更新
				updated, err := s.repo.GetByID(ctx, createdNotifications[2].ID)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusFailed, updated.Status)
				assert.Greater(t, updated.Version, initialVersions[createdNotifications[2].ID])
			},
		},
		{
			name:                   "更新失败状态同时更新重试次数",
			succeededNotifications: nil,
			failedNotifications: []domain.Notification{
				{
					ID:      createdNotifications[3].ID,
					Version: initialVersions[createdNotifications[3].ID],
				},
			},
			assertFunc: func(t *testing.T, err error, initialVersions map[uint64]int) {
				require.NoError(t, err)

				// 验证失败状态和重试次数更新
				updated, err := s.repo.GetByID(ctx, createdNotifications[3].ID)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusFailed, updated.Status)
				assert.Greater(t, updated.Version, initialVersions[createdNotifications[3].ID])
			},
		},
		{
			name: "更新成功状态和失败状态的组合",
			succeededNotifications: []domain.Notification{
				{
					ID:      createdNotifications[4].ID,
					Version: initialVersions[createdNotifications[4].ID],
				},
			},
			failedNotifications: []domain.Notification{
				{
					ID:      createdNotifications[5].ID,
					Version: initialVersions[createdNotifications[5].ID],
				},
			},
			assertFunc: func(t *testing.T, err error, initialVersions map[uint64]int) {
				require.NoError(t, err)

				// 验证成功状态更新
				updated1, err := s.repo.GetByID(ctx, createdNotifications[4].ID)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusSucceeded, updated1.Status)
				assert.Greater(t, updated1.Version, initialVersions[createdNotifications[4].ID])

				// 验证失败状态和重试次数更新
				updated2, err := s.repo.GetByID(ctx, createdNotifications[5].ID)
				require.NoError(t, err)
				assert.Equal(t, domain.SendStatusFailed, updated2.Status)
				assert.Greater(t, updated2.Version, initialVersions[createdNotifications[5].ID])
			},
		},
	}

	// 执行测试场景
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 更新状态
			err1 := s.repo.BatchUpdateStatusSucceededOrFailed(ctx, tt.succeededNotifications, tt.failedNotifications)

			// 断言结果
			tt.assertFunc(t, err1, initialVersions)
		})
	}
}

func (s *NotificationServiceTestSuite) TestGetByKeys() {
	t := s.T()

	bizID := int64(7)
	s.createTestQuota(t, domain.Notification{
		BizID:   bizID,
		Channel: domain.ChannelSMS,
	})
	notifications, err := s.repo.BatchCreate(t.Context(), []domain.Notification{
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
	})
	require.NoError(t, err)

	tests := []struct {
		name          string
		bizID         int64
		keysFunc      func() []string
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name:  "正常流程",
			bizID: bizID,
			keysFunc: func() []string {
				// 测试获取通知列表
				return []string{notifications[1].Key, notifications[0].Key, notifications[2].Key}
			},
			assertErrFunc: assert.NoError,
		},
		{
			name:  "keys为空",
			bizID: 1001,
			keysFunc: func() []string {
				return nil
			},
			assertErrFunc: func(t assert.TestingT, err error, i ...any) bool {
				return assert.ErrorIs(t, err, errs.ErrInvalidParameter)
			},
		},
		{
			name:  "不存在的key",
			bizID: 1001,
			keysFunc: func() []string {
				return []string{"non-existent-key"}
			},
			assertErrFunc: assert.NoError,
		},
		{
			name:  "不存在的业务ID",
			bizID: 999999,
			keysFunc: func() []string {
				return []string{"test-key"}
			},
			assertErrFunc: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			found, err1 := s.svc.GetByKeys(t.Context(), tt.bizID, tt.keysFunc()...)
			tt.assertErrFunc(t, err1)

			if err1 != nil {
				return
			}

			mp := make(map[int64]domain.Notification, len(notifications))
			for i := range notifications {
				mp[notifications[i].BizID] = notifications[i]
			}

			for j := range found {
				s.assertNotification(t, mp[found[j].BizID], found[j])
			}
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryGetByID() {
	t := s.T()

	bizID := int64(9)
	// 创建测试通知
	notification := s.createTestNotification(bizID)

	_ = s.createTestQuota(t, notification)

	created, err := s.repo.Create(t.Context(), notification)
	require.NoError(t, err)

	tests := []struct {
		name          string
		id            uint64
		assertErrFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, retrieved domain.Notification)
	}{
		{
			name:          "存在ID查询成功",
			id:            created.ID,
			assertErrFunc: assert.NoError,
			after: func(t *testing.T, actual domain.Notification) {
				t.Helper()
				s.assertNotification(t, notification, actual)
			},
		},
		{
			name: "不存在ID查询失败",
			id:   999999,
			assertErrFunc: func(t assert.TestingT, err error, i ...any) bool {
				return assert.Error(t, err)
			},
			after: func(t *testing.T, retrieved domain.Notification) {
				// 不做任何验证，因为应该出错
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := s.repo.GetByID(t.Context(), tt.id)
			tt.assertErrFunc(t, err)
			if err != nil {
				return
			}

			tt.after(t, retrieved)
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryGetByKey() {
	t := s.T()

	bizID := int64(10)

	notification := s.createTestNotification(bizID)

	tests := []struct {
		name          string
		before        func(t *testing.T, notification domain.Notification)
		bizID         int64
		key           string
		assertErrFunc assert.ErrorAssertionFunc
		after         func(t *testing.T, retrieved domain.Notification)
	}{
		{
			name: "存在的bizID和key查询成功",
			before: func(t *testing.T, notification domain.Notification) {
				t.Helper()
				_ = s.createTestQuota(t, notification)

				_, err := s.repo.Create(t.Context(), notification)
				require.NoError(t, err)
			},
			bizID:         notification.BizID,
			key:           notification.Key,
			assertErrFunc: assert.NoError,
			after: func(t *testing.T, actual domain.Notification) {
				t.Helper()
				s.assertNotification(t, notification, actual)
			},
		},
		{
			name:          "不存在的key查询失败",
			before:        func(t *testing.T, notification domain.Notification) {},
			bizID:         notification.BizID,
			key:           "non-existent-key",
			assertErrFunc: assert.Error,
			after: func(t *testing.T, retrieved domain.Notification) {
				// 不做任何验证，因为应该出错
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.before(t, notification)

			retrieved, err := s.repo.GetByKey(t.Context(), tt.bizID, tt.key)
			tt.assertErrFunc(t, err)

			if err != nil {
				return
			}
			tt.after(t, retrieved)
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryBatchGetByIDs() {
	t := s.T()

	bizID := int64(11)
	s.createTestQuota(t, domain.Notification{
		BizID:   bizID,
		Channel: domain.ChannelSMS,
	})
	// 创建多个测试通知
	notifications := []domain.Notification{
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
	}
	created, err := s.repo.BatchCreate(t.Context(), notifications)
	require.NoError(t, err)
	require.Len(t, created, 3)

	tests := []struct {
		name          string
		ids           []uint64
		expectedCount int
		assertErrFunc assert.ErrorAssertionFunc
	}{
		{
			name:          "全部存在的ID查询成功",
			ids:           []uint64{created[0].ID, created[1].ID, created[2].ID},
			expectedCount: 3,
			assertErrFunc: assert.NoError,
		},
		{
			name:          "部分存在的ID查询成功",
			ids:           []uint64{created[0].ID, 999999},
			expectedCount: 1,
			assertErrFunc: assert.NoError,
		},
		{
			name:          "全部不存在的ID查询返回空map",
			ids:           []uint64{999999, 888888},
			expectedCount: 0,
			assertErrFunc: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := s.repo.BatchGetByIDs(t.Context(), tt.ids)
			tt.assertErrFunc(t, err)
			assert.Equal(t, tt.expectedCount, len(retrieved))

			if err == nil && tt.expectedCount > 0 {
				for _, id := range tt.ids {
					if notification, ok := retrieved[id]; ok {
						// 验证ID
						assert.Equal(t, id, notification.ID)
					}
				}
			}
		})
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryCreateWithCallbackLog() {
	t := s.T()

	bizID := int64(12)
	notification := s.createTestNotification(bizID)

	// 设置配额
	_ = s.createTestQuota(t, notification)

	// 测试创建带回调记录的通知
	created, err := s.repo.CreateWithCallbackLog(t.Context(), notification)
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	s.assertNotification(t, notification, created)

	// 验证回调记录是否创建成功
	logs, err := s.callbackLogRepo.FindByNotificationIDs(t.Context(), []uint64{created.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, created.ID, logs[0].Notification.ID)
	assert.Equal(t, domain.CallbackLogStatusInit, logs[0].Status)
}

func (s *NotificationServiceTestSuite) createTestQuota(t *testing.T, notification domain.Notification) int32 {
	quota := int32(100)
	err := s.quotaCache.CreateOrUpdate(t.Context(), domain.Quota{
		BizID:   notification.BizID,
		Quota:   quota,
		Channel: notification.Channel,
	})
	require.NoError(t, err)
	return quota
}

func (s *NotificationServiceTestSuite) TestRepositoryBatchCreateWithCallbackLog() {
	t := s.T()

	bizID := int64(13)
	notifications := []domain.Notification{
		s.createTestNotification(bizID),
		s.createTestNotification(bizID),
	}
	s.createTestQuota(t, domain.Notification{
		BizID:   bizID,
		Channel: domain.ChannelSMS,
	})

	// 设置配额
	_ = s.createTestQuota(t, notifications[0])

	// 测试批量创建带回调记录的通知
	created, err := s.repo.BatchCreateWithCallbackLog(t.Context(), notifications)
	require.NoError(t, err)
	assert.Equal(t, len(notifications), len(created))

	// 收集所有创建的通知ID
	var notificationIDs []uint64
	for _, n := range created {
		notificationIDs = append(notificationIDs, n.ID)
	}

	// 验证回调记录是否创建成功
	logs, err := s.callbackLogRepo.FindByNotificationIDs(t.Context(), notificationIDs)
	require.NoError(t, err)
	assert.Equal(t, len(created), len(logs))

	// 创建回调记录ID到通知ID的映射
	logsMap := make(map[uint64]domain.CallbackLog)
	for _, log := range logs {
		logsMap[log.Notification.ID] = log
	}

	// 验证每个通知都有对应的回调记录
	for _, n := range created {
		log, ok := logsMap[n.ID]
		assert.True(t, ok, "应该为通知ID %d 创建回调记录", n.ID)
		assert.Equal(t, n.ID, log.Notification.ID)
		assert.Equal(t, domain.CallbackLogStatusInit, log.Status)
	}
}

func (s *NotificationServiceTestSuite) TestRepositoryMarkFailed() {
	t := s.T()

	bizID := int64(14)
	notification := s.createTestNotification(bizID)

	// 设置配额
	initQuota := s.createTestQuota(t, notification)

	// 创建通知
	created, err := s.repo.Create(t.Context(), notification)
	require.NoError(t, err)

	// 验证配额
	updatedQuota, err := s.quotaRepo.Find(t.Context(), notification.BizID, notification.Channel)
	require.NoError(t, err)
	assert.Equal(t, initQuota-1, updatedQuota.Quota)

	// 标记为失败
	created.Status = domain.SendStatusFailed
	err = s.repo.MarkFailed(t.Context(), created)
	require.NoError(t, err)

	// 验证状态更新以及配额回退
	updated, err := s.repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SendStatusFailed, updated.Status)
	assert.Greater(t, updated.Version, created.Version)

	// 验证配额是否回退
	updatedQuota, err = s.quotaRepo.Find(t.Context(), notification.BizID, notification.Channel)
	require.NoError(t, err)
	assert.Equal(t, initQuota, updatedQuota.Quota)
}

func (s *NotificationServiceTestSuite) TestRepositoryMarkSuccess() {
	t := s.T()

	bizID := int64(15)
	notification := s.createTestNotification(bizID)

	// 设置配额
	initQuota := s.createTestQuota(t, notification)

	// 先创建通知和回调记录
	created, err := s.repo.CreateWithCallbackLog(t.Context(), notification)
	require.NoError(t, err)

	// 标记为成功
	created.Status = domain.SendStatusSucceeded
	err = s.repo.MarkSuccess(t.Context(), created)
	require.NoError(t, err)

	// 验证配额
	updatedQuota, err := s.quotaRepo.Find(t.Context(), notification.BizID, notification.Channel)
	require.NoError(t, err)
	assert.Equal(t, initQuota-1, updatedQuota.Quota)

	// 验证状态更新
	updated, err := s.repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SendStatusSucceeded, updated.Status)
	assert.Greater(t, updated.Version, created.Version)

	// 验证回调记录状态是否更新为可发送
	logs, err := s.callbackLogRepo.FindByNotificationIDs(t.Context(), []uint64{created.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(logs))
	assert.Equal(t, domain.CallbackLogStatusPending, logs[0].Status)
}

func (s *NotificationServiceTestSuite) TestRepositoryFindReadyNotifications() {
	t := s.T()

	bizID := int64(16)

	// 创建一些测试数据
	// 1. 当前时间内的通知（应该被查出）
	now := time.Now()
	readyNotification1 := s.createTestNotification(bizID)
	readyNotification1.Key = fmt.Sprintf("ready-key-1-%d", now.Unix())
	readyNotification1.ScheduledSTime = now.Add(-1 * time.Minute) // 1分钟前开始
	readyNotification1.ScheduledETime = now.Add(1 * time.Hour)    // 1小时后结束

	readyNotification2 := s.createTestNotification(bizID)
	readyNotification2.Key = fmt.Sprintf("ready-key-2-%d", now.Unix())
	readyNotification2.ScheduledSTime = now.Add(-2 * time.Minute) // 2分钟前开始
	readyNotification2.ScheduledETime = now.Add(2 * time.Hour)    // 2小时后结束

	// 2. 未来的通知（不应该被查出）
	futureNotification := s.createTestNotification(bizID)
	futureNotification.Key = fmt.Sprintf("future-key-%d", now.Unix())
	futureNotification.ScheduledSTime = now.Add(1 * time.Hour) // 1小时后开始
	futureNotification.ScheduledETime = now.Add(2 * time.Hour) // 2小时后结束

	// 3. 过期的通知（不应该被查出）
	expiredNotification := s.createTestNotification(bizID)
	expiredNotification.Key = fmt.Sprintf("expired-key-%d", now.Unix())
	expiredNotification.ScheduledSTime = now.Add(-2 * time.Hour) // 2小时前开始
	expiredNotification.ScheduledETime = now.Add(-1 * time.Hour) // 1小时前结束

	// 4. 状态不为PENDING的通知（不应该被查出）
	nonPendingNotification := s.createTestNotification(bizID)
	nonPendingNotification.Key = fmt.Sprintf("non-pending-key-%d", now.Unix())
	nonPendingNotification.ScheduledSTime = now.Add(-30 * time.Minute) // 30分钟前开始
	nonPendingNotification.ScheduledETime = now.Add(30 * time.Minute)  // 30分钟后结束
	nonPendingNotification.Status = domain.SendStatusSucceeded         // 成功状态

	// 设置配额
	_ = s.createTestQuota(t, readyNotification1)

	// 创建所有测试通知
	_, err := s.repo.BatchCreate(t.Context(), []domain.Notification{
		readyNotification1,
		readyNotification2,
		futureNotification,
		expiredNotification,
		nonPendingNotification,
	})
	require.NoError(t, err)

	// 测试查找准备好的通知
	readyNotifications, err := s.repo.FindReadyNotifications(t.Context(), 0, 10)
	require.NoError(t, err)

	// 验证结果
	assert.GreaterOrEqual(t, len(readyNotifications), 2, "应该至少有2个准备好的通知")

	// 验证查询的通知是否是预期中准备好的通知
	readyKeys := make(map[string]bool)
	for _, n := range readyNotifications {
		if n.BizID == bizID {
			readyKeys[n.Key] = true
		}
	}

	assert.True(t, readyKeys[readyNotification1.Key], "应该包含ready-key-1")
	assert.True(t, readyKeys[readyNotification2.Key], "应该包含ready-key-2")
	assert.False(t, readyKeys[futureNotification.Key], "不应该包含future-key")
	assert.False(t, readyKeys[expiredNotification.Key], "不应该包含expired-key")
	assert.False(t, readyKeys[nonPendingNotification.Key], "不应该包含non-pending-key")
}

func (s *NotificationServiceTestSuite) TestRepositoryMarkTimeoutSendingAsFailed() {
	t := s.T()

	bizID := int64(17)
	notification := s.createTestNotification(bizID)

	// 设置配额
	_ = s.createTestQuota(t, notification)

	// 创建通知
	created, err := s.repo.Create(t.Context(), notification)
	require.NoError(t, err)

	// 将通知状态改为SENDING
	created.Status = domain.SendStatusSending
	err = s.repo.UpdateStatus(t.Context(), created)
	require.NoError(t, err)

	// 修改更新时间为1分钟前（模拟超时）
	err = s.db.Exec("UPDATE `notifications` SET utime = ? WHERE id = ?",
		time.Now().Add(-2*time.Minute).UnixMilli(), created.ID).Error
	require.NoError(t, err)

	// 测试标记超时发送的通知为失败
	affectedRows, err := s.repo.MarkTimeoutSendingAsFailed(t.Context(), 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, affectedRows, int64(1), "应该至少有1条记录被标记为失败")

	// 验证通知状态是否已更新为失败
	updated, err := s.repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.SendStatusFailed, updated.Status)
}

func (s *NotificationServiceTestSuite) TestServiceFindReadyNotifications() {
	t := s.T()

	bizID := int64(18)

	// 创建测试数据
	now := time.Now()
	readyNotification := s.createTestNotification(bizID)
	readyNotification.Key = fmt.Sprintf("service-ready-key-%d", now.Unix())
	readyNotification.ScheduledSTime = now.Add(-1 * time.Minute)
	readyNotification.ScheduledETime = now.Add(1 * time.Hour)

	// 设置配额
	_ = s.createTestQuota(t, readyNotification)

	// 创建测试通知
	created, err := s.repo.Create(t.Context(), readyNotification)
	require.NoError(t, err)

	// 测试Service层的FindReadyNotifications方法
	readyNotifications, err := s.svc.FindReadyNotifications(t.Context(), 0, 10)
	require.NoError(t, err)

	// 验证结果
	found := false
	for _, n := range readyNotifications {
		if n.ID == created.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "应该能找到刚创建的准备好发送的通知")
}
