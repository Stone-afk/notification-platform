package task

import (
	"context"
	"fmt"
	"github.com/meoying/dlock-go"
	"notification-platform/internal/pkg/loopjob"
	templatesvc "notification-platform/internal/service/template/manage"
	"time"
)

type SyncProviderAuditInfoTask struct {
	dclient dlock.Client
	svc     templatesvc.ChannelTemplateService
}

func (s *SyncProviderAuditInfoTask) Start(ctx context.Context) {
	const key = "notification_handling_sync_provider_audit_info"
	lj := loopjob.NewInfiniteLoop(s.dclient, s.HandleSyncProviderAuditInfo, key)
	lj.Run(ctx)
}

func (s *SyncProviderAuditInfoTask) HandleSyncProviderAuditInfo(ctx context.Context) error {
	const batchSize = 10
	const minDuration = 3 * time.Second
	now := time.Now()
	for {
		providers, total, err := s.svc.GetPendingOrInReviewProviders(ctx, 0, batchSize, now.UnixMilli())
		if err != nil {
			return fmt.Errorf("获取未完成审核的供应商关联记录失败: %w", err)
		}
		if len(providers) == 0 {
			break
		}

		err = s.svc.BatchQueryAndUpdateProviderAuditInfo(ctx, providers)
		if err != nil {
			return fmt.Errorf("获取并更新供应商审核信息失败: %w", err)
		}

		if len(providers) < batchSize {
			break
		}

		if int64(batchSize) >= total {
			break
		}
	}
	// 确保任务至少运行minDuration时间，避免过快重复执行
	duration := time.Since(now)
	if duration < minDuration {
		time.Sleep(minDuration - duration)
	}

	return nil
}

func NewSyncProviderAuditInfoTask(dclient dlock.Client, svc templatesvc.ChannelTemplateService) *SyncProviderAuditInfoTask {
	return &SyncProviderAuditInfoTask{dclient: dclient, svc: svc}
}
