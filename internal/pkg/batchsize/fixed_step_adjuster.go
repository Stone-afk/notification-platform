package batchsize

import (
	"context"
	"time"
)

// FixedStepAdjuster 使用固定步长动态调整批处理大小
type FixedStepAdjuster struct {
	batchSize         int           // 当前批次大小
	adjustStep        int           // 每次调整的步长
	minBatchSize      int           // 最小批次大小
	maxBatchSize      int           // 最大批次大小
	lastAdjustTime    time.Time     // 上次调整时间
	minAdjustInterval time.Duration // 两次调整的最小间隔

	// 响应时间阈值
	fastThreshold time.Duration // 响应时间低于此值时增大批次
	slowThreshold time.Duration // 响应时间高于此值时减小批次
}

// Adjust 根据响应时间动态调整批次大小
func (b *FixedStepAdjuster) Adjust(_ context.Context, responseTime time.Duration) (int, error) {
	// 检查是否允许调整（满足最小间隔要求）
	if !b.lastAdjustTime.IsZero() && time.Since(b.lastAdjustTime) < b.minAdjustInterval {
		return b.batchSize, nil
	}
	// 根据响应时间调整批次大小
	if responseTime < b.fastThreshold {
		// 响应快，可以增加批次大小
		if b.batchSize < b.maxBatchSize {
			b.batchSize = min(b.batchSize+b.adjustStep, b.maxBatchSize)
			b.lastAdjustTime = time.Now()
		}
	} else if responseTime > b.slowThreshold {
		// 响应慢，需要减小批次大小
		if b.batchSize > b.minBatchSize {
			b.batchSize = max(b.batchSize-b.adjustStep, b.minBatchSize)
			b.lastAdjustTime = time.Now()
		}
	}
	// 响应时间在阈值之间，不调整
	return b.batchSize, nil
}

// NewFixedStepAdjuster 创建批次大小调整器
func NewFixedStepAdjuster(initialSize, minSize, maxSize, adjustStep int, minAdjustInterval, fastThreshold, slowThreshold time.Duration) *FixedStepAdjuster {
	if initialSize < minSize {
		initialSize = minSize
	}
	if initialSize > maxSize {
		initialSize = maxSize
	}

	return &FixedStepAdjuster{
		batchSize:         initialSize,
		adjustStep:        adjustStep,
		minBatchSize:      minSize,
		maxBatchSize:      maxSize,
		lastAdjustTime:    time.Time{}, // 使用零值时间，允许首次调用时就能调整
		minAdjustInterval: minAdjustInterval,
		fastThreshold:     fastThreshold,
		slowThreshold:     slowThreshold,
	}
}
