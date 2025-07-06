package loopjob

import (
	"context"
	"notification-platform/internal/errs"
	"sync"
)

var _ ResourceSemaphore = (*MaxCntResourceSemaphore)(nil)

type MaxCntResourceSemaphore struct {
	maxCount int
	curCount int
	mu       *sync.RWMutex
}

func (r *MaxCntResourceSemaphore) Acquire(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.curCount >= r.maxCount {
		return errs.ErrExceedLimit
	}
	r.curCount++
	return nil
}

func (r *MaxCntResourceSemaphore) Release(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.curCount--
	return nil
}

func (r *MaxCntResourceSemaphore) UpdateMaxCount(maxCount int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.maxCount = maxCount
}

func NewResourceSemaphore(maxCount int) *MaxCntResourceSemaphore {
	return &MaxCntResourceSemaphore{
		maxCount: maxCount,
		mu:       &sync.RWMutex{},
		curCount: 0,
	}
}
