package loopjob

import (
	"context"
	"errors"
	"fmt"
	"github.com/gotomicro/ego/core/elog"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/pkg/sharding"
	"time"
)

type CtxKey string

type ShardingLoopJobOption func(*ShardingLoopJob)

type ShardingLoopJob struct {
	shardingStrategy  sharding.ShardingStrategy
	baseKey           string // 业务标识
	dclient           dlock.Client
	logger            *elog.Component
	biz               func(ctx context.Context) error
	retryInterval     time.Duration
	defaultTimeout    time.Duration
	resourceSemaphore ResourceSemaphore
}

func (l *ShardingLoopJob) Run(ctx context.Context) {
	for {
		for _, dst := range l.shardingStrategy.Broadcast() {
			// 超过允许抢占的上限了
			err := l.resourceSemaphore.Acquire(ctx)
			if err != nil {
				time.Sleep(l.retryInterval)
				continue
			}

			key := l.generateKey(dst.DB, dst.Table)
			lock, err := l.dclient.NewLock(ctx, key, l.retryInterval)
			if err != nil {
				l.logger.Error("初始化分布式锁失败，重试",
					elog.Any("err", err))
				err = l.resourceSemaphore.Release(ctx)
				if err != nil {
					l.logger.Error("释放表的信号量失败", elog.FieldErr(err))
				}
				continue
			}
			lockCtx, cancel := context.WithTimeout(ctx, l.defaultTimeout)
			// 抢锁
			err = lock.Lock(lockCtx)
			cancel()
			// 没有拿到锁，不管是系统错误，还是锁被人持有，都没有关系
			// 暂停一段时间之后继续
			if err != nil {
				l.logger.Error("没有抢到分布式锁，系统出现问题", elog.Any("err", err))
				err = l.resourceSemaphore.Release(ctx)
				if err != nil {
					l.logger.Error("释放表的信号量失败", elog.FieldErr(err))
				}
				continue
			}
			// 抢占成功了
			go l.tableLoop(sharding.CtxWithDst(ctx, dst), lock)
		}
	}
}

func (l *ShardingLoopJob) generateKey(db, tab string) string {
	return fmt.Sprintf("%s:%s:%s", l.baseKey, db, tab)
}

func (l *ShardingLoopJob) tableLoop(ctx context.Context, lock dlock.Lock) {
	defer func() {
		_ = l.resourceSemaphore.Release(ctx)
	}()
	// 在这里执行业务
	err := l.bizLoop(ctx, lock)
	// 要么是续约失败，要么是 ctx 本身已经过期了
	if err != nil {
		l.logger.Error("执行业务失败，将执行重试", elog.FieldErr(err))
	}
	// 不管是什么原因，都要考虑释放分布式锁了
	// 要稍微摆脱 ctx 的控制，因为此时 ctx 可能被取消了
	unCtx, cancel := context.WithTimeout(context.Background(), l.defaultTimeout)
	//nolint:contextcheck // 这里必须使用 Background Context，因为原始 ctx 可能已被取消，但仍需尝试解锁操作。
	unErr := lock.Unlock(unCtx)
	cancel()
	if unErr != nil {
		l.logger.Error("释放分布式锁失败", elog.Any("err", unErr))
	}
	err = ctx.Err()
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// 被取消，那么就要跳出循环
		l.logger.Info("任务被取消，退出任务循环")
		return
	default:
		// 不可挽回的错误，后续考虑回去
		l.logger.Error("执行任务失败，将执行重试")
		time.Sleep(l.retryInterval)
	}
}

func (l *ShardingLoopJob) bizLoop(ctx context.Context, lock dlock.Lock) error {
	const bizTimeout = 50 * time.Second
	for {
		// 可以确保业务在分布式锁过期之前结束
		bizCtx, cancel := context.WithTimeout(ctx, bizTimeout)
		err := l.biz(bizCtx)
		cancel()
		if err != nil {
			l.logger.Error("业务执行失败", elog.FieldErr(err))
		}
		if ctx.Err() != nil {
			// 要中断这个循环了
			return ctx.Err()
		}
		refCtx, cancel := context.WithTimeout(ctx, l.defaultTimeout)
		err = lock.Refresh(refCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("分布式锁续约失败 %w", err)
		}
	}
}

func NewShardingLoopJob(
	dclient dlock.Client,
	baseKey string,
	// 你要执行的业务。注意当 ctx 被取消的时候，就会退出全部循环
	biz func(ctx context.Context) error,
	shardingStrategy sharding.ShardingStrategy,
	resourceSemaphore ResourceSemaphore,
) *ShardingLoopJob {
	const defaultTimeout = 3 * time.Second
	return newShardingLoopJobLoop(dclient, baseKey, biz, shardingStrategy, time.Minute, defaultTimeout, resourceSemaphore)
}

// ShardingLoopJobLoop 用于创建一个ShardingLoopJobLoop实例，允许指定重试间隔，便于测试
func newShardingLoopJobLoop(
	dclient dlock.Client,
	baseKey string,
	biz func(ctx context.Context) error,
	shardingStrategy sharding.ShardingStrategy,
	retryInterval time.Duration,
	defaultTimeout time.Duration,
	resourceSemaphore ResourceSemaphore,
) *ShardingLoopJob {
	return &ShardingLoopJob{
		dclient:           dclient,
		shardingStrategy:  shardingStrategy,
		baseKey:           baseKey,
		logger:            elog.DefaultLogger,
		biz:               biz,
		retryInterval:     retryInterval,
		resourceSemaphore: resourceSemaphore,
		defaultTimeout:    defaultTimeout,
	}
}
