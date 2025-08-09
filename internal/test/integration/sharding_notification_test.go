//go:build e2e

package integration

import (
	"testing"

	sharding2 "notification-platform/internal/pkg/sharding"

	"github.com/stretchr/testify/assert"
	idgen "notification-platform/internal/pkg/idgenerator"

	"github.com/ego-component/egorm"

	"github.com/ecodeclub/ekit/syncx"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/repository/dao/sharding"
	shardingIoc "notification-platform/internal/test/integration/ioc/sharding"
)

type ShardingNotificationSuite struct {
	suite.Suite
	dbs             *syncx.Map[string, *egorm.Component]
	shardingDAO     *sharding.NotificationShardingDAO
	notificationStr sharding2.ShardingStrategy
	callBackStr     sharding2.ShardingStrategy
}

func (s *ShardingNotificationSuite) SetupSuite() {
	dbs := shardingIoc.InitDbs()
	notiStrategy, callbacklogStrategy := shardingIoc.InitNotificationSharding()
	s.dbs = dbs

	s.shardingDAO = sharding.NewNotificationShardingDAO(dbs, notiStrategy, callbacklogStrategy, idgen.NewGenerator())
	s.notificationStr = notiStrategy
	s.callBackStr = callbacklogStrategy
}

func (s *ShardingNotificationSuite) TearDownTest() {
	s.clearTables()
}

