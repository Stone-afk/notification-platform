package id

import (
	"sync/atomic"
	"time"

	"notification-platform/internal/pkg/hash"
)

const (
	// 位数分配常量
	timestampBits = 41 // 时间戳位数
	hashBits      = 10 // hash值位数
	sequenceBits  = 12 // 序列号位数

	// 位移常量
	sequenceShift  = 0
	hashShift      = sequenceBits
	timestampShift = hashBits + sequenceBits

	// 掩码常量
	sequenceMask  = (1 << sequenceBits) - 1
	hashMask      = (1 << hashBits) - 1
	timestampMask = (1 << timestampBits) - 1

	// 基准时间 - 2024年1月1日，可以根据实际需求调整
	epochMillis   = int64(1704067200000) // 2024-01-01 00:00:00 UTC in milliseconds
	number        = int64(1024)
	number1000    = int64(1000)
	number1000000 = int64(1000000)
)

// Generator 是ID生成器结构
type Generator struct {
	sequence int64     // 序列号计数器，使用原子操作访问
	lastTime int64     // 上次生成ID的时间戳
	epoch    time.Time // 基准时间点
}

// NewGenerator 创建一个新的ID生成器
func NewGenerator() *Generator {
	return &Generator{
		sequence: 0,
		lastTime: 0,
		epoch:    time.Unix(epochMillis/number1000, (epochMillis%number1000)*number1000000),
	}
}

// GenerateID 根据雪花算法变种生成ID
// bizId: 业务ID
// key: 业务关键字
func (g *Generator) GenerateID(bizID int64, key string) int64 {
	timestamp := time.Now().UnixMilli() - epochMillis
	// 计算hash值并取余
	hashValue := hash.Hash(bizID, key)
	// 使用原子操作安全地递增序列号
	sequence := atomic.AddInt64(&g.sequence, 1) - 1 // 减1是因为AddInt64返回递增后的值

	// 组装最终ID
	id := (timestamp&timestampMask)<<timestampShift | // 时间戳部分
		(hashValue&hashMask)<<hashShift | // hash值部分
		(sequence & sequenceMask) // 序列号部分
	return id
}

// ExtractTimestamp 从ID中提取时间戳
func ExtractTimestamp(id int64) time.Time {
	timestamp := (id >> timestampShift) & timestampMask
	return time.Unix(0, (timestamp+epochMillis)*int64(time.Millisecond))
}

// ExtractHashValue 从ID中提取hash值
func ExtractHashValue(id int64) int64 {
	return (id >> hashShift) & hashMask
}

// ExtractSequence 从ID中提取序列号部分
func ExtractSequence(id int64) int64 {
	return id & sequenceMask
}
