package ioc

import (
	"time"

	"github.com/sony/sonyflake"
)

func InitIDGenerator() *sonyflake.Sonyflake {
	// 使用固定设置的ID生成器
	return sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: time.Now(),
		MachineID: func() (uint16, error) {
			return 1, nil
		},
	})
}
