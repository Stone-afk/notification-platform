package sharding

import (
	"sync"

	"notification-platform/internal/pkg/sharding"

	"github.com/ecodeclub/ekit/syncx"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"notification-platform/internal/test/ioc"
)

const (
	testTableNum = 2
	testDBNum    = 2
)

func InitNotificationSharding() (notificationStrategy, callbacklogStrategy sharding.ShardingStrategy) {
	return sharding.NewShardingStrategy("notification", "notification", testTableNum, testDBNum), sharding.NewShardingStrategy("notification", "callback_log", testTableNum, testDBNum)
}

func InitTxnSharding() (notificationStrategy, txnStrategy sharding.ShardingStrategy) {
	return sharding.NewShardingStrategy("notification", "notification", testTableNum, testDBNum), sharding.NewShardingStrategy("notification", "tx_notification", testTableNum, testDBNum)
}

// Use a singleton pattern with sync.Once to prevent data races
var (
	once sync.Once
	dbs  *syncx.Map[string, *egorm.Component]
	mu   sync.Mutex
)

func InitDbs() *syncx.Map[string, *egorm.Component] {
	mu.Lock()
	defer mu.Unlock()

	once.Do(func() {
		dsn0 := "root:root@tcp(localhost:13316)/notification_0?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s&multiStatements=true&interpolateParams=true"
		ioc.WaitForDBSetup(dsn0)
		econf.Set("mysql0", map[string]any{
			"dsn":   dsn0,
			"debug": true,
		})

		db0 := egorm.Load("mysql0").Build()

		dsn1 := "root:root@tcp(localhost:13316)/notification_1?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s&multiStatements=true&interpolateParams=true"
		ioc.WaitForDBSetup(dsn1)
		econf.Set("mysql1", map[string]any{
			"dsn":   dsn1,
			"debug": true,
		})
		db1 := egorm.Load("mysql1").Build()
		dbs = &syncx.Map[string, *egorm.Component]{}
		dbs.Store("notification_0", db0)
		dbs.Store("notification_1", db1)
	})

	return dbs
}
