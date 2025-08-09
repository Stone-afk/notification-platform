//go:build e2e

package integration

import (
	"context"
	"testing"
	"time"

	"notification-platform/internal/pkg/loopjob"

	"github.com/ego-component/egorm"

	"github.com/ecodeclub/ekit/syncx"

	"github.com/stretchr/testify/assert"

	"github.com/ego-component/eetcd/registry"
	"github.com/gotomicro/ego/client/egrpc/resolver"
	"github.com/meoying/dlock-go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	clientv1 "notification-platform/api/proto/gen/client/v1"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/retry"
	sharding2 "notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/repository/dao/sharding"
	configmocks "notification-platform/internal/service/config/mocks"
	notificationtask "notification-platform/internal/service/notification/task"
	shardingIoc "notification-platform/internal/test/integration/ioc/sharding"
	"notification-platform/internal/test/integration/testgrpc"
	testioc "notification-platform/internal/test/ioc"
)

// MockGrpcServer implements the TransactionCheckServiceServer interface.
type MockShardingGrpcServer struct {
	clientv1.UnimplementedTransactionCheckServiceServer
}

// Check simulates checking the transaction status based on the transaction key.
func (s *MockShardingGrpcServer) Check(_ context.Context, req *clientv1.TransactionCheckServiceCheckRequest) (*clientv1.TransactionCheckServiceCheckResponse, error) {
	// Simulate different responses based on the key
	switch req.Key {
	case "tx_1", "tx_11":
		// Return COMMITTED status
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_COMMITTED,
		}, nil
	case "tx_44":
		// Return CANCEL status
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_CANCEL,
		}, nil
	case "tx_22":
		// Return UNKNOWN status but will fail due to max retries
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_UNKNOWN,
		}, nil
	case "tx_23":
		// Return UNKNOWN status to trigger retry
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_UNKNOWN,
		}, nil
	default:
		return &clientv1.TransactionCheckServiceCheckResponse{
			Status: clientv1.TransactionCheckServiceCheckResponse_UNKNOWN,
		}, nil
	}
}

type ShardingTxNotificationTask struct {
	suite.Suite
	dbs                 *syncx.Map[string, *egorm.Component]
	txnRepo             repository.TxNotificationRepository
	txnDAO              *sharding.TxNShardingDAO
	notificationStr     sharding2.ShardingStrategy
	txnShardingStrategy sharding2.ShardingStrategy
	lock                dlock.Client
}

func (s *ShardingTxNotificationTask) SetupSuite() {
	dbs := shardingIoc.InitDbs()
	notiStrategy, txnStrategy := shardingIoc.InitTxnSharding()
	s.dbs = dbs
	s.txnDAO = sharding.NewTxNShardingDAO(dbs, notiStrategy, txnStrategy)

	// 使用真实的 TxnTaskDAO 作为 DAO 层实现
	txnTaskDAO := sharding.NewTxnTaskDAO(dbs, txnStrategy)

	s.txnRepo = repository.NewTxNotificationRepository(txnTaskDAO)

	s.notificationStr = notiStrategy
	s.txnShardingStrategy = txnStrategy

	// Initialize distributed lock with Redis client
	redisClient := testioc.InitRedisClient()
	s.lock = testioc.InitDistributedLock(redisClient)
}

func (s *ShardingTxNotificationTask) TearDownTest() {
	s.clearTables()
}

