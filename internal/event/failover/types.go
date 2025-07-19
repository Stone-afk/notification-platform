package failover

import "context"

const (
	FailoverTopic = "fail_over_event"
)

type ConnPoolEvent struct {
	SQL  string `json:"sql"`
	Args []any  `json:"args"`
}

//go:generate mockgen -source=./producer.go -package=evtmocks -destination=../mocks/conn_pool_event_producer.mock.go -typed ConnPoolEventProducer
type ConnPoolEventProducer interface {
	Produce(ctx context.Context, evt ConnPoolEvent) error
}
