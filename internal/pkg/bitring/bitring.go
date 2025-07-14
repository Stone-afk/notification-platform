package bitring

import "sync"

const (
	// bitsPerWord 表示一个uint64的位数
	bitsPerWord = 64
	// bitsMask 用于位操作的掩码(63 = 0x3f)
	bitsMask = bitsPerWord - 1
	// bitsShift 用于计算位的偏移量(6 = log2(64))
	bitsShift = 6

	// 默认值
	defaultSize        = 128
	defaultConsecutive = 3
)

// BitRing 以位(bit)方式记录事件的滑动窗口。
type BitRing struct {
	words       []uint64     // 实际存储
	size        int          // 窗口长度
	pos         int          // 下一写入位置
	filled      bool         // 标记环形缓冲区是否已满（完成一轮循环）
	eventCount  int          // 当前窗口内事件发生数
	threshold   float64      // 事件发生率阈值
	consecutive int          // 连续事件触发次数
	mu          sync.RWMutex // 保证并发安全
}

// Add 记录一次结果；eventHappened=true 表示事件发生
func (br *BitRing) Add(eventHappened bool) {
	br.mu.Lock()
	defer br.mu.Unlock()

	oldBit := br.bitAt(br.pos)

	// 更新计数
	if br.filled && oldBit {
		br.eventCount--
	}
	br.setBit(br.pos, eventHappened)
	if eventHappened {
		br.eventCount++
	}

	// 前进指针
	br.pos++
	if br.pos == br.size {
		br.pos = 0
		br.filled = true
	}
}

// IsConditionMet 判断是否满足触发条件
// 当连续事件次数达到阈值或事件发生率超过阈值时返回true
func (br *BitRing) IsConditionMet() bool {
	br.mu.RLock()
	defer br.mu.RUnlock()

	window := br.windowSize()
	if window == 0 {
		return false
	}

	// 1. 连续发生事件 consecutive 次
	if window >= br.consecutive {
		allEvents := true
		for i := 1; i <= br.consecutive; i++ {
			pos := (br.pos - i + br.size) % br.size
			if !br.bitAt(pos) {
				allEvents = false
				break
			}
		}
		if allEvents {
			return true
		}
	}

	// 2. 事件发生率超过阈值
	if float64(br.eventCount)/float64(window) > br.threshold {
		return true
	}
	return false
}

// windowSize 返回当前有效的窗口大小
func (br *BitRing) windowSize() int {
	if br.filled {
		return br.size
	}
	return br.pos
}

// bitAt 获取指定位置的bit值
func (br *BitRing) bitAt(idx int) bool {
	word := idx >> bitsShift    // 等价于 idx / 64
	off := uint(idx & bitsMask) // 等价于 idx % 64
	return (br.words[word]>>off)&1 == 1
}

// setBit 设置指定位置的bit值
func (br *BitRing) setBit(idx int, v bool) {
	word := idx >> bitsShift
	off := uint(idx & bitsMask)
	if v {
		br.words[word] |= 1 << off
	} else {
		br.words[word] &^= 1 << off
	}
}

// NewBitRing 创建一个新的BitRing
// size: 滑动窗口大小
// threshold: 事件发生率阈值(0.0-1.0)，超过此阈值将触发IsConditionMet返回true
// consecutive: 连续出现多少次事件将触发IsConditionMet返回true
func NewBitRing(size int, threshold float64, consecutive int) *BitRing {
	if size <= 0 {
		size = defaultSize
	}
	if consecutive <= 0 {
		consecutive = defaultConsecutive
	}
	if consecutive > size { // 安全起见
		consecutive = size
	}
	// 确保阈值在有效范围内
	if threshold < 0 {
		threshold = 0
	} else if threshold > 1 {
		threshold = 1
	}

	return &BitRing{
		words:       make([]uint64, (size+bitsMask)/bitsPerWord),
		size:        size,
		threshold:   threshold,
		consecutive: consecutive,
	}
}
