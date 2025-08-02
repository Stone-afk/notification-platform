package grpc

import (
	"context"
	"time"

	"notification-platform/internal/api/grpc/interceptor/jwt"

	configv1 "notification-platform/api/proto/gen/config/v1"
	"notification-platform/internal/domain"
	"notification-platform/internal/pkg/retry"
	"notification-platform/internal/service/config"
)

type ConfigServer struct {
	configv1.UnimplementedBusinessConfigServiceServer
	configSvc config.BusinessConfigService
}

func NewConfigServer(configSvc config.BusinessConfigService) *ConfigServer {
	return &ConfigServer{
		configSvc: configSvc,
	}
}

func (c *ConfigServer) GetByIDs(ctx context.Context, request *configv1.GetByIDsRequest) (*configv1.GetByIDsResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (c *ConfigServer) GetByID(ctx context.Context, request *configv1.GetByIDRequest) (*configv1.GetByIDResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (c *ConfigServer) Delete(ctx context.Context, request *configv1.DeleteRequest) (*configv1.DeleteResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (c *ConfigServer) SaveConfig(ctx context.Context, request *configv1.SaveConfigRequest) (*configv1.SaveConfigResponse, error) {
	if request == nil || request.Config == nil {
		return &configv1.SaveConfigResponse{
			Success: false,
		}, nil
	}
	bizID, err := jwt.GetBizIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	// Convert from protobuf to domain model
	domainConfig := protoToDomainBusinessConfig(request.Config)
	domainConfig.ID = bizID
	// We need to set the ID from the request context or another source
	// For now, we'll assume it's already set

	// Call the service to save the config
	err = c.configSvc.SaveConfig(ctx, domainConfig)
	if err != nil {
		return &configv1.SaveConfigResponse{
			Success: false,
		}, err
	}

	return &configv1.SaveConfigResponse{
		Success: true,
	}, nil
}

// protoToDomainBusinessConfig converts a protobuf BusinessConfig to domain BusinessConfig
func protoToDomainBusinessConfig(protoConfig *configv1.BusinessConfig) domain.BusinessConfig {
	var domainConfig domain.BusinessConfig

	// Set the fields from protobuf
	// Note: ID must be set from elsewhere or context, as it's not in the proto
	domainConfig.OwnerID = protoConfig.OwnerId
	domainConfig.OwnerType = protoConfig.OwnerType
	domainConfig.RateLimit = int(protoConfig.RateLimit)

	// Convert ChannelConfig if exists
	if protoConfig.ChannelConfig != nil {
		channelConfig := &domain.ChannelConfig{
			Channels: make([]domain.ChannelItem, 0, len(protoConfig.ChannelConfig.Channels)),
		}

		// Convert each channel item
		for _, channel := range protoConfig.ChannelConfig.Channels {
			channelConfig.Channels = append(channelConfig.Channels, domain.ChannelItem{
				Channel:  channel.Channel,
				Priority: int(channel.Priority),
				Enabled:  channel.Enabled,
			})
		}

		// Convert retry policy if exists
		if protoConfig.ChannelConfig.RetryPolicy != nil {
			retryPolicy := convertProtoRetryConfig(protoConfig.ChannelConfig.RetryPolicy)
			channelConfig.RetryPolicy = retryPolicy
		}

		domainConfig.ChannelConfig = channelConfig
	}

	// Convert TxnConfig if exists
	if protoConfig.TxnConfig != nil {
		txnConfig := &domain.TxnConfig{
			ServiceName:  protoConfig.TxnConfig.ServiceName,
			InitialDelay: int(protoConfig.TxnConfig.InitialDelay),
		}

		// Convert retry policy if exists
		if protoConfig.TxnConfig.RetryPolicy != nil {
			txnConfig.RetryPolicy = convertProtoRetryConfig(protoConfig.TxnConfig.RetryPolicy)
		}

		domainConfig.TxnConfig = txnConfig
	}

	// Convert QuotaConfig if exists
	if protoConfig.Quota != nil && protoConfig.Quota.Monthly != nil {
		domainConfig.Quota = &domain.QuotaConfig{
			Monthly: domain.MonthlyConfig{
				SMS:   int(protoConfig.Quota.Monthly.Sms),
				EMAIL: int(protoConfig.Quota.Monthly.Email),
			},
		}
	}

	// Convert CallbackConfig if exists
	if protoConfig.CallbackConfig != nil {
		callbackConfig := &domain.CallbackConfig{
			ServiceName: protoConfig.CallbackConfig.ServiceName,
		}

		// Convert retry policy if exists
		if protoConfig.CallbackConfig.RetryPolicy != nil {
			callbackConfig.RetryPolicy = convertProtoRetryConfig(protoConfig.CallbackConfig.RetryPolicy)
		}

		domainConfig.CallbackConfig = callbackConfig
	}

	return domainConfig
}

func convertProtoRetryConfig(protoRetry *configv1.RetryConfig) *retry.Config {
	return &retry.Config{
		Type: "exponential",
		ExponentialBackoff: &retry.ExponentialBackoffConfig{
			InitialInterval: time.Duration(protoRetry.InitialBackoffMs) * time.Millisecond,
			MaxInterval:     time.Duration(protoRetry.MaxBackoffMs) * time.Millisecond,
			MaxRetries:      protoRetry.MaxAttempts,
		},
	}
}
