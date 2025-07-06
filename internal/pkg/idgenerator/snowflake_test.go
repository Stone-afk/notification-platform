//go:build unit

package id

import (
	"testing"

	"gitee.com/flycash/notification-platform/internal/pkg/hash"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndExtract(t *testing.T) {
	t.Parallel()
	generator := NewGenerator()

	// 测试数据
	bizID := int64(43)
	key := "test-key1"

	// 生成ID
	id := generator.GenerateID(bizID, key)

	// 提取并验证哈希值
	hashValue := ExtractHashValue(id)
	hashWantVal := hash.Hash(bizID, key)
	assert.Equal(t, hashWantVal%1024, hashValue)
	if hashValue < 0 || hashValue >= 1024 {
		t.Errorf("哈希值不在有效范围内: %d", hashValue)
	}

	// 提取并验证序列号
	sequence := ExtractSequence(id)
	// 第一个生成的ID的序列号应该是0
	assert.Equal(t, int64(0), sequence)
	if sequence < 0 || sequence >= (1<<12) {
		t.Errorf("序列号不在有效范围内: %d", sequence)
	}
}

func TestIDUniqueness(t *testing.T) {
	t.Parallel()
	generator := NewGenerator()

	// 生成多个ID并检查唯一性
	idCount := 1000
	idSet := make(map[int64]struct{}, idCount)

	for i := 0; i < idCount; i++ {
		bizID := int64(i % 100)                // 循环使用一些bizId
		key := "key-" + string(rune('A'+i%26)) // 循环使用一些key

		id := generator.GenerateID(bizID, key) // 使用当前时间

		if _, exists := idSet[id]; exists {
			t.Fatalf("发现重复ID: %d", id)
		}

		idSet[id] = struct{}{}
	}

	t.Logf("成功生成 %d 个不同的ID", idCount)
}

func TestSequenceIncrement(t *testing.T) {
	t.Parallel()
	// 测试序列号自增功能
	generator := NewGenerator()

	// 使用相同的bizId, key和时间生成多个ID
	bizID := int64(123)
	key := "same-key"

	// 生成多个ID并验证序列号递增
	count := 10
	ids := make([]int64, count)
	for i := 0; i < count; i++ {
		ids[i] = generator.GenerateID(bizID, key)

		// 验证序列号是否递增
		sequence := ExtractSequence(ids[i])
		assert.Equal(t, int64(i), sequence, "序列号应该从0开始递增")
	}

	// 验证时间戳和哈希值部分相同
	for i := 1; i < count; i++ {
		// 哈希值应该相同
		hash1 := ExtractHashValue(ids[0])
		hash2 := ExtractHashValue(ids[i])
		assert.Equal(t, hash1, hash2, "相同输入产生的不同ID的哈希值应该一致")
	}
}

func TestSequenceRollover(t *testing.T) {
	t.Parallel()
	// 测试序列号溢出回环
	generator := NewGenerator()

	// 直接修改generator中的序列号为最大值
	generator.sequence = sequenceMask

	bizID := int64(456)
	key := "rollover-test"

	// 生成第一个ID，此时序列号应为最大值
	id1 := generator.GenerateID(bizID, key)
	seq1 := ExtractSequence(id1)
	assert.Equal(t, int64(sequenceMask), seq1, "序列号应为最大值")

	// 生成第二个ID，此时序列号应回环为0
	id2 := generator.GenerateID(bizID, key)
	seq2 := ExtractSequence(id2)
	assert.Equal(t, int64(0), seq2, "序列号溢出后应回环为0")
}
