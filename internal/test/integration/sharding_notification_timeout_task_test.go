package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/syncx"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/loopjob"
	shardingStr "notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/dao"
	"notification-platform/internal/repository/dao/sharding"
	notificationtask "notification-platform/internal/service/notification/task"
	shardingIoc "notification-platform/internal/test/integration/ioc/sharding"
	testioc "notification-platform/internal/test/ioc"
)

type ShardingNotificationTimeoutTaskSuite struct {
	suite.Suite
	taskDao     *sharding.NotificationTask
	taskService *notificationtask.SendingTimeoutTaskV2
	nstr        shardingStr.ShardingStrategy
	dbs         *syncx.Map[string, *egorm.Component]
}

func (s *ShardingNotificationTimeoutTaskSuite) SetupSuite() {
	dbs := shardingIoc.InitDbs()
	s.taskDao = sharding.NewNotificationTask(dbs)

	repo := repository.NewNotificationRepository(s.taskDao, nil)
	redisClient := testioc.InitRedisClient()
	lockClient := testioc.InitDistributedLock(redisClient)
	nstr, _ := shardingIoc.InitNotificationSharding()
	s.nstr = nstr
	sem := loopjob.NewResourceSemaphore(20)
	s.taskService = notificationtask.NewSendingTimeoutTaskV2(lockClient, repo, sem, nstr)

	s.dbs = dbs
}

func (s *ShardingNotificationTimeoutTaskSuite) TearDownSuite() {
}

func (s *ShardingNotificationTimeoutTaskSuite) TestRepositoryMarkTimeoutSendingAsFailed() {
	t := s.T()
	bizID := int64(40001)
	key := "timeout_001"
	now := time.Now()
	noti := dao.Notification{
		ID:                177677855699636224,
		BizID:             bizID,
		Key:               "timeout_001",
		Receivers:         `["+8613812345678", "+8613912345678"]`,
		Channel:           "SMS",
		TemplateID:        5001,
		TemplateVersionID: 2,
		TemplateParams:    `{"order_id":"20230425001","amount":"299.00"}`,
		Status:            domain.SendStatusSending.String(),
		ScheduledSTime:    1735660800,
		ScheduledETime:    1735747200,
		Version:           1,
		Ctime:             now.UnixMilli(),
		Utime:             now.Add(-2 * time.Minute).UnixMilli(),
	}
	dst := s.nstr.Shard(bizID, key)
	gormDB, ok := s.dbs.Load(dst.DB)
	require.True(t, ok)
	err := gormDB.WithContext(s.T().Context()).Table(dst.Table).Where("biz_id = ? and `key` = ?", bizID, key).Delete(&dao.Notification{}).Error
	require.NoError(t, err)
	// 创建通知
	err = gormDB.WithContext(s.T().Context()).Table(dst.Table).Create(noti).Error
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	s.taskService.Start(ctx)
	<-ctx.Done()
	var res dao.Notification
	err = gormDB.WithContext(s.T().Context()).Table(dst.Table).
		Where("biz_id = ? and `key` = ?", bizID, key).Scan(&res).Error
	require.NoError(t, err)
	// 验证通知状态是否已更新为失败
	require.NoError(t, err)
	assert.Equal(t, domain.SendStatusFailed.String(), res.Status)
}

func TestShardingNotificationTimeoutTask(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ShardingNotificationTimeoutTaskSuite))
}
