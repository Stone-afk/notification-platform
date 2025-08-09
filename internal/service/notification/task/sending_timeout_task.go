package task

import (
	"context"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/pkg/loopjob"
	"notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository"
	"time"
)

type SendingTimeoutTask struct {
	dclient dlock.Client
	repo    repository.NotificationRepository
}

func (s *SendingTimeoutTask) HandleSendingTimeout(ctx context.Context) error {
	const batchSize = 10
	const defaultSleepTime = time.Second * 10
	cnt, err := s.repo.MarkTimeoutSendingAsFailed(ctx, batchSize)
	if err != nil {
		return err
	}
	// 说明 SENDING 的不多，可以休息一下
	if cnt < batchSize {
		// 这里可以随便设置，在分钟以内都可以
		time.Sleep(defaultSleepTime)
	}
	return nil
}

func (s *SendingTimeoutTask) Start(ctx context.Context) {
	const key = "notification_handling_sending_timeout"
	lj := loopjob.NewInfiniteLoop(s.dclient, s.HandleSendingTimeout, key)
	lj.Run(ctx)
}

func NewSendingTimeoutTask(dclient dlock.Client, repo repository.NotificationRepository) *SendingTimeoutTask {
	return &SendingTimeoutTask{dclient: dclient, repo: repo}
}

type SendingTimeoutTaskV2 struct {
	dclient dlock.Client
	repo    repository.NotificationRepository
	sem     loopjob.ResourceSemaphore
	str     sharding.ShardingStrategy
}

func NewSendingTimeoutTaskV2(dclient dlock.Client,
	repo repository.NotificationRepository,
	sem loopjob.ResourceSemaphore,
	str sharding.ShardingStrategy,
) *SendingTimeoutTaskV2 {
	return &SendingTimeoutTaskV2{dclient: dclient, repo: repo, sem: sem, str: str}
}

func (s *SendingTimeoutTaskV2) Start(ctx context.Context) {
	const key = "notification_handling_sending_timeout_v2"
	lj := loopjob.NewShardingLoopJob(s.dclient, key, s.HandleSendingTimeout, s.str, s.sem)
	go lj.Run(ctx)
}

func (s *SendingTimeoutTaskV2) HandleSendingTimeout(ctx context.Context) error {
	const batchSize = 10
	const defaultSleepTime = time.Second * 10
	cnt, err := s.repo.MarkTimeoutSendingAsFailed(ctx, batchSize)
	if err != nil {
		return err
	}
	// 说明 SENDING 的不多，可以休息一下
	if cnt < batchSize {
		// 这里可以随便设置，在分钟以内都可以
		time.Sleep(defaultSleepTime)
	}
	return nil
}
