package batchsize

import (
	"context"
	"notification-platform/internal/pkg/ringbuffer"
	"sync"
	"time"
)

// RingBufferAdjuster 基于过去执行时间维护的环形缓冲区动态调整批大小
// 当前执行时间超过历史平均值时减少批大小，低于平均值时增加批大小
type RingBufferAdjuster struct {
	mutex          *sync.RWMutex
	timeBuffer     *ringbuffer.TimeDurationRingBuffer // 历史执行时间的环形缓冲区
	batchSize      int                                // 当前批大小
	minBatchSize   int                                // 最小批大小
	maxBatchSize   int                                // 最大批大小
	adjustStep     int                                // 每次调整的步长（增加或减少）
	cooldownPeriod time.Duration                      // 调整后的冷却期
	lastAdjustTime time.Time                          // 上次调整时间
}

// Adjust 根据响应时间动态调整批次大小
// 1. 记录响应时间到环形缓冲区
// 2. 如果当前时间比平均时间长，且不在冷却期，则减少批大小
// 3. 如果当前时间比平均时间短，且不在冷却期，则增加批大小
func (a *RingBufferAdjuster) Adjust(_ context.Context, responseTime time.Duration) (int, error) {
	// 思来想去，我觉得作为一个放在 pkg 里面的通用实现，还是得自己加锁保护线程安全
	a.mutex.Lock()
	defer a.mutex.Unlock()
	// 记录当前响应时间到环形缓冲区
	a.timeBuffer.Add(responseTime)

	// 需要至少收集一轮数据才能开始调整
	if a.timeBuffer.Len() < a.timeBuffer.Cap() {
		return a.batchSize, nil
	}

	// 如果处于冷却期内，不调整批大小
	if !a.lastAdjustTime.IsZero() && time.Since(a.lastAdjustTime) < a.cooldownPeriod {
		return a.batchSize, nil
	}

	// 获取平均执行时间
	avgTime := a.timeBuffer.Avg()
	// 根据响应时间调整批大小
	if responseTime > avgTime {
		// 响应时间高于平均值，减小批大小
		if a.batchSize > a.minBatchSize {
			a.batchSize = max(a.batchSize-a.adjustStep, a.minBatchSize)
			a.lastAdjustTime = time.Now() // 更新调整时间戳
		}
	} else if responseTime < avgTime {
		// 响应时间低于平均值，增加批大小
		if a.batchSize < a.maxBatchSize {
			a.batchSize = min(a.batchSize+a.adjustStep, a.maxBatchSize)
			a.lastAdjustTime = time.Now() // 更新调整时间戳
		}
	}
	// 响应时间等于平均值，不调整
	return a.batchSize, nil
}

// NewRingBufferAdjuster 创建基于环形缓冲区的批大小调整器
// initialSize: 初始批大小(基准值)
// minSize: 最小允许的批大小
// maxSize: 最大允许的批大小
// adjustStep: 每次调整的步长（增加或减少）
// cooldownPeriod: 调整后的冷却时间
// bufferSize: 环形缓冲区大小(历史执行时间数量)
func NewRingBufferAdjuster(initialSize, minSize, maxSize, adjustStep int,
	cooldownPeriod time.Duration, bufferSize int,
) *RingBufferAdjuster {
	if initialSize < minSize {
		initialSize = minSize
	} else if initialSize > maxSize {
		initialSize = maxSize
	}

	if bufferSize <= 0 {
		bufferSize = 128 // 默认维护128条历史记录
	}

	timeBuffer, _ := ringbuffer.NewTimeDurationRingBuffer(bufferSize)

	return &RingBufferAdjuster{
		timeBuffer:     timeBuffer,
		batchSize:      initialSize,
		minBatchSize:   minSize,
		maxBatchSize:   maxSize,
		adjustStep:     adjustStep,
		cooldownPeriod: cooldownPeriod,
		lastAdjustTime: time.Time{}, // 零值时间，初始允许立即调整
		mutex:          &sync.RWMutex{},
	}
}
