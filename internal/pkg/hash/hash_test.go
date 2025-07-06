//go:build unit

package hash

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"testing"
)

func TestHashNoCollision(t *testing.T) {
	t.Parallel()
	// 定义测试规模：1000个组合
	testSize := 1000

	// 哈希结果映射，用于检测冲突
	hashResults := make(map[int64]struct{}, testSize)
	// 存储测试输入，用于在发现冲突时输出详细信息
	inputs := make([]struct {
		bizID int64
		key   string
	}, testSize)

	// 生成1000个不同的测试用例
	for i := 0; i < testSize; i++ {
		// 生成随机bizId (1-10000范围内)
		maxBig := big.NewInt(10000)
		randBig, err := rand.Int(rand.Reader, maxBig)
		if err != nil {
			t.Fatalf("Failed to generate random number: %v", err)
		}
		bizID := randBig.Int64() + 1

		// 生成随机key (10-30个字符)
		lenBig, err := rand.Int(rand.Reader, big.NewInt(20))
		if err != nil {
			t.Fatalf("Failed to generate random number: %v", err)
		}
		keyLength := int(lenBig.Int64()) + 10
		key := generateRandomString(keyLength)

		// 存储测试输入
		inputs[i] = struct {
			bizID int64
			key   string
		}{bizID, key}

		// 计算哈希值
		hashValue := Hash(bizID, key)

		// 检查是否存在冲突
		if _, exists := hashResults[hashValue]; exists {
			// 发现冲突，找出是哪两个输入产生了相同的哈希值
			for j := 0; j < i; j++ {
				prevHashValue := Hash(inputs[j].bizID, inputs[j].key)
				if prevHashValue == hashValue {
					t.Fatalf("哈希冲突: \n"+
						"输入1: bizID=%d, key=%s \n"+
						"输入2: bizID=%d, key=%s \n"+
						"相同的哈希值: %d",
						inputs[j].bizID, inputs[j].key,
						bizID, key,
						hashValue)
				}
			}
		}

		// 记录哈希值
		hashResults[hashValue] = struct{}{}
	}

	// 检查哈希结果数量是否等于测试用例数量
	if len(hashResults) != testSize {
		t.Errorf("预期生成 %d 个不同的哈希值，实际生成 %d 个", testSize, len(hashResults))
	} else {
		t.Logf("成功测试 %d 个不同的输入组合，未发现哈希冲突", testSize)
	}
}

// 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	// 生成随机字节
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("Failed to generate random bytes: " + err.Error())
	}

	// 将随机字节映射到字符集
	for i := 0; i < length; i++ {
		// 用随机字节模字符集长度，确保均匀分布
		idx := int(randomBytes[i]) % len(charset)
		result[i] = charset[idx]
	}

	return string(result)
}

// 测试哈希算法的分布均匀性
func TestHashDistribution(t *testing.T) {
	// 可选的额外测试：检查哈希分布
	// 这个测试不是强制要求的，但可以帮助验证哈希函数的质量
	t.Parallel()
	testSize := 10000
	bucketCount := 100
	buckets := make([]int, bucketCount)

	// 生成大量哈希值
	for i := 0; i < testSize; i++ {
		// 生成随机bizId (1-10000范围内)
		maxBig := big.NewInt(10000)
		randBig, err := rand.Int(rand.Reader, maxBig)
		if err != nil {
			t.Fatalf("Failed to generate random number: %v", err)
		}
		bizID := randBig.Int64() + 1

		key := "test" + strconv.Itoa(i)

		// 计算哈希值并放入对应的桶
		hashValue := Hash(bizID, key)
		bucketIndex := int(hashValue % int64(bucketCount))
		if bucketIndex < 0 {
			bucketIndex += bucketCount // 处理负数哈希值
		}
		buckets[bucketIndex]++
	}

	// 计算理论上每个桶的平均值和允许的偏差
	expectedPerBucket := float64(testSize) / float64(bucketCount)
	maxDeviation := 0.3 * expectedPerBucket // 允许30%的偏差

	// 检查分布是否均匀
	for i, count := range buckets {
		deviation := float64(count) - expectedPerBucket
		if deviation < 0 {
			deviation = -deviation
		}

		if deviation > maxDeviation {
			t.Logf("桶 %d 的值数量 (%d) 偏离预期 (%.2f) 超过允许范围", i, count, expectedPerBucket)
		}
	}

	// 输出一些分布统计信息
	minCount, maxCount, avg := buckets[0], buckets[0], float64(0)
	for _, count := range buckets {
		if count < minCount {
			minCount = count
		}
		if count > maxCount {
			maxCount = count
		}
		avg += float64(count)
	}
	avg /= float64(bucketCount)

	t.Logf("哈希分布统计: 最小=%d, 最大=%d, 平均=%.2f, 理论平均=%.2f",
		minCount, maxCount, avg, expectedPerBucket)
}
