package batchsize

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRingBufferAdjuster(t *testing.T) {
	tests := []struct {
		name        string
		initialSize int
		minSize     int
		maxSize     int
		adjustStep  int
		bufferSize  int
		wantSize    int
	}{
		{
			name:        "正常参数",
			initialSize: 100,
			minSize:     50,
			maxSize:     200,
			adjustStep:  5,
			bufferSize:  128,
			wantSize:    100,
		},
		{
			name:        "初始大小低于最小值",
			initialSize: 30,
			minSize:     50,
			maxSize:     200,
			adjustStep:  5,
			bufferSize:  128,
			wantSize:    50,
		},
		{
			name:        "初始大小高于最大值",
			initialSize: 250,
			minSize:     50,
			maxSize:     200,
			adjustStep:  5,
			bufferSize:  128,
			wantSize:    200,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adjuster := NewRingBufferAdjuster(
				tc.initialSize,
				tc.minSize,
				tc.maxSize,
				tc.adjustStep,
				10*time.Second,
				tc.bufferSize,
			)

			assert.NotNil(t, adjuster)
			size, err := adjuster.Adjust(t.Context(), 0) // 初始调用获取当前批大小
			assert.NoError(t, err)
			assert.Equal(t, tc.wantSize, size)
		})
	}
}

func TestRingBufferAdjuster_BatchSizeAdjustment(t *testing.T) {
	t.Run("批大小调整基本行为", func(t *testing.T) {
		// 创建调整器：初始大小100，最小50，最大200，步长5，冷却期短以便测试
		adjuster := NewRingBufferAdjuster(100, 50, 200, 5, 10*time.Millisecond, 3)

		// 初始化环形缓冲区
		// 填充相同的时间使平均值稳定在50ms
		for i := 0; i < 5; i++ {
			size, err := adjuster.Adjust(t.Context(), 50*time.Millisecond)
			assert.NoError(t, err)
			assert.Equal(t, 100, size, "初始化阶段应保持初始批大小")
		}

		// 响应时间变慢，批大小应减小
		size, err := adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 95, size, "响应时间增加时批大小应减小")

		// 冷却期内不应再调整
		size, err = adjuster.Adjust(t.Context(), 100*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 95, size, "冷却期内批大小不应改变")

		// 等待冷却期结束
		time.Sleep(20 * time.Millisecond)

		// 响应时间变快，批大小应增加
		size, err = adjuster.Adjust(t.Context(), 40*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 100, size, "响应时间减少时批大小应增加")
	})

	t.Run("批大小边界值测试", func(t *testing.T) {
		// 测试批大小下限
		minAdjuster := NewRingBufferAdjuster(55, 50, 200, 10, 10*time.Millisecond, 3)

		// 初始化环形缓冲区
		for i := 0; i < 5; i++ {
			size, err := minAdjuster.Adjust(t.Context(), 50*time.Millisecond)
			assert.NoError(t, err)
			assert.Equal(t, 55, size, "初始化阶段应保持初始批大小")
		}

		// 连续调整使批大小达到最小值
		currentSize := 55
		for i := 0; i < 10 && currentSize > 50; i++ {
			time.Sleep(15 * time.Millisecond)                                  // 等待冷却期
			size, err := minAdjuster.Adjust(t.Context(), 200*time.Millisecond) // 高响应时间
			assert.NoError(t, err)
			currentSize = size
		}

		// 再次尝试减小
		size, err := minAdjuster.Adjust(t.Context(), 200*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 50, size, "批大小不应低于最小值")

		// 测试批大小上限
		maxAdjuster := NewRingBufferAdjuster(195, 50, 200, 10, 10*time.Millisecond, 3)

		// 初始化环形缓冲区
		for i := 0; i < 5; i++ {
			size, err := maxAdjuster.Adjust(t.Context(), 100*time.Millisecond)
			assert.NoError(t, err)
			assert.Equal(t, 195, size, "初始化阶段应保持初始批大小")
		}

		// 连续调整使批大小达到最大值
		currentSize = 195
		for i := 0; i < 10 && currentSize < 200; i++ {
			time.Sleep(15 * time.Millisecond)                                 // 等待冷却期
			size, err := maxAdjuster.Adjust(t.Context(), 20*time.Millisecond) // 低响应时间
			assert.NoError(t, err)
			currentSize = size
		}

		// 再次尝试增加
		size, err = maxAdjuster.Adjust(t.Context(), 20*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 200, size, "批大小不应高于最大值")
	})
}

func TestRingBufferAdjuster_Behavior(t *testing.T) {
	// 创建一个冷却期明确的调整器
	adjuster := NewRingBufferAdjuster(100, 50, 200, 5, 50*time.Millisecond, 5)
	ctx := t.Context()

	// 第1阶段：初始化环形缓冲区 - 使用相同的响应时间
	for i := 0; i < 5; i++ {
		size, err := adjuster.Adjust(ctx, 50*time.Millisecond)
		assert.NoError(t, err)
		assert.Equal(t, 100, size, "初始化阶段应保持初始批大小")
	}

	// 第2阶段：单次高响应时间 - 应减小批大小
	size, err := adjuster.Adjust(ctx, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 95, size, "响应时间变慢时应减小批大小")

	// 在冷却期内 - 不应调整
	size, err = adjuster.Adjust(ctx, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 95, size, "冷却期内不应调整批大小")

	// 等待冷却期结束
	time.Sleep(60 * time.Millisecond)

	// 第3阶段：持续高响应时间 - 应进一步减小批大小
	size, err = adjuster.Adjust(ctx, 100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 90, size, "冷却期后持续高响应应继续减小批大小")

	// 等待冷却期结束
	time.Sleep(60 * time.Millisecond)

	// 第4阶段：响应时间变快 - 应增加批大小
	size, err = adjuster.Adjust(ctx, 30*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 95, size, "响应时间变快时应增加批大小")

	// 等待冷却期结束
	time.Sleep(60 * time.Millisecond)

	// 第5阶段：持续快速响应 - 应继续增加批大小
	size, err = adjuster.Adjust(ctx, 30*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 100, size, "持续快速响应应继续增加批大小")

	// 等待冷却期结束
	time.Sleep(60 * time.Millisecond)

	// 继续快速响应
	size, err = adjuster.Adjust(ctx, 30*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, 105, size, "持续快速响应应继续增加批大小")
}
