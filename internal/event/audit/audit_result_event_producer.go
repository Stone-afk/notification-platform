package audit

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"notification-platform/internal/pkg/mqx"
)

func NewResultCallbackEventProducer(producer *kafka.Producer) (ResultCallbackEventProducer, error) {
	return mqx.NewGeneralProducer[CallbackResultEvent](producer, eventName)
}
