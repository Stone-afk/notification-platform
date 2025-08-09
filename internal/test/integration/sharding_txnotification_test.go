//go:build e2e

package integration

import (
	"testing"

	sharding2 "notification-platform/internal/pkg/sharding"

	idgen "notification-platform/internal/pkg/idgenerator"

	"github.com/stretchr/testify/assert"

	"github.com/ecodeclub/ekit/syncx"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/repository/dao/sharding"
	shardingIoc "notification-platform/internal/test/integration/ioc/sharding"
)

type ShardingTxNotificationSuite struct {
	suite.Suite
	dbs                 *syncx.Map[string, *gorm.DB]
	txnShardingDAO      *sharding.TxNShardingDAO
	notificationDAO     *sharding.NotificationShardingDAO
	notificationStr     sharding2.ShardingStrategy
	txnShardingStrategy sharding2.ShardingStrategy
}

func (s *ShardingTxNotificationSuite) SetupSuite() {
	dbs := shardingIoc.InitDbs()
	notiStrategy, callbacklogStrategy := shardingIoc.InitNotificationSharding()
	notiForTxn, txnStrategy := shardingIoc.InitTxnSharding()

	s.dbs = dbs
	s.notificationDAO = sharding.NewNotificationShardingDAO(dbs, notiStrategy, callbacklogStrategy, idgen.NewGenerator())

	// Use the constructor instead of direct initialization
	s.txnShardingDAO = sharding.NewTxNShardingDAO(dbs, notiForTxn, txnStrategy)

	s.notificationStr = notiStrategy
	s.txnShardingStrategy = txnStrategy
}

func (s *ShardingTxNotificationSuite) TearDownTest() {
	s.clearTables()
}

func (s *ShardingTxNotificationSuite) TestPrepare() {
	t := s.T()

	// Create a TxNotification
	txn := dao.TxNotification{
		BizID:         20001,
		Key:           "tx_prepare_test_001",
		Status:        domain.TxNotificationStatusPrepare.String(),
		NextCheckTime: 123,
		CheckCount:    1,
		Ctime:         0,
		Utime:         0,
	}

	// Create a notification
	notification := dao.Notification{
		ID:                0,
		BizID:             20001,
		Key:               "tx_prepare_test_001",
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
	// Call Prepare
	notificationID, err := s.txnShardingDAO.Prepare(t.Context(), txn, notification)
	require.NoError(t, err)
	notification.ID = notificationID
	dst := s.notificationStr.Shard(20001, "tx_prepare_test_001")
	iddst := s.notificationStr.ShardWithID(int64(notificationID))
	assert.Equal(t, dst, iddst)

	// Create expected and actual notification maps for assertion
	expectedNotifications := map[[2]string][]dao.Notification{
		{
			"notification_0",
			"notification_0",
		}: {},
		{
			"notification_0",
			"notification_1",
		}: {},
		{
			"notification_1",
			"notification_0",
		}: {},
		{
			"notification_1",
			"notification_1",
		}: {notification},
	}

	// Use the helper method to assert notifications
	actualNotifications := s.getAllNotifications()
	s.assertNotifications(expectedNotifications, actualNotifications)

	// Set the expected NotificationID since we don't know it beforehand
	txn.NotificationID = notification.ID

	// Create expected and actual tx notification maps for assertion
	expectedTxNotifications := map[[2]string][]dao.TxNotification{
		{
			"notification_0",
			"tx_notification_0",
		}: {},
		{
			"notification_0",
			"tx_notification_1",
		}: {},
		{
			"notification_1",
			"tx_notification_0",
		}: {},
		{
			"notification_1",
			"tx_notification_1",
		}: {txn},
	}
	actualTxNotifications := s.getAllTxNotifications()
	// Use the helper method to assert tx notifications
	s.assertTxNotifications(expectedTxNotifications, actualTxNotifications)
}

func (s *ShardingTxNotificationSuite) TestUpdateStatus() {
	t := s.T()

	// Create a TxNotification and Notification first
	txn := dao.TxNotification{
		BizID:  20004,
		Key:    "tx_update_status_test_001",
		Status: domain.TxNotificationStatusPrepare.String(),
		Ctime:  0,
		Utime:  0,
	}

	notification := dao.Notification{
		ID:                0,
		BizID:             20004,
		Key:               "tx_update_status_test_001",
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

	id, err := s.txnShardingDAO.Prepare(t.Context(), txn, notification)
	require.NoError(t, err)
	notification.ID = id

	initialNotification, err := s.notificationDAO.GetByID(t.Context(), notification.ID)
	require.NoError(t, err)

	err = s.txnShardingDAO.UpdateStatus(
		t.Context(),
		txn.BizID,
		txn.Key,
		domain.TxNotificationStatusCommit,
		domain.SendStatusPending,
	)
	require.NoError(t, err)

	updatedTxn := s.getTxNotificationByBizIDAndKey(txn.BizID, txn.Key)
	require.Equal(t, domain.TxNotificationStatusCommit.String(), updatedTxn.Status)

	updatedNotification, err := s.notificationDAO.GetByID(t.Context(), notification.ID)
	require.NoError(t, err)
	require.Equal(t, domain.SendStatusPending.String(), updatedNotification.Status)
	require.True(t, updatedNotification.Utime > initialNotification.Utime)
}

func (s *ShardingTxNotificationSuite) TestUpdateStatus_InvalidStatus() {
	t := s.T()

	// Create a TxNotification with a non-PREPARE status
	txn := dao.TxNotification{
		BizID:  1003,
		Key:    "tx_update_invalid_status_test_001",
		Status: domain.TxNotificationStatusPrepare.String(),
		Ctime:  0,
		Utime:  0,
	}

	notification := dao.Notification{
		ID:                175000000000000002,
		BizID:             1003,
		Key:               "tx_update_invalid_status_test_001",
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

	// Prepare the TxNotification and Notification
	_, err := s.txnShardingDAO.Prepare(t.Context(), txn, notification)
	require.NoError(t, err)

	// First update should succeed
	err = s.txnShardingDAO.UpdateStatus(
		t.Context(),
		txn.BizID,
		txn.Key,
		domain.TxNotificationStatus("SUCCEEDED"),
		domain.SendStatusSucceeded,
	)
	require.NoError(t, err)

	// Second update should fail because status is no longer PREPARE
	err = s.txnShardingDAO.UpdateStatus(
		t.Context(),
		txn.BizID,
		txn.Key,
		domain.TxNotificationStatus("FAILED"),
		domain.SendStatusFailed,
	)
	require.Error(t, err)
	require.Equal(t, dao.ErrUpdateStatusFailed, err)
}

func TestShardingTxNotificationSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ShardingTxNotificationSuite))
}

