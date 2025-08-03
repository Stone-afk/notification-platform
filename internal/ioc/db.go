package ioc

import (
	"context"
	"database/sql"
	"github.com/ecodeclub/ekit/retry"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"notification-platform/internal/pkg/gormx/metrics"
	"notification-platform/internal/pkg/gormx/tracing"
	"notification-platform/internal/repository/dao"
	"time"
)

func InitDB() *egorm.Component {
	WaitForDBSetup(econf.GetString("mysql.dsn"))
	db := egorm.Load("mysql").Build()
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	// 这个是自己手搓的
	tracePlugin := tracing.NewGormTracingPlugin()
	metricsPlugin := metrics.NewGormMetricsPlugin()
	err = db.Use(tracePlugin)
	if err != nil {
		panic(err)
	}
	err = db.Use(metricsPlugin)
	if err != nil {
		panic(err)
	}
	return db
}

func WaitForDBSetup(dsn string) {
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	const maxInterval = 10 * time.Second
	const maxRetries = 10
	strategy, err := retry.NewExponentialBackoffRetryStrategy(time.Second, maxInterval, maxRetries)
	if err != nil {
		panic(err)
	}

	const timeout = 5 * time.Second
	for {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err = sqlDB.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
		next, ok := strategy.Next()
		if !ok {
			panic("WaitForDBSetup 重试失败......")
		}
		time.Sleep(next)
	}
}
