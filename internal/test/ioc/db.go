package ioc

import (
	"context"
	"database/sql"
	"github.com/ecodeclub/ekit/retry"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"notification-platform/internal/repository/dao"
	"sync"
	"time"
)

func WaitForDBSetup(dsn string) {
	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	if err1 := ping(sqlDB); err1 != nil {
		panic(err1)
	}
}

func ping(sqlDB *sql.DB) error {
	const maxInterval = 10 * time.Second
	const maxRetries = 10
	strategy, err := retry.NewExponentialBackoffRetryStrategy(time.Second, maxInterval, maxRetries)
	if err != nil {
		return err
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
			panic("Ping DB 重试失败......")
		}
		time.Sleep(next)
	}
	return nil
}

var (
	db         *egorm.Component
	initDBOnce sync.Once
)

func InitDBAndTables() *egorm.Component {
	initDBOnce.Do(func() {
		if db != nil {
			return
		}
		econf.Set("mysql", map[string]any{
			"dsn":   "root:root@tcp(localhost:13316)/notification?collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s&multiStatements=true&interpolateParams=true&charset=utf8mb4",
			"debug": true,
		})
		WaitForDBSetup(econf.GetStringMapString("mysql")["dsn"])
		db = egorm.Load("mysql").Build()

		if err := dao.InitTables(db); err != nil {
			panic(err)
		}
	})

	return db
}

func InitDBWithCustomConnPool(cp gorm.ConnPool) *gorm.DB {
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn: cp,
	}), &gorm.Config{
		ConnPool: cp,
	})
	if err != nil {
		panic(err)
	}
	return gdb
}
