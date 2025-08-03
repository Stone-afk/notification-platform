package ioc

import (
	task "notification-platform/internal/service/notification/task"
	"notification-platform/internal/service/scheduler"
)

func InitTasks(t1 *task.AsyncRequestResultCallbackTask,
	t2 scheduler.NotificationScheduler,
	t3 *task.SendingTimeoutTask,
	t4 *task.TxCheckTask,
) []Task {
	return []Task{
		t1,
		t2,
		t3,
		t4,
	}
}