func (s *ShardingTxNotificationSuite) clearTables() {
	s.T().Helper()
	s.dbs.Range(func(_ string, db *gorm.DB) bool {
		err := db.Exec("delete from `notification_0` where biz_id > 20000 AND biz_id < 30000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `notification_1` where biz_id > 20000 AND biz_id < 30000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `tx_notification_0` where biz_id > 20000 AND biz_id < 30000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `tx_notification_1` where biz_id > 20000 AND biz_id < 30000").Error
		require.NoError(s.T(), err)
		return true
	})
}

func (s *ShardingTxNotificationSuite) getTxNotificationByBizIDAndKey(bizID int64, key string) dao.TxNotification {
	t := s.T()
	t.Helper()

	dst := s.txnShardingStrategy.Shard(bizID, key)
	db, ok := s.dbs.Load(dst.DB)
	require.True(t, ok)

	var txn dao.TxNotification
	err := db.WithContext(t.Context()).
		Table(dst.Table).
		Where("biz_id = ? AND `key` = ?", bizID, key).
		First(&txn).Error
	require.NoError(t, err)

	return txn
}

func (s *ShardingTxNotificationSuite) assertNotifications(wantNotifications, actualNotifications map[[2]string][]dao.Notification) {
	require.Equal(s.T(), len(wantNotifications), len(actualNotifications))
	for key, wantVal := range wantNotifications {
		actualVal, ok := actualNotifications[key]
		require.True(s.T(), ok)
		require.Equal(s.T(), len(wantVal), len(actualVal))
		for idx := range actualVal {
			wantNotification := wantVal[idx]
			actualNotification := actualVal[idx]
			require.True(s.T(), actualNotification.Ctime > 0)
			require.True(s.T(), actualNotification.Utime > 0)
			actualNotification.Ctime = 0
			actualNotification.Utime = 0
			require.Equal(s.T(), wantNotification, actualNotification)
		}
	}
}

func (s *ShardingTxNotificationSuite) assertTxNotifications(wantTxNotifications, actualTxNotifications map[[2]string][]dao.TxNotification) {
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

func (s *ShardingTxNotificationSuite) getAllNotifications() map[[2]string][]dao.Notification {
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

func (s *ShardingTxNotificationSuite) getNotifications(table string, db *gorm.DB) []dao.Notification {
	t := s.T()
	var notifications []dao.Notification
	err := db.WithContext(t.Context()).Table(table).
		Where(`biz_id > 20000 AND biz_id < 30000`).
		Order("id asc").
		Find(&notifications).Error
	require.NoError(t, err)
	return notifications
}

func (s *ShardingTxNotificationSuite) getAllTxNotifications() map[[2]string][]dao.TxNotification {
	s.T().Helper()
	ans := make(map[[2]string][]dao.TxNotification)
	s.dbs.Range(func(key string, db *gorm.DB) bool {
		table1 := "tx_notification_0"
		notifications := s.getTxNotifications(table1, db)
		ans[[2]string{key, table1}] = notifications
		table2 := "tx_notification_1"
		notification1s := s.getTxNotifications(table2, db)
		ans[[2]string{key, table2}] = notification1s
		return true
	})
	return ans
}

func (s *ShardingTxNotificationSuite) getTxNotifications(table string, db *gorm.DB) []dao.TxNotification {
	t := s.T()
	var txns []dao.TxNotification
	err := db.WithContext(t.Context()).Table(table).
		Where(`biz_id > 20000 AND biz_id < 30000`).
		Order("notification_id asc").
		Find(&txns).Error
	require.NoError(t, err)
	return txns
}
