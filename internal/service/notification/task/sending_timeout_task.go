package task

import (
	"context"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/pkg/loopjob"
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
