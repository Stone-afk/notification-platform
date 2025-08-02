package grpc

import (
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	notificationv1 "notification-platform/api/proto/gen/notification/v1"
	"notification-platform/internal/api/grpc/interceptor/jwt"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	notificationsvc "notification-platform/internal/service/notification"
	templatesvc "notification-platform/internal/service/template/manage"
)

const (
	batchSizeLimit = 100
)

// NotificationServer 通知平台gRPC服务器处理gRPC请求
type NotificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	notificationv1.UnimplementedNotificationQueryServiceServer

	notificationSvc notificationsvc.NotificationService
	sendSvc         notificationsvc.SendNotificationService
	txnSvc          notificationsvc.TxNotificationService
	templateSvc     templatesvc.ChannelTemplateService
}

// SendNotification 处理同步发送通知请求
func (s *NotificationServer) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	response := &notificationv1.SendNotificationResponse{}

	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	// 构建领域对象
	notification, err := s.buildNotification(ctx, req.Notification, bizID)
	if err != nil {
		response.ErrorCode = notificationv1.ErrorCode_INVALID_PARAMETER
		response.ErrorMessage = err.Error()
		response.Status = notificationv1.SendStatus_FAILED
		return response, nil
	}

	// 调用发送服务
	result, err := s.sendSvc.SendNotification(ctx, notification)
	if err != nil {
		if s.isSystemError(err) {
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			response.ErrorCode = s.convertToGRPCErrorCode(err)
			response.ErrorMessage = err.Error()
			response.Status = notificationv1.SendStatus_FAILED
			return response, nil
		}
	}

	response.NotificationId = result.NotificationID
	response.Status = s.convertToGRPCSendStatus(result.Status)
	return response, nil
}

// SendNotificationAsync 处理异步发送通知请求
func (s *NotificationServer) SendNotificationAsync(ctx context.Context, req *notificationv1.SendNotificationAsyncRequest) (*notificationv1.SendNotificationAsyncResponse, error) {
	response := &notificationv1.SendNotificationAsyncResponse{}

	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 构建领域对象
	notification, err := s.buildNotification(ctx, req.Notification, bizID)
	if err != nil {
		response.ErrorCode = notificationv1.ErrorCode_INVALID_PARAMETER
		response.ErrorMessage = err.Error()
		return response, nil
	}

	// 执行发送
	result, err := s.sendSvc.SendNotificationAsync(ctx, notification)
	if err != nil {
		// 区分系统错误和业务错误
		if s.isSystemError(err) {
			// 系统错误通过gRPC错误返回
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			// 业务错误通过ErrorCode返回
			response.ErrorCode = s.convertToGRPCErrorCode(err)
			response.ErrorMessage = err.Error()
			return response, nil
		}
	}

	// 将结果转换为响应
	response.NotificationId = result.NotificationID
	return response, nil
}

// BatchSendNotifications 处理批量同步发送通知请求
func (s *NotificationServer) BatchSendNotifications(ctx context.Context, req *notificationv1.BatchSendNotificationsRequest) (*notificationv1.BatchSendNotificationsResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 处理空请求或空通知列表
	if req == nil || len(req.Notifications) == 0 {
		return &notificationv1.BatchSendNotificationsResponse{
			TotalCount:   0,
			SuccessCount: 0,
			Results:      []*notificationv1.SendNotificationResponse{},
		}, nil
	}

	if len(req.Notifications) > batchSizeLimit {
		return nil, status.Errorf(codes.InvalidArgument, "%v: %d > %d", errs.ErrBatchSizeOverLimit, len(req.Notifications), batchSizeLimit)
	}

	var hasError bool
	results := make([]*notificationv1.SendNotificationResponse, len(req.Notifications))
	// 构建领域对象
	notifications := make([]domain.Notification, 0, len(req.Notifications))
	for i := range req.Notifications {
		notification, err1 := s.buildNotification(ctx, req.Notifications[i], bizID)
		if err1 != nil {
			results[i] = &notificationv1.SendNotificationResponse{
				ErrorCode:    notificationv1.ErrorCode_INVALID_PARAMETER,
				ErrorMessage: err1.Error(),
				Status:       notificationv1.SendStatus_FAILED,
			}
			hasError = true
			continue
		}
		notifications = append(notifications, notification)
	}
	if hasError {
		return &notificationv1.BatchSendNotificationsResponse{
			TotalCount:   int32(len(results)),
			SuccessCount: int32(0),
			Results:      results,
		}, nil
	}

	// 执行发送
	responses, err := s.sendSvc.BatchSendNotifications(ctx, notifications...)
	if err != nil {
		if s.isSystemError(err) {
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			for i := range results {
				results[i] = &notificationv1.SendNotificationResponse{
					ErrorCode:    s.convertToGRPCErrorCode(err),
					ErrorMessage: err.Error(),
					Status:       notificationv1.SendStatus_FAILED,
				}
			}
			return &notificationv1.BatchSendNotificationsResponse{
				TotalCount:   int32(len(results)),
				SuccessCount: int32(0),
				Results:      results,
			}, nil
		}
	}

	// 将结果转换为响应
	const first = 0
	successCount := int32(0)
	for i := range responses.Results {
		results[i] = s.buildGRPCSendResponse(responses.Results[i], nil)
		if notifications[first].SendStrategyConfig.Type == domain.SendStrategyImmediate &&
			domain.SendStatusSucceeded == responses.Results[i].Status {
			successCount++
		}
		if notifications[first].SendStrategyConfig.Type != domain.SendStrategyImmediate &&
			domain.SendStatusPending == responses.Results[i].Status {
			successCount++
		}
	}
	return &notificationv1.BatchSendNotificationsResponse{
		TotalCount:   int32(len(results)),
		SuccessCount: successCount,
		Results:      results,
	}, nil
}

