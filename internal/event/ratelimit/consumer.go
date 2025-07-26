package ratelimit

import (
	"github.com/gotomicro/ego/core/elog"
	"notification-platform/internal/pkg/mqx"
	"notification-platform/internal/pkg/ratelimit"
	notificationsvc "notification-platform/internal/service/notification"
	"time"
)

const (
	defaultPollInterval = time.Second
)

type RequestRateLimitedEventConsumer struct {
	srv      notificationsvc.SendNotificationService
	consumer mqx.Consumer

	limiter          ratelimit.Limiter
	limitedKey       string
	lookbackDuration time.Duration
	sleepDuration    time.Duration

	logger *elog.Component
}