func (s *ShardingTxNotificationTask) TestCheckBackTaskV2() {
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

	task := notificationtask.NewTxCheckTaskV2(s.txnRepo, configSvc, s.lock, s.txnShardingStrategy, s.notificationStr, loopjob.NewResourceSemaphore(20))

	// Setup test data across shards
	now := time.Now().UnixMilli()

	// Creating test transactions with different states
	// These will be distributed across different shards based on bizID and key
	txnsToCreate := []struct {
		bizID      int64
		key        string
		status     string
		nextCheck  int64
		checkCount int64
	}{
		// Will be checked and committed - in PREPARE state with next check time in the past
		{30001, "tx_1", domain.TxNotificationStatusPrepare.String(), now - (time.Second * 11).Milliseconds(), 1},
		// Won't be checked - already in COMMIT state
		{30002, "tx_2", domain.TxNotificationStatusCommit.String(), 0, 1},
		// Won't be checked - already in CANCEL state
		{30003, "tx_3", domain.TxNotificationStatusCancel.String(), 0, 1},
		// Won't be checked - already in FAIL state
		{30004, "tx_4", domain.TxNotificationStatusFail.String(), 0, 1},
		// Won't be checked - next check time is in the future
		{30005, "tx_5", domain.TxNotificationStatusPrepare.String(), now + (time.Second * 30).Milliseconds(), 1},
		// Will be checked and committed - on last retry
		{30001, "tx_11", domain.TxNotificationStatusPrepare.String(), now - (time.Second * 11).Milliseconds(), 2},
		// Will be checked and failed - max retries reached
		{30001, "tx_22", domain.TxNotificationStatusPrepare.String(), now - (time.Second * 11).Milliseconds(), 3},
		// Will be checked but unknown result - will be retried
		{30001, "tx_23", domain.TxNotificationStatusPrepare.String(), now - (time.Second * 11).Milliseconds(), 1},
		// Will be checked and canceled
		{30002, "tx_44", domain.TxNotificationStatusPrepare.String(), now - (time.Second * 11).Milliseconds(), 1},
	}

	// Create the transactions in the sharded database
	s.createShardedTxNotifications(txnsToCreate)

	// Initialize ETCD registry and gRPC server for transaction check service
	etcdClient := testioc.InitEtcdClient()
	reg := registry.Load("").Build(registry.WithClientEtcd(etcdClient))

	// Start the mock gRPC server
	go func() {
		server := testgrpc.NewServer[clientv1.TransactionCheckServiceServer](
			"order.notification.callback.service",
			reg,
			&MockShardingGrpcServer{},
			func(s grpc.ServiceRegistrar, srv clientv1.TransactionCheckServiceServer) {
				clientv1.RegisterTransactionCheckServiceServer(s, srv)
			},
		)
		err := server.Start("127.0.0.1:30002")
		if err != nil {
			require.NoError(t, err)
		}
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)
	resolver.Register("etcd", reg)

	// Run the task for a short duration
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	task.Start(ctx)
	<-ctx.Done()

	// Get all notifications and tx notifications after task execution
	actualNotifications := s.getAllNotifications()
	actualTxNotifications := s.getAllTxNotifications()

	// Define expected notifications based on the test data and expected outcomes
	expectedTxNotifications := map[[2]string][]dao.TxNotification{}
	expectedNotifications := map[[2]string][]dao.Notification{}

	// We'll use the same test data as in the test setup to build our expectations
	testData := []struct {
		bizID      int64
		key        string
		status     string
		nextCheck  int64
		checkCount int64
	}{
		// Will be checked and committed - in PREPARE state with next check time in the past
		{30001, "tx_1", domain.TxNotificationStatusCommit.String(), 0, 2},
		// Won't be checked - already in COMMIT state
		{30002, "tx_2", domain.TxNotificationStatusCommit.String(), 0, 1},
		// Won't be checked - already in CANCEL state
		{30003, "tx_3", domain.TxNotificationStatusCancel.String(), 0, 1},
		// Won't be checked - already in FAIL state
		{30004, "tx_4", domain.TxNotificationStatusFail.String(), 0, 1},
		// Won't be checked - next check time is in the future
		{30005, "tx_5", domain.TxNotificationStatusPrepare.String(), now + (time.Second * 30).Milliseconds(), 1},
		// Will be checked and committed - on last retry
		{30001, "tx_11", domain.TxNotificationStatusCommit.String(), 0, 3},
		// Will be checked and failed - max retries reached
		{30001, "tx_22", domain.TxNotificationStatusFail.String(), 0, 4},
		// Will be checked but unknown result - will be retried
		{30001, "tx_23", domain.TxNotificationStatusPrepare.String(), now + (time.Second * 30).Milliseconds(), 2},
		// Will be checked and canceled
		{30002, "tx_44", domain.TxNotificationStatusCancel.String(), 0, 2},
	}

	// Process each test case to set up expected state
	for _, txData := range testData {
		// Determine the database and table for this transaction based on sharding
		dbDst := s.txnShardingStrategy.Shard(txData.bizID, txData.key)
		txTableName := dbDst.Table
		notiDst := s.notificationStr.Shard(txData.bizID, txData.key)

		// Add to expected tx notifications
		txKey := [2]string{dbDst.DB, txTableName}
		if _, ok := expectedTxNotifications[txKey]; !ok {
			expectedTxNotifications[txKey] = []dao.TxNotification{}
		}
		expectedTxNotifications[txKey] = append(expectedTxNotifications[txKey], dao.TxNotification{
			BizID:         txData.bizID,
			Key:           txData.key,
			Status:        txData.status,
			CheckCount:    int(txData.checkCount),
			NextCheckTime: txData.nextCheck,
		})

		// Add to expected notifications
		notKey := [2]string{notiDst.DB, notiDst.Table}
		if _, ok := expectedNotifications[notKey]; !ok {
			expectedNotifications[notKey] = []dao.Notification{}
		}
		expectedNotifications[notKey] = append(expectedNotifications[notKey], dao.Notification{
			BizID:   txData.bizID,
			Key:     txData.key,
			Channel: "SMS",
			Status:  getStatus(txData.status),
		})
	}
	txnDsts := s.txnShardingStrategy.Broadcast()
	for _, dst := range txnDsts {
		if _, ok := expectedTxNotifications[[2]string{dst.DB, dst.Table}]; !ok {
			expectedTxNotifications[[2]string{dst.DB, dst.Table}] = []dao.TxNotification{}
		}
	}
	notiDsts := s.notificationStr.Broadcast()
	for _, dst := range notiDsts {
		if _, ok := expectedNotifications[[2]string{dst.DB, dst.Table}]; !ok {
			expectedNotifications[[2]string{dst.DB, dst.Table}] = []dao.Notification{}
		}
	}
	s.assertTxNotifications(expectedTxNotifications, actualTxNotifications)
	s.assertNotifications(expectedNotifications, actualNotifications)
}

