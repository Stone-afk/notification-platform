package loopjob

import "context"

// ResourceSemaphore 信号量，控制抢占资源的最大信号量
type ResourceSemaphore interface {
	Acquire(ctx context.Context) error
	Release(ctx context.Context) error
}
