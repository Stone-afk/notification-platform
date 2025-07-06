package scheduler

import "context"

// NotificationScheduler 通知调度服务接口
type NotificationScheduler interface {
	// Start 启动调度服务
	Start(ctx context.Context)
}