func (s *ShardingTxNotificationTask) assertTxNotifications(wantTxNotifications, actualTxNotifications map[[2]string][]dao.TxNotification) {
	require.Equal(s.T(), len(wantTxNotifications), len(actualTxNotifications))
	for key, wantVal := range wantTxNotifications {
		actualVal, ok := actualTxNotifications[key]
		require.True(s.T(), ok)
		require.Equal(s.T(), len(wantVal), len(actualVal))
		for idx := range actualVal {
			wantNotification := wantVal[idx]
			actualNotification := actualVal[idx]
			assert.Equal(s.T(), wantNotification.Status, actualNotification.Status)
			assert.Equal(s.T(), wantNotification.Key, actualNotification.Key)
			assert.Equal(s.T(), wantNotification.BizID, actualNotification.BizID)
			assert.Equal(s.T(), wantNotification.CheckCount, actualNotification.CheckCount)

		}
	}
}

func (s *ShardingTxNotificationTask) assertNotifications(wantNotifications, actualNotifications map[[2]string][]dao.Notification) {
	require.Equal(s.T(), len(wantNotifications), len(actualNotifications))
	for key, wantVal := range wantNotifications {
		actualVal, ok := actualNotifications[key]
		require.True(s.T(), ok)
		require.Equal(s.T(), len(wantVal), len(actualVal))
		for idx := range actualVal {
			wantNotification := wantVal[idx]
			actualNotification := actualVal[idx]
			assert.Equal(s.T(), wantNotification.Status, actualNotification.Status)
			assert.Equal(s.T(), wantNotification.Key, actualNotification.Key)
			assert.Equal(s.T(), wantNotification.BizID, actualNotification.BizID)
		}
	}
}