// buildGRPCSendResponse 将领域响应转换为gRPC响应
func (s *NotificationServer) buildGRPCSendResponse(result domain.SendResponse, err error) *notificationv1.SendNotificationResponse {
	response := &notificationv1.SendNotificationResponse{
		NotificationId: result.NotificationID,
		Status:         s.convertToGRPCSendStatus(result.Status),
	}
	// 如果有错误，提取错误代码和消息
	if err != nil {
		response.ErrorMessage = err.Error()
		response.ErrorCode = s.convertToGRPCErrorCode(err)

		// 如果状态不是失败，但有错误，更新状态为失败
		if response.Status != notificationv1.SendStatus_FAILED {
			response.Status = notificationv1.SendStatus_FAILED
		}
	}
	return response
}

// BatchSendNotificationsAsync 处理批量异步发送通知请求
func (s *NotificationServer) BatchSendNotificationsAsync(ctx context.Context, req *notificationv1.BatchSendNotificationsAsyncRequest) (*notificationv1.BatchSendNotificationsAsyncResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 处理空请求或空通知列表
	if req == nil || len(req.Notifications) == 0 {
		return &notificationv1.BatchSendNotificationsAsyncResponse{
			NotificationIds: []uint64{},
		}, nil
	}

	if len(req.Notifications) > batchSizeLimit {
		return nil, status.Errorf(codes.InvalidArgument, "%v: %d > %d", errs.ErrBatchSizeOverLimit, len(req.Notifications), batchSizeLimit)
	}

	// 构建领域对象
	notifications := make([]domain.Notification, 0, len(req.Notifications))
	for i := range req.Notifications {
		notification, err1 := s.buildNotification(ctx, req.Notifications[i], bizID)
		if err1 != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v: %#v", err1, req.Notifications[i])
		}
		notifications = append(notifications, notification)
	}

	// 执行发送
	result, err := s.sendSvc.BatchSendNotificationsAsync(ctx, notifications...)
	if err != nil {
		if s.isSystemError(err) {
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else {
			return nil, status.Errorf(codes.InvalidArgument, "批量异步发送失败: %v", err)
		}
	}

	// 将结果转换为响应
	return &notificationv1.BatchSendNotificationsAsyncResponse{
		NotificationIds: result.NotificationIDs,
	}, nil
}

// TxPrepare 处理事务通知准备请求
func (s *NotificationServer) TxPrepare(ctx context.Context, request *notificationv1.TxPrepareRequest) (*notificationv1.TxPrepareResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 构建领域对象
	txn, err := s.buildTxNotification(ctx, request.Notification, bizID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "无效的请求参数: %v", err)
	}

	// 执行操作
	_, err = s.txnSvc.Prepare(ctx, txn.Notification)
	return &notificationv1.TxPrepareResponse{}, err
}

