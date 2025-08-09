package isolation

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
)

type DB struct {
	*sql.DB
	Name  string
	Count int64
}

func (c *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	atomic.AddInt64(&c.Count, 1)
	fmt.Println(c.Name)
	return c.DB.ExecContext(ctx, query, args...)
}
