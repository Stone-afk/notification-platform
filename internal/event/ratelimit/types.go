package ratelimit

import (
	"context"
	"notification-platform/internal/domain"
)

const (
	eventName = "request_rate_limited_events"
)

type RequestRateLimitedEvent struct {
	Notifications []domain.Notification `json:"notifications"`
}

//go:generate mockgen -source=./producer.go -package=evtmocks -destination=../mocks/ratelimit_event_producer.mock.go -typed RequestRateLimitedEventProducer
type RequestRateLimitedEventProducer interface {
	Produce(ctx context.Context, evt RequestRateLimitedEvent) error
}
