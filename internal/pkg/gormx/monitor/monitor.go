package monitor

import (
	"context"
	"database/sql"
	"github.com/gotomicro/ego/core/elog"
	"sync/atomic"
	"time"
)

const (
	timeout             = 5 * time.Second
	defaultFailCount    = 3
	defaultSuccessCount = 3
)

// Heartbeat 心跳监控
type Heartbeat struct {
	db             *sql.DB
	logger         *elog.Component
	health         *atomic.Bool
	failCounter    *atomic.Int32 // 连续失败计数器
	successCounter *atomic.Int32 // 连续成功计数器（用于恢复）
}

func (h *Heartbeat) Health() bool {
	return h.health.Load()
}

func (*Heartbeat) Report(error) {}

func (h *Heartbeat) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// 如果超时就返回
			if ctx.Err() != nil {
				h.logger.Error("ctx超时退出", elog.FieldErr(ctx.Err()))
				return
			}
			return
		case <-ticker.C:
			// 执行健康检查
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			err := h.healthOneLoop(timeoutCtx)
			cancel()
			if err != nil {
				h.logger.Error("ConnPool健康检查失败", elog.FieldErr(err))
			}
		}
	}
}

func (h *Heartbeat) healthOneLoop(ctx context.Context) error {
	err := h.db.PingContext(ctx)
	if err != nil {
		// 失败时递增失败计数器，重置成功计数器
		h.successCounter.Store(0)
		if h.failCounter.Add(1) >= defaultFailCount {
			h.health.Store(false)
			h.failCounter.Store(0) // 重置计数器
		}
		return err
	}
	// 成功时递增成功计数器，重置失败计数器
	h.failCounter.Store(0)
	if h.successCounter.Add(1) >= defaultSuccessCount {
		h.health.Store(true)
		h.successCounter.Store(0) // 重置计数器
	}
	return nil
}

func NewHeartbeatDBMonitor(db *sql.DB) *Heartbeat {
	he := &atomic.Bool{}
	he.Store(true)

	h := &Heartbeat{
		db:             db,
		logger:         elog.DefaultLogger,
		health:         he,
		failCounter:    &atomic.Int32{},
		successCounter: &atomic.Int32{},
	}
	go h.healthCheck(context.Background())
	return h
}
