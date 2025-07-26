package template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/domain"
	auditevt "notification-platform/internal/event/audit"
	"notification-platform/internal/pkg/mqx"
	templatesvc "notification-platform/internal/service/template/manage"
	"time"
)

const (
	auditEventName = "audit_result_events"
)

type AuditResultConsumer struct {
	svc      templatesvc.ChannelTemplateService
	consumer mqx.Consumer

	batchSize    int
	batchTimeout time.Duration

	logger *elog.Component
}

func (c *AuditResultConsumer) Consume(ctx context.Context) error {
	// 限制时间
	batchTimer := time.NewTimer(c.batchTimeout)
	defer batchTimer.Stop()

	var (
		versions    = make([]domain.ChannelTemplateVersion, 0, c.batchSize)
		versionIDs  = make([]int64, 0, c.batchSize)
		evt         auditevt.CallbackResultEvent
		lastMessage *kafka.Message
	)
collectBatch:
	for {
		select {
		case <-ctx.Done():
			// ctx 被取消
			break collectBatch
		case <-batchTimer.C:
			// 达到时间限制，跳出循环
			break collectBatch
		default:
			// 达到批量大小限制，跳出循环
			if len(versions) == c.batchSize {
				break collectBatch
			}
		}

		msg, err := c.consumer.ReadMessage(c.batchTimeout)
		if err != nil {
			var kErr kafka.Error
			if errors.As(err, &kErr) && kErr.Code() == kafka.ErrTimedOut {
				// 聚合当前批次已超时
				break
			}
			return fmt.Errorf("获取消息失败: %w", err)
		}

		if err = json.Unmarshal(msg.Value, &evt); err != nil {
			c.logger.Warn("解析消息失败",
				elog.FieldErr(err),
				elog.Any("msg", msg))
			// 跳过不合规的消息
			// 解析失败，跳过本条，继续下一轮
			continue
		}

		if !evt.ResourceType.IsTemplate() {
			continue
		}

		version := domain.ChannelTemplateVersion{
			ID:           evt.ResourceID,
			AuditID:      evt.AuditID,
			AuditorID:    evt.AuditorID,
			AuditTime:    evt.AuditTime,
			AuditStatus:  domain.AuditStatus(evt.AuditStatus),
			RejectReason: evt.RejectReason,
		}

		if version.AuditStatus.IsApproved() {
			versionIDs = append(versionIDs, version.ID)
		}

		versions = append(versions, version)
		lastMessage = msg
	}

	// 没有可以更新的内容
	if len(versions) == 0 {
		return nil
	}

	err := c.svc.BatchUpdateVersionAuditStatus(ctx, versions)
	if err != nil {
		c.logger.Warn("更新模版版本内部审核信息失败",
			elog.FieldErr(err),
			elog.Any("versions", versions),
		)
		return err
	}

	err = c.svc.BatchSubmitForProviderReview(ctx, versionIDs)
	if err != nil {
		c.logger.Warn("内部审核通过的模版，提交到供应商侧审核失败",
			elog.FieldErr(err),
			elog.Any("versionIDs", versionIDs),
		)
		return err
	}

	// 只提交每个分区的最后一条消息，这也就假定当前消费者只消费一个分区
	_, err = c.consumer.CommitMessage(lastMessage)
	if err != nil {
		c.logger.Warn("提交消息失败",
			elog.FieldErr(err),
			elog.Any("partition", lastMessage.TopicPartition.Partition),
			elog.Any("offset", lastMessage.TopicPartition.Offset))
		return err
	}
	return nil
}

func (c *AuditResultConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费审核状态结果事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func NewAuditResultConsumer(svc templatesvc.ChannelTemplateService,
	consumer *kafka.Consumer,
	batchSize int,
	batchTimeout time.Duration,
) (*AuditResultConsumer, error) {
	err := consumer.SubscribeTopics([]string{auditEventName}, nil)
	if err != nil {
		return nil, err
	}
	return &AuditResultConsumer{
		svc:          svc,
		consumer:     consumer,
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		logger:       elog.DefaultLogger,
	}, nil
}
