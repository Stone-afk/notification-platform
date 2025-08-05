package ioc

import "github.com/google/wire"

var BaseSet = wire.NewSet(InitDBAndTables, InitProviderEncryptKey, InitCache, InitMQ, InitRedis, InitRedisClient, InitDistributedLock)

// 在你还没有引入自己定义的 ID 生成算法之前，你用下面这个
// var BaseSet = wire.NewSet(InitDBAndTables, InitProviderEncryptKey, InitCache, InitMQ, InitIDGenerator, InitRedis, InitRedisClient, InitDistributedLock)
