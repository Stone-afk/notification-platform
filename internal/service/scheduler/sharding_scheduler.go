package scheduler

import (
	"context"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/errs"
	"notification-platform/internal/pkg/batchsize"
	"notification-platform/internal/pkg/bitring"
	"notification-platform/internal/pkg/loopjob"
	"notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/sender"
	"sync/atomic"
	"time"
)

// ShardingScheduler 通知调度服务实现
type ShardingScheduler struct {
	repo              repository.NotificationRepository
	sender            sender.NotificationSender
	minLoopDuration   time.Duration
	batchSize         atomic.Uint64
	batchSizeAdjuster batchsize.Adjuster

	errorEvents *bitring.BitRing
	job         *loopjob.ShardingLoopJob
}

// Start 启动调度服务
// 当 ctx 被取消的或者关闭的时候，就会结束循环
func (s *ShardingScheduler) Start(ctx context.Context) {
	go s.job.Run(ctx)
}

func (s *ShardingScheduler) loop(ctx context.Context) error {
	for {
		// 记录开始执行时间
		start := time.Now()

		// 批量发送已就绪的通知
		cnt, err := s.batchSendReadyNotifications(ctx)

		// 记录响应时间
		responseTime := time.Since(start)

		// 记录错误事件
		s.errorEvents.Add(err != nil)
		// 判断错误事件是否已达到预设的条件 —— 连续出现三次错误，或者错误率达到阈值
		if s.errorEvents.IsConditionMet() {
			return errs.ErrErrorConditionIsMet
		}

		// 根据响应时间调整batchSize
		newBatchSize, err1 := s.batchSizeAdjuster.Adjust(ctx, responseTime)
		if err1 == nil {
			s.batchSize.Store(uint64(newBatchSize))
		}

		// 没有数据时，响应时间非常快，需要等待一段时间
		if cnt == 0 {
			time.Sleep(s.minLoopDuration - responseTime)
			continue
		}
	}
}

// batchSendReadyNotifications 批量发送已就绪的通知
func (s *ShardingScheduler) batchSendReadyNotifications(ctx context.Context) (int, error) {
	const defaultTimeout = 3 * time.Second

	loopCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	const offset = 0
	notifications, err := s.repo.FindReadyNotifications(loopCtx, offset, int(s.batchSize.Load()))
	if err != nil {
		return 0, err
	}

	if len(notifications) == 0 {
		return 0, nil
	}

	_, err = s.sender.BatchSend(ctx, notifications)
	return len(notifications), err
}

// NewShardingScheduler 创建通知调度服务
func NewShardingScheduler(
	repo repository.NotificationRepository,
	notificationSender sender.NotificationSender,
	dclient dlock.Client,
	shardingStrategy sharding.ShardingStrategy,
	sem loopjob.ResourceSemaphore,
	minLoopDuration time.Duration,
	batchSize int,
	batchSizeAdjuster batchsize.Adjuster,
	errorEvents *bitring.BitRing,
) *ShardingScheduler {
	const key = "notification_platform_async_sharding_scheduler"
	s := &ShardingScheduler{
		repo:              repo,
		sender:            notificationSender,
		minLoopDuration:   minLoopDuration,
		batchSizeAdjuster: batchSizeAdjuster,
		errorEvents:       errorEvents,
	}
	s.job = loopjob.NewShardingLoopJob(dclient, key, s.loop, shardingStrategy, sem)
	s.batchSize.Store(uint64(batchSize))
	return s
}
