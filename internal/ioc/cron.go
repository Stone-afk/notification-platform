package ioc

import (
	"github.com/gotomicro/ego/task/ecron"
	"notification-platform/internal/repository"
	quota "notification-platform/internal/service/quota/task"
)

func Crons(q *quota.MonthlyResetCron, bCfg repository.BusinessConfigRepository) []ecron.Ecron {
	q1 := ecron.Load("cron.quotaMonthlyReset").Build(ecron.WithJob(q.Do))
	q2 := ecron.Load("cron.loadBusinessLocalCache").Build(ecron.WithJob(bCfg.LoadCache))
	return []ecron.Ecron{q1, q2}
}
