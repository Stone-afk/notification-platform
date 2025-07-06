package notification

import (
	"context"
	"fmt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	idgen "notification-platform/internal/pkg/idgenerator"
	"notification-platform/internal/service/sendstrategy"
	"notification-platform/internal/service/template/manage"
)

// sendService 执行器实现
type sendService struct {
	notificationSvc NotificationService
	templateSvc     manage.ChannelTemplateService
	idGenerator     *idgen.Generator
	sendStrategy    sendstrategy.SendStrategy
}

// SendNotification 同步单条发送
func (s *sendService) SendNotification(ctx context.Context, n domain.Notification) (domain.SendResponse, error) {
	resp := domain.SendResponse{
		Status: domain.SendStatusFailed,
	}

	// 参数校验
	if err := n.Validate(); err != nil {
		return resp, err
	}

	// 生成通知ID，后续考虑分库分表
	id := s.idGenerator.GenerateID(n.BizID, n.Key)
	n.ID = uint64(id)
	// 发送通知
	response, err := s.sendStrategy.Send(ctx, n)
	// 处理策略错误
	if err != nil {
		// 通用的发送失败错误
		return resp, fmt.Errorf("%w, 发送通知失败，原因：%w", errs.ErrSendNotificationFailed, err)
	}
	return response, nil
}

// SendNotificationAsync 异步单条发送
func (s *sendService) SendNotificationAsync(ctx context.Context, n domain.Notification) (domain.SendResponse, error) {
	// 参数校验
	if err := n.Validate(); err != nil {
		return domain.SendResponse{}, err
	}

	// 生成通知ID
	id := s.idGenerator.GenerateID(n.BizID, n.Key)
	n.ID = uint64(id)

	// 使用异步接口但要立即发送，修改为延时发送
	// 本质上这是一个不怎好的用法，但是业务方可能不清楚，所以我们兼容一下
	n.ReplaceAsyncImmediate()
	return s.sendStrategy.Send(ctx, n)
}

// BatchSendNotifications 同步批量发送
func (s *sendService) BatchSendNotifications(ctx context.Context, notifications ...domain.Notification) (domain.BatchSendResponse, error) {
	response := domain.BatchSendResponse{}

	// 参数校验
	if len(notifications) == 0 {
		return response, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}

	// 校验并且生成 ID
	for i := range notifications {
		n := notifications[i]
		if err := notifications[i].Validate(); err != nil {
			return domain.BatchSendResponse{}, fmt.Errorf("参数非法 %w", err)
		}
		// 生成通知ID
		id := s.idGenerator.GenerateID(n.BizID, n.Key)
		notifications[i].ID = uint64(id)
	}

	// 发送通知，这里有一个隐含的假设，就是发送策略必须是相同的。
	results, err := s.sendStrategy.BatchSend(ctx, notifications)
	response.Results = results
	if err != nil {
		return response, fmt.Errorf("%w", errs.ErrSendNotificationFailed)
	}
	// 从响应获取结果
	return response, nil
}

// BatchSendNotificationsAsync 异步批量发送
func (s *sendService) BatchSendNotificationsAsync(ctx context.Context, notifications ...domain.Notification) (domain.BatchSendAsyncResponse, error) {
	// 参数校验
	if len(notifications) == 0 {
		return domain.BatchSendAsyncResponse{}, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}
	ids := make([]uint64, 0, len(notifications))
	// 生成 ID，并且进行校验
	for i := range notifications {
		if err := notifications[i].Validate(); err != nil {
			return domain.BatchSendAsyncResponse{}, fmt.Errorf("参数非法 %w", err)
		}
		// 生成通知ID
		id := s.idGenerator.GenerateID(notifications[i].BizID, notifications[i].Key)
		notifications[i].ID = uint64(id)
		ids = append(ids, uint64(id))
		notifications[i].ReplaceAsyncImmediate()
	}

	// 发送通知，隐含假设这一批的发送策略是一样的。
	_, err := s.sendStrategy.BatchSend(ctx, notifications)
	if err != nil {
		return domain.BatchSendAsyncResponse{}, fmt.Errorf("发送失败 %w", errs.ErrSendNotificationFailed)
	}
	return domain.BatchSendAsyncResponse{
		NotificationIDs: ids,
	}, nil
}

// NewSendService 创建执行器实例
func NewSendService(templateSvc manage.ChannelTemplateService, notificationSvc NotificationService, sendStrategy sendstrategy.SendStrategy) SendNotificationService {
	return &sendService{
		notificationSvc: notificationSvc,
		templateSvc:     templateSvc,
		idGenerator:     idgen.NewGenerator(),
		sendStrategy:    sendStrategy,
	}
}
