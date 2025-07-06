package hash

import (
	"hash/fnv"
	"math/bits"
	"strconv"
)

const (
	hashMask int64 = 0x7FFFFFFFFFFFFFFF
	number13       = 13
	number29       = 29
	number31       = 31
)

// Hash 生成一个64位哈希值，基于bizId和key，具有极低的碰撞概率。
// 它使用FNV-1a算法结合额外的位运算混合技术。
// 返回值保证为非负数。
func Hash(bizID int64, key string) int64 {
	// 将bizId和字符串key组合成一个字符串以最大化熵值
	combinedKey := strconv.FormatInt(bizID, 10) + ":" + key

	// 使用FNV-1a作为基础算法（64位）
	h := fnv.New64a()
	_, _ = h.Write([]byte(combinedKey))
	hash := h.Sum64()

	// 应用额外的混合函数进一步减少碰撞
	hash = mixHash(hash, uint64(bizID))

	// 转换为int64类型，并确保结果为非负数（通过清除符号位）
	return int64(hash) & hashMask
}

// mixHash 应用额外的混合算法以改善哈希分布
func mixHash(h, salt uint64) uint64 {
	// 选择具有良好分布特性的常量
	const (
		prime1 = 11400714819323198485
		prime2 = 14029467366897019727
		prime3 = 1609587929392839161
	)

	// 用盐值（bizID）进行混合
	h ^= salt + prime1

	// 应用位旋转和乘法以产生雪崩效应
	h = bits.RotateLeft64(h, number13)
	h *= prime2
	h = bits.RotateLeft64(h, number29)
	h *= prime3
	h = bits.RotateLeft64(h, number31)

	return h
}
