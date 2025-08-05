package ioc

import (
	"time"

	"github.com/sony/sonyflake"
)

// 在引入自定义分库分表算法之前临时用的 ID 生成算法
// InitIDGenerator
func InitIDGenerator() *sonyflake.Sonyflake {
	// 使用固定设置的ID生成器
	return sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: time.Now(),
		MachineID: func() (uint16, error) {
			return 1, nil
		},
	})
}
