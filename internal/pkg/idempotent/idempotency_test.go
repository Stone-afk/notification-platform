package idempotent

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// IdempotencyServiceTest 定义了一组测试IdempotencyService接口的测试用例
type IdempotencyServiceTest struct {
	Name string
	// 创建一个新的IdempotencyService实例
	NewService func() (IdempotencyService, func(), error)
}

// RunTests 运行所有测试用例
func (ist IdempotencyServiceTest) RunTests(t *testing.T) {
	t.Helper()
	t.Run("TestExists", func(t *testing.T) {
		ist.TestExists(t)
	})
	t.Run("TestMExists", func(t *testing.T) {
		ist.TestMExists(t)
	})
}

// TestExists 测试单个键的幂等性检查
func (ist IdempotencyServiceTest) TestExists(t *testing.T) {
	t.Parallel()
	service, cleanup, err := ist.NewService()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	ctx := t.Context()

	// 第一次调用，键不应该存在
	key := "test-key-1"
	exists, err := service.Exists(ctx, key)
	require.NoError(t, err)
	assert.False(t, exists, "Key should not exist initially")

	// 再次调用同一个键，应该存在
	exists, err = service.Exists(ctx, key)
	require.NoError(t, err)
	assert.True(t, exists, "Key should exist after first call")

	// 使用不同的键，应该不存在
	newKey := "test-key-2"
	exists, err = service.Exists(ctx, newKey)
	require.NoError(t, err)
	assert.False(t, exists, "New key should not exist")
}

// TestMExists 测试多个键的幂等性检查
func (ist IdempotencyServiceTest) TestMExists(t *testing.T) {
	t.Parallel()
	service, cleanup, err := ist.NewService()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	ctx := t.Context()

	// 首次检查多个键
	keys := []string{"batch-key-1", "batch-key-2", "batch-key-3"}
	exists, err := service.MExists(ctx, keys...)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(exists), "Results length should match keys length")

	for i, res := range exists {
		assert.False(t, res, "Key %s should not exist initially", keys[i])
	}

	// 再次检查同一批键
	exists, err = service.MExists(ctx, keys...)
	require.NoError(t, err)
	for i, res := range exists {
		assert.True(t, res, "Key %s should exist after first call", keys[i])
	}
}

// TestRedisImplementation 测试Redis实现
func TestRedisImplementation(t *testing.T) {
	t.Parallel()
	t.Skip()
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		client.Close()
	})

	// 检查Redis连接
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	t.Cleanup(cancel)

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis server is not available, skipping test")
		return
	}

	// 先清空所有测试相关的key
	err := client.FlushDB(ctx).Err()
	if err != nil {
		t.Skip("Failed to flush Redis DB, skipping test: " + err.Error())
		return
	}

	// 创建测试套件
	redisTest := IdempotencyServiceTest{
		Name: "RedisIdempotencyService",
		NewService: func() (IdempotencyService, func(), error) {
			// 清理测试键前缀
			cleanup := func() {
				ctx := t.Context()
				iter := client.Scan(ctx, 0, "idempotency:*", 100).Iterator()
				for iter.Next(ctx) {
					client.Del(ctx, iter.Val())
				}
			}

			cleanup()

			// 创建服务实例
			service := NewRedisIdempotencyService(client, 10*time.Minute)
			return service, cleanup, nil
		},
	}

	// 运行所有测试
	redisTest.RunTests(t)
}

// TestBloomFilterImplementation 测试BloomFilter实现
func TestBloomFilterImplementation(t *testing.T) {
	t.Skip()
	t.Parallel()
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		client.Close()
	})

	// 检查Redis连接
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	t.Cleanup(cancel)

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis server is not available, skipping test")
		return
	}

	// 检查是否支持布隆过滤器
	_, err := client.Do(ctx, "MODULE", "LIST").Result()
	if err != nil {
		t.Skip("Failed to check Redis modules, skipping test: " + err.Error())
		return
	}

	// 设置布隆过滤器名称和参数
	filterName := "test-bloom-filter"
	capacity := uint64(10000)
	errorRate := 0.01

	// 尝试创建布隆过滤器
	err = client.Del(ctx, filterName).Err()
	if err != nil {
		t.Skip("Failed to clean up previous bloom filter, skipping test: " + err.Error())
		return
	}

	_, err = client.Do(ctx, "BF.RESERVE", filterName, errorRate, capacity).Result()
	if err != nil {
		t.Skip("BloomFilter module is not available in Redis, skipping test: " + err.Error())
		return
	}

	// 创建测试套件
	bloomTest := IdempotencyServiceTest{
		Name: "BloomIdempotencyService",
		NewService: func() (IdempotencyService, func(), error) {
			// 清理函数
			afterFunc := func() {
				ctx := t.Context()
				// 删除布隆过滤器
				client.Del(ctx, filterName)
				client.Do(ctx, "BF.RESERVE", filterName, errorRate, capacity)
			}

			// 先执行一次清理
			afterFunc()

			// 创建服务实例
			service := NewBloomService(client, filterName, capacity, errorRate)
			return service, afterFunc, nil
		},
	}

	// 运行所有测试
	bloomTest.RunTests(t)
}

// TestRedisMixImplementation 测试混合策略实现
func TestRedisMixImplementation(t *testing.T) {
	t.Parallel()
	capacity := uint64(10000)
	errorRate := 0.01
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	client.Do(t.Context(), "BF.RESERVE", "mix_filter", errorRate, capacity)
	ctx := t.Context()
	require.NoError(t, client.Ping(ctx).Err())

	mixTest := IdempotencyServiceTest{
		Name: "RedisMix",
		NewService: func() (IdempotencyService, func(), error) {
			return NewRedisMix(client), func() {}, nil
		},
	}

	t.Run("全不存在场景", func(t *testing.T) {
		t.Parallel()
		svc, cleanup, _ := mixTest.NewService()
		t.Cleanup(cleanup)

		res, err := svc.MExists(ctx, "key1", "key2")
		require.NoError(t, err)
		assert.Equal(t, []bool{false, false}, res)
	})

	t.Run("混合存在场景", func(t *testing.T) {
		t.Parallel()
		svc, cleanup, _ := mixTest.NewService()
		t.Cleanup(cleanup)

		// 预置部分key在bloom
		client.BFAdd(ctx, "mix_filter", "vkey2")
		client.BFAdd(ctx, "mix_filter", "vkey4")

		client.SetNX(ctx, "mixIdempotency:vkey2", "xxxx", 100*time.Second)

		res, err := svc.MExists(ctx, "vkey1", "vkey2", "vkey3", "vkey4")
		require.NoError(t, err)
		assert.Equal(t, []bool{false, true, false, false}, res)
		client.Del(ctx, "mixIdempotency:vkey1", "mixIdempotency:vkey2", "mixIdempotency:vkey3", "mixIdempotency:vkey4")
	})

	t.Run("顺序一致性验证", func(t *testing.T) {
		t.Parallel()
		svc, _, _ := mixTest.NewService()

		keys := []string{"z", "a", "m"}
		_, err := svc.MExists(ctx, keys...) // 首次调用生成记录
		require.NoError(t, err)

		res, err := svc.MExists(ctx, keys...)
		require.NoError(t, err)
		assert.Equal(t, []bool{true, true, true}, res)
		client.Del(ctx, "mixIdempotency:z", "mixIdempotency:a", "mixIdempotency:m")
	})
}
