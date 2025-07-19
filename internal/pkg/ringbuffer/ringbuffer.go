package ringbuffer

import (
	"errors"
	"sync"
	"time"
)

// ErrInvalidCapacity 当创建环形缓冲区的容量小于等于0时返回
var ErrInvalidCapacity = errors.New("环形缓冲区容量必须大于0")

// TimeDurationRingBuffer 是一个固定容量、线程安全的环形缓冲区
// 用于保存最近 capacity 个 time.Duration 并能 O(1) 计算平均值。
type TimeDurationRingBuffer struct {
	mu       sync.RWMutex
	buffer   []time.Duration // 环形存储
	capacity int             // 固定容量
	index    int             // 下一个写入位置
	count    int             // 当前已写入数量 (<= capacity)
	sum      time.Duration   // buffer 中元素之和，便于 O(1) 求平均
}

// Add 向环形缓冲区追加一个样本 d
// 当缓冲区已满时会覆盖最老的数据，同时维护 sum
func (r *TimeDurationRingBuffer) Add(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == r.capacity {
		// 缓冲区已满，需要减去即将被覆盖的值
		r.sum -= r.buffer[r.index]
	} else {
		r.count++
	}
	r.buffer[r.index] = d
	r.sum += d
	r.index = (r.index + 1) % r.capacity
}

// Avg 返回当前缓冲区内样本的平均值
// 当尚无样本时返回 0
func (r *TimeDurationRingBuffer) Avg() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return 0
	}
	return r.sum / time.Duration(r.count)
}

// Len 返回当前已存样本数量
func (r *TimeDurationRingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}

// Cap 返回缓冲区容量
func (r *TimeDurationRingBuffer) Cap() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.capacity
}

// Reset 清空环形缓冲区，重置所有计数和总和
func (r *TimeDurationRingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 清空数据不是必须的，但这样可以释放内存
	for i := range r.buffer {
		r.buffer[i] = 0
	}
	r.index = 0
	r.count = 0
	r.sum = 0
}

// NewTimeDurationRingBuffer 创建一个容量为 capacity 的 TimeDurationRingBuffer。
// capacity 必须 >0，否则返回错误。
func NewTimeDurationRingBuffer(capacity int) (*TimeDurationRingBuffer, error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &TimeDurationRingBuffer{
		buffer:   make([]time.Duration, capacity),
		capacity: capacity,
	}, nil
}
