package connpool

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gotomicro/ego/core/elog"
	"gorm.io/gorm"
	"notification-platform/internal/errs"
	"notification-platform/internal/event/failover"
	"notification-platform/internal/pkg/gormx/monitor"
)

type DBWithFailOver struct {
	db        gorm.ConnPool
	logger    *elog.Component
	dbMonitor monitor.DBMonitor
	producer  failover.ConnPoolEventProducer
}

func (d *DBWithFailOver) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if !d.dbMonitor.Health() {
		// 可以考虑直接拒绝
		return nil, errs.ErrDatabaseError
	}
	return d.db.PrepareContext(ctx, query)
}

func (d *DBWithFailOver) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !d.dbMonitor.Health() {
		// 可以考虑直接拒绝
		return nil, errs.ErrDatabaseError
	}
	return d.db.QueryContext(ctx, query, args...)
}

func (d *DBWithFailOver) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	//if !d.dbMonitor.Health() {
	//	// 可以考虑直接拒绝
	//	return &sql.Row{
	//		// 私有字段，要考虑使用 unsafe 来赋值
	//		err: errs.ErrDatabaseError,
	//	}
	//}
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *DBWithFailOver) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !d.dbMonitor.Health() {
		err := d.producer.Produce(ctx, failover.ConnPoolEvent{
			SQL:  query,
			Args: args,
		})
		if err != nil {
			return nil, fmt.Errorf("数据库有问题转异步失败")
		}
		// 通过 ErrToAsync 代表我这边转异步了
		return nil, errs.ErrToAsync
	}
	return d.db.ExecContext(ctx, query, args...)
}

func NewDBWithFailOver(db *sql.DB,
	dbMonitor monitor.DBMonitor,
	producer failover.ConnPoolEventProducer,
) *DBWithFailOver {
	return &DBWithFailOver{
		db:        db,
		logger:    elog.DefaultLogger,
		dbMonitor: dbMonitor,
		producer:  producer,
	}
}