func (s *ShardingTxNotificationTask) mockConfigMap() map[int64]domain.BusinessConfig {
	return map[int64]domain.BusinessConfig{
		30001: {
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
		30002: {
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

func (s *ShardingTxNotificationTask) clearTables() {
	s.T().Helper()
	s.dbs.Range(func(_ string, db *gorm.DB) bool {
		err := db.Exec("delete from `notification_0` where biz_id > 30000 AND biz_id < 40000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `notification_1` where biz_id > 30000 AND biz_id < 40000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `tx_notification_0` where biz_id > 30000 AND biz_id < 40000").Error
		require.NoError(s.T(), err)
		err = db.Exec("delete from `tx_notification_1` where biz_id > 30000 AND biz_id < 40000").Error
		require.NoError(s.T(), err)
		return true
	})
}

func (s *ShardingTxNotificationTask) createShardedTxNotifications(txnsData []struct {
	bizID      int64
	key        string
	status     string
	nextCheck  int64
	checkCount int64
},
) {
	t := s.T()

	// Group transactions by shard to batch insert them
	txnsByDstDB := make(map[string]map[string][]dao.TxNotification)

	for _, txData := range txnsData {
		dst := s.txnShardingStrategy.Shard(txData.bizID, txData.key)

		if _, ok := txnsByDstDB[dst.DB]; !ok {
			txnsByDstDB[dst.DB] = make(map[string][]dao.TxNotification)
		}

		now := time.Now().UnixMilli()
		txn := dao.TxNotification{
			Key:           txData.key,
			BizID:         txData.bizID,
			Status:        txData.status,
			CheckCount:    int(txData.checkCount),
			NextCheckTime: txData.nextCheck,
			Ctime:         now,
			Utime:         now,
		}
		noti := dao.Notification{
			BizID:   txData.bizID,
			Key:     txData.key,
			Channel: "SMS",
			Status:  getStatus(txData.status),
			Ctime:   now,
			Utime:   now,
		}
		_, err := s.txnDAO.Prepare(t.Context(), txn, noti)
		require.NoError(t, err)
	}
}

func getStatus(txStatus string) string {
	switch txStatus {
	case "PREPARE":
		return "PREPARE"
	case "CANCEL":
		return "FAILED"
	case "FAIL":
		return "FAILED"
	case "COMMIT":
		return "PENDING"
	}
	return ""
}

func TestShardingTxNotificationTask(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ShardingTxNotificationTask))
}

func (s *ShardingTxNotificationTask) getAllNotifications() map[[2]string][]dao.Notification {
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

func (s *ShardingTxNotificationTask) getNotifications(table string, db *gorm.DB) []dao.Notification {
	t := s.T()
	var notifications []dao.Notification
	err := db.WithContext(t.Context()).Table(table).
		Where(`biz_id > 30000 AND biz_id < 40000`).
		Order("id asc").
		Find(&notifications).Error
	require.NoError(t, err)
	return notifications
}

func (s *ShardingTxNotificationTask) getAllTxNotifications() map[[2]string][]dao.TxNotification {
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

func (s *ShardingTxNotificationTask) getTxNotifications(table string, db *gorm.DB) []dao.TxNotification {
	t := s.T()
	var txns []dao.TxNotification
	err := db.WithContext(t.Context()).Table(table).
		Where(`biz_id > 30000 AND biz_id < 40000`).
		Order("notification_id asc").
		Find(&txns).Error
	require.NoError(t, err)
	return txns
}
