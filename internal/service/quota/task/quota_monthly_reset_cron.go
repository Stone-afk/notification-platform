package task

import (
	"context"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/repository"
	"notification-platform/internal/service/quota"
	"time"
)

type MonthlyResetCron struct {
	bizRepo   repository.BusinessConfigRepository
	svc       quota.QuotaService
	batchSize int
	logger    *elog.Component
}

func (c *MonthlyResetCron) Do(ctx context.Context) error {
	offset := 0
	for {
		const loopTimeout = time.Second * 15
		ctx, cancel := context.WithTimeout(ctx, loopTimeout)
		cnt, err := c.oneLoop(ctx, offset)
		cancel()
		if err != nil {
			c.logger.Error("查找 Biz配置失败", elog.FieldErr(err))
			// 继续尝试下一批
			offset += c.batchSize
			continue
		}

		if cnt < c.batchSize {
			return nil
		}
		offset += cnt
	}
}

func (c *MonthlyResetCron) oneLoop(ctx context.Context, offset int) (int, error) {
	const findTimeout = time.Second * 3
	ctx, cancel := context.WithTimeout(ctx, findTimeout)
	defer cancel()
	bizs, err := c.bizRepo.Find(ctx, offset, 0)
	if err != nil {
		// 一般都是无可挽回的错误了
		return 0, err
	}
	const resetTimeout = time.Second * 10
	ctx, cancel = context.WithTimeout(ctx, resetTimeout)
	defer cancel()
	for _, cfg := range bizs {
		err = c.svc.ResetQuota(ctx, cfg)
		if err != nil {
			continue
		}
	}
	return len(bizs), nil
}

func NewQuotaMonthlyResetCron(bizRepo repository.BusinessConfigRepository, svc quota.QuotaService) *MonthlyResetCron {
	const batchSize = 10
	return &MonthlyResetCron{
		bizRepo:   bizRepo,
		logger:    elog.DefaultLogger,
		batchSize: batchSize,
		svc:       svc,
	}
}
