package ratelimit

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"notification-platform/internal/pkg/mqx"
)

func NewRequestRateLimitedEventProducer(producer *kafka.Producer) (RequestRateLimitedEventProducer, error) {
	return NewRequestRateLimitedEventProducerWithTopic(producer, eventName)
}

func NewRequestRateLimitedEventProducerWithTopic(producer *kafka.Producer, topic string) (RequestRateLimitedEventProducer, error) {
	return mqx.NewGeneralProducer[RequestRateLimitedEvent](producer, topic)
}