func (s *NotificationServer) buildTxNotification(ctx context.Context, n *notificationv1.Notification, bizID int64) (domain.TxNotification, error) {
	if n == nil {
		return domain.TxNotification{}, errors.New("通知不能为空")
	}

	// 构建基本Notification
	noti, err := s.buildNotification(ctx, n, bizID)
	noti.Status = domain.SendStatusPrepare
	if err != nil {
		return domain.TxNotification{}, status.Errorf(codes.InvalidArgument, "无效的请求参数: %v", err)
	}
	return domain.TxNotification{
		BizID:        bizID,
		Key:          n.Key,
		Notification: noti,
		Status:       domain.TxNotificationStatusPrepare,
	}, nil
}

// TxCommit 处理事务通知提交请求
func (s *NotificationServer) TxCommit(ctx context.Context, request *notificationv1.TxCommitRequest) (*notificationv1.TxCommitResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	err = s.txnSvc.Commit(ctx, bizID, request.GetKey())
	return &notificationv1.TxCommitResponse{}, err
}

// TxCancel 处理事务通知取消请求
func (s *NotificationServer) TxCancel(ctx context.Context, request *notificationv1.TxCancelRequest) (*notificationv1.TxCancelResponse, error) {
	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	// 执行操作
	err = s.txnSvc.Cancel(ctx, bizID, request.GetKey())
	return &notificationv1.TxCancelResponse{}, err
}

// QueryNotification 处理单条查询通知请求
func (s *NotificationServer) QueryNotification(ctx context.Context, req *notificationv1.QueryNotificationRequest) (*notificationv1.QueryNotificationResponse, error) {
	// 检查请求参数
	if req == nil || req.Key == "" {
		return nil, status.Errorf(codes.InvalidArgument, "请求参数无效: key不能为空")
	}

	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 查询通知
	notifications, err := s.notificationSvc.GetByKeys(ctx, bizID, req.Key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询通知失败: %v", err)
	}

	// 检查是否找到通知
	if len(notifications) == 0 {
		return &notificationv1.QueryNotificationResponse{
			Result: &notificationv1.SendNotificationResponse{},
		}, nil
	}

	// 将结果转换为响应
	const zero = 0
	return &notificationv1.QueryNotificationResponse{
		Result: &notificationv1.SendNotificationResponse{
			NotificationId: notifications[zero].ID,
			Status:         s.convertToGRPCSendStatus(notifications[zero].Status),
		},
	}, nil
}

// BatchQueryNotifications 处理批量查询通知请求
func (s *NotificationServer) BatchQueryNotifications(ctx context.Context, req *notificationv1.BatchQueryNotificationsRequest) (*notificationv1.BatchQueryNotificationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "请求不能为空")
	}

	if len(req.Keys) == 0 {
		return &notificationv1.BatchQueryNotificationsResponse{
			Results: []*notificationv1.SendNotificationResponse{},
		}, nil
	}

	if len(req.Keys) > batchSizeLimit {
		return nil, status.Errorf(codes.InvalidArgument, "%v: %d > %d", errs.ErrBatchSizeOverLimit, len(req.Keys), batchSizeLimit)
	}

	// 从metadata中解析Authorization JWT Token
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// 查询通知
	notifications, err := s.notificationSvc.GetByKeys(ctx, bizID, req.Keys...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "批量查询通知失败: %v", err)
	}

	// 将结果转换为响应
	response := &notificationv1.BatchQueryNotificationsResponse{
		Results: make([]*notificationv1.SendNotificationResponse, 0, len(notifications)),
	}

	for i := range notifications {
		response.Results = append(response.Results, &notificationv1.SendNotificationResponse{
			NotificationId: notifications[i].ID,
			Status:         s.convertToGRPCSendStatus(notifications[i].Status),
		})
	}

	return response, nil
}

func (s *NotificationServer) convertToGRPCSendStatus(status domain.SendStatus) notificationv1.SendStatus {
	switch status {
	case domain.SendStatusPrepare:
		return notificationv1.SendStatus_PREPARE
	case domain.SendStatusCanceled:
		return notificationv1.SendStatus_CANCELED
	case domain.SendStatusPending:
		return notificationv1.SendStatus_PENDING
	case domain.SendStatusSucceeded:
		return notificationv1.SendStatus_SUCCEEDED
	case domain.SendStatusFailed:
		return notificationv1.SendStatus_FAILED
	default:
		return notificationv1.SendStatus_SEND_STATUS_UNSPECIFIED
	}
}

