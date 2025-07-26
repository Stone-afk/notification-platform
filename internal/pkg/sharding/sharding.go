package sharding

import (
	"context"
	"fmt"
	"strings"

	"notification-platform/internal/pkg/hash"
	idgen "notification-platform/internal/pkg/idgenerator"
)

type ShardingStrategy struct {
	dbPrefix      string
	tablePrefix   string
	tableSharding int64
	dbSharding    int64
}

type Dst struct {
	Table string
	DB    string

	TableSuffix int64
	DBSuffix    int64
}

func NewShardingStrategy(dbPrefix, tablePrefix string, tableSharding, dbSharding int64) ShardingStrategy {
	return ShardingStrategy{
		dbSharding:    dbSharding,
		tableSharding: tableSharding,
		dbPrefix:      dbPrefix,
		tablePrefix:   tablePrefix,
	}
}

func (s ShardingStrategy) Shard(bizID int64, key string) Dst {
	hashValue := hash.Hash(bizID, key)
	dbHash := hashValue % s.dbSharding
	tabHash := (hashValue / s.dbSharding) % s.tableSharding
	return Dst{
		TableSuffix: tabHash,
		Table:       fmt.Sprintf("%s_%d", s.tablePrefix, tabHash),
		DBSuffix:    dbHash,
		DB:          fmt.Sprintf("%s_%d", s.dbPrefix, dbHash),
	}
}

func (s ShardingStrategy) ShardWithID(id int64) Dst {
	hashValue := idgen.ExtractHashValue(id)
	dbHash := hashValue % s.dbSharding
	tabHash := (hashValue / s.dbSharding) % s.tableSharding
	return Dst{
		TableSuffix: tabHash,
		Table:       fmt.Sprintf("%s_%d", s.tablePrefix, tabHash),
		DB:          fmt.Sprintf("%s_%d", s.dbPrefix, dbHash),
		DBSuffix:    dbHash,
	}
}

func (s ShardingStrategy) Broadcast() []Dst {
	ans := make([]Dst, 0, s.tableSharding*s.dbSharding)
	for i := 0; i < int(s.dbSharding); i++ {
		for j := 0; j < int(s.tableSharding); j++ {
			ans = append(ans, Dst{
				TableSuffix: int64(j),
				Table:       fmt.Sprintf("%s_%d", s.tablePrefix, j),
				DB:          fmt.Sprintf("%s_%d", s.dbPrefix, i),
				DBSuffix:    int64(i),
			})
		}
	}
	return ans
}

func (s ShardingStrategy) TablePrefix() string {
	return s.tablePrefix
}

type dstKey struct{}

func DstFromCtx(ctx context.Context) (Dst, bool) {
	val := ctx.Value(dstKey{})
	res, ok := val.(Dst)
	return res, ok
}

func CtxWithDst(ctx context.Context, dst Dst) context.Context {
	return context.WithValue(ctx, dstKey{}, dst)
}

// ExtractSuffixAndFormatFromTable 从表名中提取后缀，按照下划线分隔并返回最后一个元素
func (s ShardingStrategy) ExtractSuffixAndFormatFromTable(tableName string) string {
	parts := strings.Split(tableName, "_")
	suffix := parts[len(parts)-1]
	return fmt.Sprintf("%s_%s", s.tablePrefix, suffix)
}
