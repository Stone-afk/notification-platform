package ringbuffer

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestNewTimeDurationRingBuffer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		capacity int
		wantErr  bool
	}{
		{
			name:     "有效容量",
			capacity: 10,
			wantErr:  false,
		},
		{
			name:     "零容量",
			capacity: 0,
			wantErr:  true,
		},
		{
			name:     "负容量",
			capacity: -5,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rb, err := NewTimeDurationRingBuffer(tc.capacity)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rb)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rb)
				assert.Equal(t, tc.capacity, rb.Cap())
				assert.Equal(t, 0, rb.Len())
			}
		})
	}
}

func TestTimeDurationRingBuffer_AddAndAvg(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		capacity int
		adds     []time.Duration
		wantAvg  time.Duration
		wantLen  int
	}{
		{
			name:     "空缓冲区",
			capacity: 5,
			adds:     []time.Duration{},
			wantAvg:  0,
			wantLen:  0,
		},
		{
			name:     "小于容量",
			capacity: 4,
			adds: []time.Duration{
				time.Millisecond,
				2 * time.Millisecond,
			},
			wantAvg: 1500 * time.Microsecond, // (1ms + 2ms)/2
			wantLen: 2,
		},
		{
			name:     "等于容量",
			capacity: 3,
			adds: []time.Duration{
				time.Second,
				2 * time.Second,
				3 * time.Second,
			},
			wantAvg: 2 * time.Second,
			wantLen: 3,
		},
		{
			name:     "覆盖最旧值",
			capacity: 2,
			adds: []time.Duration{
				time.Second,
				3 * time.Second,
				5 * time.Second, // 覆盖 1s，留下 3s、5s => avg=4s
			},
			wantAvg: 4 * time.Second,
			wantLen: 2,
		},
		{
			name:     "多次覆盖",
			capacity: 3,
			adds: []time.Duration{
				time.Second,
				2 * time.Second,
				3 * time.Second,
				4 * time.Second, // 覆盖 1s
				5 * time.Second, // 覆盖 2s
				6 * time.Second, // 覆盖 3s
			},
			wantAvg: 5 * time.Second, // (4s+5s+6s)/3
			wantLen: 3,
		},
		{
			name:     "零值",
			capacity: 2,
			adds: []time.Duration{
				0,
				0,
			},
			wantAvg: 0,
			wantLen: 2,
		},
		{
			name:     "异常大值",
			capacity: 2,
			adds: []time.Duration{
				time.Duration(1<<62 - 1),
				time.Duration(1<<62 - 1),
			},
			wantAvg: time.Duration(1<<62 - 1),
			wantLen: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rb, err := NewTimeDurationRingBuffer(tc.capacity)
			require.NoError(t, err)
			for _, d := range tc.adds {
				rb.Add(d)
			}
			assert.Equal(t, tc.wantLen, rb.Len())
			assert.Equal(t, tc.wantAvg, rb.Avg())
		})
	}
}

func TestTimeDurationRingBuffer_Reset(t *testing.T) {
	t.Parallel()
	rb, err := NewTimeDurationRingBuffer(5)
	require.NoError(t, err)

	rb.Add(time.Second)
	rb.Add(2 * time.Second)
	assert.Equal(t, 2, rb.Len())
	assert.Equal(t, 1500*time.Millisecond, rb.Avg())

	rb.Reset()
	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, time.Duration(0), rb.Avg())

	// 重置后继续添加
	rb.Add(3 * time.Second)
	assert.Equal(t, 1, rb.Len())
	assert.Equal(t, 3*time.Second, rb.Avg())
}

func TestTimeDurationRingBuffer_ConcurrentSafety(t *testing.T) {
	t.Parallel()
	const (
		capacity   = 128
		concurrent = 64
		loops      = 1_000
	)
	rb, err := NewTimeDurationRingBuffer(capacity)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(concurrent)
	for i := 0; i < concurrent; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < loops; j++ {
				rb.Add(time.Duration(j) * time.Millisecond)
				_ = rb.Avg()
			}
		}()
	}
	wg.Wait()

	// 最终长度应始终为 capacity
	assert.Equal(t, capacity, rb.Len())
}

func TestTimeDurationRingBuffer_CapAndLen(t *testing.T) {
	t.Parallel()
	capacity := 10
	rb, err := NewTimeDurationRingBuffer(capacity)
	require.NoError(t, err)

	assert.Equal(t, capacity, rb.Cap())
	assert.Equal(t, 0, rb.Len())

	rb.Add(time.Second)
	rb.Add(2 * time.Second)

	assert.Equal(t, capacity, rb.Cap())
	assert.Equal(t, 2, rb.Len())

	for i := 0; i < capacity; i++ {
		rb.Add(time.Duration(i) * time.Second)
	}

	assert.Equal(t, capacity, rb.Cap())
	assert.Equal(t, capacity, rb.Len())
}