// convertToGRPCErrorCode 将错误映射为gRPC错误代码
func (s *NotificationServer) convertToGRPCErrorCode(err error) notificationv1.ErrorCode {
	// 注意：这个函数只处理业务错误，系统错误由isSystemError判断后直接通过gRPC status返回
	switch {
	case errors.Is(err, errs.ErrInvalidParameter):
		return notificationv1.ErrorCode_INVALID_PARAMETER

	case errors.Is(err, errs.ErrTemplateNotFound):
		return notificationv1.ErrorCode_TEMPLATE_NOT_FOUND

	case errors.Is(err, errs.ErrChannelDisabled):
		return notificationv1.ErrorCode_CHANNEL_DISABLED

	case errors.Is(err, errs.ErrRateLimited):
		return notificationv1.ErrorCode_RATE_LIMITED

	case errors.Is(err, errs.ErrBizIDNotFound):
		return notificationv1.ErrorCode_BIZ_ID_NOT_FOUND

	case errors.Is(err, errs.ErrSendNotificationFailed):
		return notificationv1.ErrorCode_SEND_NOTIFICATION_FAILED

	case errors.Is(err, errs.ErrCreateNotificationFailed):
		return notificationv1.ErrorCode_CREATE_NOTIFICATION_FAILED

	case errors.Is(err, errs.ErrNotificationNotFound):
		return notificationv1.ErrorCode_NOTIFICATION_NOT_FOUND

	case errors.Is(err, errs.ErrNoAvailableProvider):
		return notificationv1.ErrorCode_NO_AVAILABLE_PROVIDER

	case errors.Is(err, errs.ErrNoAvailableChannel):
		return notificationv1.ErrorCode_NO_AVAILABLE_CHANNEL

	case errors.Is(err, errs.ErrConfigNotFound):
		return notificationv1.ErrorCode_CONFIG_NOT_FOUND

	case errors.Is(err, errs.ErrNoQuotaConfig):
		return notificationv1.ErrorCode_NO_QUOTA_CONFIG

	case errors.Is(err, errs.ErrNoQuota):
		return notificationv1.ErrorCode_NO_QUOTA

	case errors.Is(err, errs.ErrQuotaNotFound):
		return notificationv1.ErrorCode_QUOTA_NOT_FOUND

	case errors.Is(err, errs.ErrProviderNotFound):
		return notificationv1.ErrorCode_PROVIDER_NOT_FOUND

	case errors.Is(err, errs.ErrUnknownChannel):
		return notificationv1.ErrorCode_UNKNOWN_CHANNEL

	default:
		return notificationv1.ErrorCode_ERROR_CODE_UNSPECIFIED
	}
}

func (s *NotificationServer) buildNotification(ctx context.Context, n *notificationv1.Notification, bizID int64) (domain.Notification, error) {
	notification, err := domain.NewNotificationFromAPI(n)
	if err != nil {
		return domain.Notification{}, err
	}

	tmpl, err := s.templateSvc.GetTemplateByID(ctx, notification.Template.ID)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("%w: 模板ID: %s", errs.ErrInvalidParameter, n.TemplateId)
	}

	if !tmpl.HasPublished() {
		return domain.Notification{}, fmt.Errorf("%w: 模板ID: %s 未发布", errs.ErrInvalidParameter, n.TemplateId)
	}

	notification.BizID = bizID
	notification.Template.VersionID = tmpl.ActiveVersionID
	return notification, nil
}

// isSystemError 判断错误是否为系统错误
func (s *NotificationServer) isSystemError(err error) bool {
	return errors.Is(err, errs.ErrDatabaseError) ||
		errors.Is(err, errs.ErrExternalServiceError) ||
		errors.Is(err, errs.ErrNotificationDuplicate) ||
		errors.Is(err, errs.ErrNotificationVersionMismatch)
}

// NewNotificationServer 创建通知平台gRPC服务器
func NewNotificationServer(notificationSvc notificationsvc.NotificationService,
	sendSvc notificationsvc.SendNotificationService,
	txnSvc notificationsvc.TxNotificationService,
	templateSvc templatesvc.ChannelTemplateService,
) *NotificationServer {
	return &NotificationServer{
		notificationSvc: notificationSvc,
		sendSvc:         sendSvc,
		txnSvc:          txnSvc,
		templateSvc:     templateSvc,
	}
}
