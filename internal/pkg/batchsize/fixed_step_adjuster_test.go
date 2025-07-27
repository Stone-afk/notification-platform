package batchsize

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFixedStepAdjuster(t *testing.T) {
	tests := []struct {
		name          string
		initialSize   int
		minSize       int
		maxSize       int
		adjustStep    int
		interval      time.Duration
		fastThreshold time.Duration
		slowThreshold time.Duration
		expectSize    int
	}{
		{
			name:          "正常初始化",
			initialSize:   50,
			minSize:       10,
			maxSize:       100,
			adjustStep:    5,
			interval:      time.Second * 10,
			fastThreshold: 150 * time.Millisecond,
			slowThreshold: 200 * time.Millisecond,
			expectSize:    50,
		},
		{
			name:          "初始值小于最小值",
			initialSize:   5,
			minSize:       10,
			maxSize:       100,
			adjustStep:    5,
			interval:      time.Second * 10,
			fastThreshold: 150 * time.Millisecond,
			slowThreshold: 200 * time.Millisecond,
			expectSize:    10, // 应该被调整为最小值
		},
		{
			name:          "初始值大于最大值",
			initialSize:   150,
			minSize:       10,
			maxSize:       100,
			adjustStep:    5,
			interval:      time.Second * 10,
			fastThreshold: 150 * time.Millisecond,
			slowThreshold: 200 * time.Millisecond,
			expectSize:    100, // 应该被调整为最大值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjuster := NewFixedStepAdjuster(tt.initialSize, tt.minSize, tt.maxSize, tt.adjustStep, tt.interval, tt.fastThreshold, tt.slowThreshold)

			// 仅测试公共行为：第一次调用 Adjust 应返回初始值
			// 使用中间值避免立即调整，175ms 在阈值之间
			firstSize, err := adjuster.Adjust(t.Context(), 175*time.Millisecond)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectSize, firstSize)
		})
	}
}

func TestAdjustBatchSize(t *testing.T) {
	t.Run("响应时间影响批次大小", func(t *testing.T) {
		t.Parallel()
		// 创建一个无间隔限制的调整器
		adjuster := NewFixedStepAdjuster(50, 10, 100, 10, 0, 150*time.Millisecond, 200*time.Millisecond)

		// 1. 响应时间在中间范围 - 保持不变
		size, err := adjuster.Adjust(t.Context(), 175*time.Millisecond) // 在阈值之间
		assert.NoError(t, err)
		assert.Equal(t, 50, size, "中间响应时间不应改变批次大小")

		// 2. 快速响应 - 增加批次大小
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond) // 低于快速阈值
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "快速响应应增加批次大小")

		// 3. 再次快速响应 - 继续增加
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 70, size, "连续快速响应应继续增加批次大小")

		// 4. 慢速响应 - 减少批次大小
		size, err = adjuster.Adjust(t.Context(), 250*time.Millisecond) // 高于慢速阈值
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "慢速响应应减少批次大小")
	})

	t.Run("批次大小有边界限制", func(t *testing.T) {
		t.Parallel()

		adjuster := NewFixedStepAdjuster(90, 10, 100, 20, 0, 150*time.Millisecond, 200*time.Millisecond)

		// 1. 接近最大值时快速响应
		size, err := adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 100, size, "应增加到但不超过最大值")

		// 2. 已达最大值时快速响应
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 100, size, "不应超过最大值")

		// 重新创建一个接近最小值的调整器
		adjuster = NewFixedStepAdjuster(30, 10, 100, 20, 0, 150*time.Millisecond, 200*time.Millisecond)

		// 3. 接近最小值时慢速响应
		size, err = adjuster.Adjust(t.Context(), 250*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 10, size, "应减少到但不低于最小值")

		// 4. 已达最小值时慢速响应
		size, err = adjuster.Adjust(t.Context(), 250*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 10, size, "不应低于最小值")
	})

	t.Run("调整间隔限制", func(t *testing.T) {
		// 创建一个有间隔限制的调整器
		adjuster := NewFixedStepAdjuster(50, 10, 100, 10, time.Millisecond*100, 150*time.Millisecond, 200*time.Millisecond)

		// 1. 首次调用应正常调整
		size, err := adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "首次调用应正常调整")

		// 2. 紧接着的调用不应调整
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 60, size, "间隔内不应调整")

		// 3. 等待间隔后调用应正常调整
		time.Sleep(time.Millisecond * 150)
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 70, size, "等待足够间隔后应可调整")
	})
}

func TestContinuousAdjustment(t *testing.T) {
	t.Run("连续调整行为", func(t *testing.T) {
		// 初始化一个间隔设为0的调整器
		adjuster := NewFixedStepAdjuster(50, 10, 100, 10, 0, 150*time.Millisecond, 200*time.Millisecond)

		// 初始大小验证
		initialSize, _ := adjuster.Adjust(t.Context(), 175*time.Millisecond) // 不触发调整的中间值
		assert.Equal(t, 50, initialSize)

		// 连续增长直到上限
		sizes := []int{60, 70, 80, 90, 100, 100}
		for _, expected := range sizes {
			size, err := adjuster.Adjust(t.Context(), 100*time.Millisecond) // 快速响应
			assert.NoError(t, err)
			assert.Equal(t, expected, size)
		}

		// 连续减少直到下限
		sizes = []int{90, 80, 70, 60, 50, 40, 30, 20, 10, 10}
		for _, expected := range sizes {
			size, err := adjuster.Adjust(t.Context(), 250*time.Millisecond) // 慢速响应
			assert.NoError(t, err)
			assert.Equal(t, expected, size)
		}
	})
}