func (s *ShardingNotificationSuite) TestCreate() {
	t := s.T()
	// 00
	noti := dao.Notification{
		ID:                0,
		BizID:             1001,
		Key:               "order_paid_20230425_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}
	// 11
	noti1 := dao.Notification{
		ID:                0,
		BizID:             1002,
		Key:               "order_paid_20230425_002",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}
	// 10
	noti2 := dao.Notification{
		ID:                0,
		BizID:             109,
		Key:               "payxxxxd_20230425_003",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}
	notifications := []dao.Notification{noti, noti1, noti2}
	for _, notification := range notifications {
		_, err := s.shardingDAO.Create(t.Context(), notification)
		require.NoError(t, err)
	}
	actualNotifications := s.getAllNotifications()
	s.assertNotifications(map[[2]string][]dao.Notification{
		{
			"notification_0",
			"notification_0",
		}: {
			noti,
		},
		{
			"notification_0",
			"notification_1",
		}: {},
		{
			"notification_1",
			"notification_0",
		}: {
			noti2,
		},
		{
			"notification_1",
			"notification_1",
		}: {
			noti1,
		},
	}, actualNotifications)

	actualCallbackLogs := s.getAllCallbackLogs()
	s.assertCallbackLogs(map[[2]string][]dao.CallbackLog{
		{
			"notification_0",
			"callback_log_0",
		}: {},
		{
			"notification_0",
			"callback_log_1",
		}: {},
		{
			"notification_1",
			"callback_log_0",
		}: {},
		{
			"notification_1",
			"callback_log_1",
		}: {},
	}, actualCallbackLogs)
}

func (s *ShardingNotificationSuite) TestBatchCreate() {
	t := s.T()

	// Create test notifications across different shards
	noti1 := dao.Notification{
		ID:                0,
		BizID:             5001,
		Key:               "batch_create_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	noti2 := dao.Notification{
		ID:                0,
		BizID:             5002,
		Key:               "batch_create_test_002",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425002","amount":"199.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	noti3 := dao.Notification{
		ID:                0,
		BizID:             5003,
		Key:               "batch_create_test_003",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425003","amount":"399.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	// Put notifications in a slice
	notifications := []dao.Notification{noti1, noti2, noti3}

	// Call BatchCreate
	createdNotifications, err := s.shardingDAO.BatchCreate(t.Context(), notifications)
	require.NoError(t, err)
	require.Equal(t, len(notifications), len(createdNotifications))

	// Verify created notifications
	actualNotifications := s.getAllNotifications()

	// Determine expected notification distribution based on sharding
	expectedNotifications := map[[2]string][]dao.Notification{
		{"notification_0", "notification_0"}: {},
		{"notification_0", "notification_1"}: {},
		{"notification_1", "notification_0"}: {},
		{"notification_1", "notification_1"}: {},
	}

	// Add notifications to expected map based on their sharding
	for _, notification := range notifications {
		dst := s.notificationStr.Shard(notification.BizID, notification.Key)
		dstKey := [2]string{dst.DB, dst.Table}
		expectedNotifications[dstKey] = append(expectedNotifications[dstKey], notification)
	}

	s.assertNotifications(expectedNotifications, actualNotifications)

	// Verify no callback logs were created
	actualCallbackLogs := s.getAllCallbackLogs()
	s.assertCallbackLogs(map[[2]string][]dao.CallbackLog{
		{"notification_0", "callback_log_0"}: {},
		{"notification_0", "callback_log_1"}: {},
		{"notification_1", "callback_log_0"}: {},
		{"notification_1", "callback_log_1"}: {},
	}, actualCallbackLogs)
}

func (s *ShardingNotificationSuite) TestBatchCreateWithCallbackLog() {
	t := s.T()

	// Create test notifications across different shards
	noti1 := dao.Notification{
		ID:                0,
		BizID:             6001,
		Key:               "batch_create_with_callback_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	noti2 := dao.Notification{
		ID:                0,
		BizID:             6002,
		Key:               "batch_create_with_callback_002",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425002","amount":"199.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	noti3 := dao.Notification{
		ID:                0,
		BizID:             6003,
		Key:               "batch_create_with_callback_003",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425003","amount":"399.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	// Put notifications in a slice
	notifications := []dao.Notification{noti1, noti2, noti3}

	// Call BatchCreateWithCallbackLog
	createdNotifications, err := s.shardingDAO.BatchCreateWithCallbackLog(t.Context(), notifications)
	require.NoError(t, err)
	require.Equal(t, len(notifications), len(createdNotifications))

	// Verify created notifications
	actualNotifications := s.getAllNotifications()

	// Determine expected notification distribution based on sharding
	expectedNotifications := map[[2]string][]dao.Notification{
		{"notification_0", "notification_0"}: {},
		{"notification_0", "notification_1"}: {},
		{"notification_1", "notification_0"}: {},
		{"notification_1", "notification_1"}: {},
	}

	// Add notifications to expected map based on their sharding
	for _, notification := range notifications {
		dst := s.notificationStr.Shard(notification.BizID, notification.Key)
		dstKey := [2]string{dst.DB, dst.Table}
		expectedNotifications[dstKey] = append(expectedNotifications[dstKey], notification)
	}

	s.assertNotifications(expectedNotifications, actualNotifications)

	// Verify callback logs were created
	actualCallbackLogs := s.getAllCallbackLogs()

	// Verify that callback logs exist for each notification
	notificationMap := make(map[uint64][2]string)
	for _, notification := range createdNotifications {
		dst := s.callBackStr.ShardWithID(int64(notification.ID))
		notificationMap[notification.ID] = [2]string{dst.DB, dst.Table}
	}

	// Check that each notification has a callback log
	for key, logsByTables := range actualCallbackLogs {
		for idx := range logsByTables {
			log := logsByTables[idx]
			vv, ok := notificationMap[log.NotificationID]
			require.True(t, ok)
			assert.Equal(t, vv, key)
		}
	}
}

func (s *ShardingNotificationSuite) TestGetByID() {
	t := s.T()

	noti := dao.Notification{
		ID:                174041368535257088,
		BizID:             1001,
		Key:               "order_paid_getbyid_test",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	created, err := s.shardingDAO.Create(t.Context(), noti)
	require.NoError(t, err)

	retrieved, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created, retrieved)
}

func (s *ShardingNotificationSuite) TestBatchGetByIDs() {
	t := s.T()

	// Create multiple notifications with different sharding destinations
	noti1 := dao.Notification{
		ID:                174044109181030400,
		BizID:             1001,
		Key:               "order_batch_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	noti2 := dao.Notification{
		ID:                174044208453103616, // This should shard to a different DB/table
		BizID:             1002,
		Key:               "order_batch_test_002",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425002","amount":"199.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	// Create the notifications
	created1, err := s.shardingDAO.Create(t.Context(), noti1)
	require.NoError(t, err)
	created2, err := s.shardingDAO.Create(t.Context(), noti2)
	require.NoError(t, err)

	// Collect the IDs for batch retrieval
	ids := []uint64{created1.ID, created2.ID}

	// Add a non-existent ID to test handling of missing notifications
	nonExistentID := uint64(174041368535257999)
	allIDs := append(ids, nonExistentID)

	// Test the BatchGetByIDs method
	retrievedMap, err := s.shardingDAO.BatchGetByIDs(t.Context(), allIDs)
	require.NoError(t, err)

	// Should only have 2 results (the non-existent ID should not be in the map)
	require.Equal(t, 2, len(retrievedMap))

	retrieved1, ok := retrievedMap[created1.ID]
	require.True(t, ok)
	retrieved2, ok := retrievedMap[created2.ID]
	require.True(t, ok)

	_, ok = retrievedMap[nonExistentID]
	require.False(t, ok)
	require.Equal(t, created1, retrieved1)
	require.Equal(t, created2, retrieved2)
}

func (s *ShardingNotificationSuite) TestGetByKey() {
	t := s.T()

	// Create a notification
	noti := dao.Notification{
		ID:                174047664640495616,
		BizID:             1003,
		Key:               "order_getbykey_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	created, err := s.shardingDAO.Create(t.Context(), noti)
	require.NoError(t, err)

	// Test GetByKey
	retrieved, err := s.shardingDAO.GetByKey(t.Context(), created.BizID, created.Key)
	require.NoError(t, err)

	require.Equal(t, created, retrieved)

	// Test with non-existent key
	_, err = s.shardingDAO.GetByKey(t.Context(), created.BizID, "non_existent_key")
	require.Error(t, err)
}

func (s *ShardingNotificationSuite) TestGetByKeys() {
	t := s.T()

	// Create multiple notifications with same bizID but different keys
	bizID := int64(2001)
	noti1 := dao.Notification{
		ID:                174058501397692416,
		BizID:             bizID,
		Key:               "order_getbykeys_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	noti2 := dao.Notification{
		ID:                174058706396958720,
		BizID:             bizID,
		Key:               "order_getbykeys_test_002",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425002","amount":"199.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	created1, err := s.shardingDAO.Create(t.Context(), noti1)
	require.NoError(t, err)
	created2, err := s.shardingDAO.Create(t.Context(), noti2)
	require.NoError(t, err)

	// Create a map to match retrieved notifications by key
	expectedByKey := map[string]dao.Notification{
		created1.Key: created1,
		created2.Key: created2,
	}

	// Add a non-existent key
	nonExistentKey := "non_existent_key"
	keys := []string{created1.Key, created2.Key, nonExistentKey}

	// Test GetByKeys
	retrievedList, err := s.shardingDAO.GetByKeys(t.Context(), bizID, keys...)
	require.NoError(t, err)

	// Should only find 2 notifications (not the non-existent one)
	require.Equal(t, 2, len(retrievedList))

	// Check each retrieved notification matches the expected one
	for _, retrieved := range retrievedList {
		expected, ok := expectedByKey[retrieved.Key]
		require.True(t, ok)

		require.Equal(t, expected, retrieved)
	}
}

func (s *ShardingNotificationSuite) TestCASStatus() {
	t := s.T()

	// Create a notification
	noti := dao.Notification{
		ID:                174066471210094592,
		BizID:             1004,
		Key:               "order_casstatus_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	created, err := s.shardingDAO.Create(t.Context(), noti)
	require.NoError(t, err)

	updateNoti := created
	updateNoti.Status = "SENDING"

	err = s.shardingDAO.CASStatus(t.Context(), updateNoti)
	require.NoError(t, err)

	retrieved, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "SENDING", retrieved.Status)
	require.Equal(t, created.Version+1, retrieved.Version)

	updateNoti.Status = "SUCCEEDED"
	err = s.shardingDAO.CASStatus(t.Context(), updateNoti)
	require.Error(t, err)

	retrieved2, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "SENDING", retrieved2.Status) // Still PROCESSING, not SUCCEEDED
}

func (s *ShardingNotificationSuite) TestUpdateStatus() {
	t := s.T()

	// Create a notification
	noti := dao.Notification{
		ID:                174068747234459648,
		BizID:             1005,
		Key:               "order_updatestatus_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	created, err := s.shardingDAO.Create(t.Context(), noti)
	require.NoError(t, err)

	// Update status (no version check)
	updateNoti := created
	updateNoti.Status = "SUCCEEDED"

	err = s.shardingDAO.UpdateStatus(t.Context(), updateNoti)
	require.NoError(t, err)

	// Retrieve the notification to verify status change
	retrieved, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", retrieved.Status)
	require.Equal(t, created.Version+1, retrieved.Version)

	// Update again, no need to update version in test data unlike CAS
	updateNoti.Status = "FAILED"
	err = s.shardingDAO.UpdateStatus(t.Context(), updateNoti)
	require.NoError(t, err)

	// Verify second update
	retrieved2, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "FAILED", retrieved2.Status)
	require.Equal(t, created.Version+2, retrieved2.Version) // Version incremented twice
}

func (s *ShardingNotificationSuite) TestMarkSuccess() {
	t := s.T()

	// Create a notification
	noti := dao.Notification{
		ID:                174070884113907712,
		BizID:             3001,
		Key:               "order_marksuccess_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	// Create a notification with callback log
	created, err := s.shardingDAO.CreateWithCallbackLog(t.Context(), noti)
	require.NoError(t, err)

	// Mark as success
	updateNoti := created
	updateNoti.Status = "SUCCEEDED"
	err = s.shardingDAO.MarkSuccess(t.Context(), updateNoti)
	require.NoError(t, err)

	// Verify notification status changed
	retrieved, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "SUCCEEDED", retrieved.Status)
	require.Equal(t, created.Version+1, retrieved.Version)

	// Verify callback log status changed to pending
	actualCallbackLogs := s.getAllCallbackLogs()

	// Find the callback log for our notification
	var callbackFound bool
	for _, logs := range actualCallbackLogs {
		for _, log := range logs {
			if log.NotificationID == created.ID {
				callbackFound = true
				require.Equal(t, domain.CallbackLogStatusPending.String(), log.Status)
			}
		}
	}
	require.True(t, callbackFound, "Callback log for notification not found")
}

func (s *ShardingNotificationSuite) TestMarkFailed() {
	t := s.T()

	// Create a notification
	noti := dao.Notification{
		ID:                174071356163481600,
		BizID:             3002,
		Key:               "order_markfailed_test_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	created, err := s.shardingDAO.Create(t.Context(), noti)
	require.NoError(t, err)

	// Mark as failed
	updateNoti := created
	updateNoti.Status = "FAILED"
	err = s.shardingDAO.MarkFailed(t.Context(), updateNoti)
	require.NoError(t, err)

	// Verify notification status changed
	retrieved, err := s.shardingDAO.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "FAILED", retrieved.Status)
	require.Equal(t, created.Version+1, retrieved.Version)
}

func (s *ShardingNotificationSuite) TestBatchUpdateStatusSucceededOrFailed() {
	t := s.T()

	successNoti1 := dao.Notification{
		ID:                174071684895449088,
		BizID:             4001,
		Key:               "order_batchupdate_success_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	successNoti2 := dao.Notification{
		ID:                174071777677746176,
		BizID:             4002,
		Key:               "order_batchupdate_success_002",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425002","amount":"199.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	failedNoti1 := dao.Notification{
		ID:                174071943857635328,
		BizID:             4003,
		Key:               "order_batchupdate_failed_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425003","amount":"399.00"}`,
		Status:            "PENDING",
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             0,
		Utime:             0,
	}

	// Create all notifications with callback logs
	createdSuccess1, err := s.shardingDAO.CreateWithCallbackLog(t.Context(), successNoti1)
	require.NoError(t, err)
	createdSuccess2, err := s.shardingDAO.CreateWithCallbackLog(t.Context(), successNoti2)
	require.NoError(t, err)
	createdFailed1, err := s.shardingDAO.CreateWithCallbackLog(t.Context(), failedNoti1)
	require.NoError(t, err)

	// Update statuses in batch
	successNotifications := []dao.Notification{createdSuccess1, createdSuccess2}
	failedNotifications := []dao.Notification{createdFailed1}

	// Call the batch update method
	err = s.shardingDAO.BatchUpdateStatusSucceededOrFailed(
		t.Context(),
		successNotifications,
		failedNotifications,
	)
	require.NoError(t, err)

	retrieved1, err := s.shardingDAO.GetByID(t.Context(), createdSuccess1.ID)
	require.NoError(t, err)
	require.Equal(t, domain.SendStatusSucceeded.String(), retrieved1.Status)
	require.Equal(t, createdSuccess1.Version+1, retrieved1.Version)

	retrieved2, err := s.shardingDAO.GetByID(t.Context(), createdSuccess2.ID)
	require.NoError(t, err)
	require.Equal(t, domain.SendStatusSucceeded.String(), retrieved2.Status)
	require.Equal(t, createdSuccess2.Version+1, retrieved2.Version)

	// Verify failed notification
	retrievedFailed, err := s.shardingDAO.GetByID(t.Context(), createdFailed1.ID)
	require.NoError(t, err)
	require.Equal(t, domain.SendStatusFailed.String(), retrievedFailed.Status)
	require.Equal(t, createdFailed1.Version+1, retrievedFailed.Version)

	// Verify callback logs for successful notifications are updated to pending
	actualCallbackLogs := s.getAllCallbackLogs()

	successIDs := map[uint64]bool{
		createdSuccess1.ID: false,
		createdSuccess2.ID: false,
	}

	for _, logs := range actualCallbackLogs {
		for _, log := range logs {
			if _, ok := successIDs[log.NotificationID]; ok {
				successIDs[log.NotificationID] = true
				require.Equal(t, domain.CallbackLogStatusPending.String(), log.Status)
			}
		}
	}

	// Ensure all success IDs' callback logs were found and updated
	for id, found := range successIDs {
		require.True(t, found, "Callback log for notification ID %d was not found or not updated", id)
	}
}

func TestShardingNotificationSuite(t *testing.T) {
	suite.Run(t, new(ShardingNotificationSuite))
}

func (s *ShardingNotificationSuite) clearTables() {
	s.T().Helper()
	s.dbs.Range(func(key string, db *gorm.DB) bool {
		err := db.Exec("truncate table `callback_log_0` ").Error
		require.NoError(s.T(), err)
		err = db.Exec("truncate table `callback_log_1`").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `notification_0` where biz_id < 10000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `notification_1` where biz_id < 10000").Error
		require.NoError(s.T(), err)
		return true
	})
}

func (s *ShardingNotificationSuite) getAllNotifications() map[[2]string][]dao.Notification {
	s.T().Helper()
	ans := make(map[[2]string][]dao.Notification)
	s.dbs.Range(func(key string, db *gorm.DB) bool {
		table1 := "notification_0"
		notifications := s.getNotifications(table1, db)
		ans[[2]string{key, table1}] = notifications
		table2 := "notification_1"
		notification1s := s.getNotifications(table2, db)
		ans[[2]string{key, table2}] = notification1s
		return true
	})
	return ans
}

func (s *ShardingNotificationSuite) getNotifications(table string, db *gorm.DB) []dao.Notification {
	t := s.T()
	var notifications []dao.Notification
	err := db.WithContext(t.Context()).Table(table).
		Where("biz_id < 10000").
		Order("id asc").
		Find(&notifications).Error
	require.NoError(t, err)
	return notifications
}

func (s *ShardingNotificationSuite) getAllCallbackLogs() map[[2]string][]dao.CallbackLog {
	s.T().Helper()
	ans := make(map[[2]string][]dao.CallbackLog)
	s.dbs.Range(func(key string, db *gorm.DB) bool {
		table1 := "callback_log_0"
		notifications := s.getCallbackLogs(table1, db)
		ans[[2]string{key, table1}] = notifications
		table2 := "callback_log_1"
		notification1s := s.getCallbackLogs(table2, db)
		ans[[2]string{key, table2}] = notification1s
		return true
	})
	return ans
}

func (s *ShardingNotificationSuite) getCallbackLogs(table string, db *gorm.DB) []dao.CallbackLog {
	t := s.T()
	var logs []dao.CallbackLog
	err := db.WithContext(t.Context()).Table(table).
		Order("notification_id asc").
		Find(&logs).Error
	require.NoError(t, err)
	return logs
}

// 比较消息的通用方法
func (s *ShardingNotificationSuite) assertNotifications(wantNotifications, actualNotifications map[[2]string][]dao.Notification) {
	require.Equal(s.T(), len(wantNotifications), len(actualNotifications))
	for key, wantVal := range wantNotifications {
		actualVal, ok := actualNotifications[key]
		require.True(s.T(), ok)
		require.Equal(s.T(), len(wantVal), len(actualVal))
		for idx := range actualVal {
			actualNotification := actualVal[idx]
			require.True(s.T(), actualNotification.Ctime > 0)
			require.True(s.T(), actualNotification.Utime > 0)
			actualVal[idx].ID = 0
			wantVal[idx].Ctime = 0
			wantVal[idx].Utime = 0
			actualVal[idx].Ctime = 0
			actualVal[idx].Utime = 0
		}
		assert.ElementsMatch(s.T(), wantVal, actualVal)
	}
}

func (s *ShardingNotificationSuite) assertCallbackLogs(wantLogs, actualLogs map[[2]string][]dao.CallbackLog) {
	require.Equal(s.T(), len(wantLogs), len(actualLogs))
	for key, wantVal := range wantLogs {
		actualVal, ok := actualLogs[key]
		require.True(s.T(), ok)
		require.Equal(s.T(), len(wantVal), len(actualVal))
		for idx := range actualVal {
			actualLog := actualVal[idx]
			require.True(s.T(), actualLog.Ctime > 0)
			require.True(s.T(), actualLog.Utime > 0)
			require.True(s.T(), actualLog.NextRetryTime > 0)
			actualLog.Ctime = 0
			actualLog.Utime = 0
			actualLog.NextRetryTime = 0
			actualLog.ID = 0

		}
		require.ElementsMatch(s.T(), wantVal, actualVal)
	}
}

func (s *ShardingNotificationSuite) assertTxNotifications(wantTxNotifications, actualTxNotifications map[[2]string][]dao.TxNotification) {
	require.Equal(s.T(), len(wantTxNotifications), len(actualTxNotifications))
	for key, wantVal := range wantTxNotifications {
		actualVal, ok := actualTxNotifications[key]
		require.True(s.T(), ok)
		require.Equal(s.T(), len(wantVal), len(actualVal))
		for idx := range actualVal {
			wantNotification := wantVal[idx]
			actualNotification := actualVal[idx]
			require.True(s.T(), actualNotification.Ctime > 0)
			require.True(s.T(), actualNotification.Utime > 0)
			actualNotification.TxID = 0
			actualNotification.Ctime = 0
			actualNotification.Utime = 0
			require.Equal(s.T(), wantNotification, actualNotification)
		}
	}
}
