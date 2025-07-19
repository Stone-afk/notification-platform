package registry

import (
	"context"
	"io"
)

type Registry interface {
	Register(ctx context.Context, si ServiceInstance) error
	UnRegister(ctx context.Context, si ServiceInstance) error

	ListServices(ctx context.Context, name string) ([]ServiceInstance, error)
	Subscribe(name string) <-chan Event

	io.Closer
}

type ServiceInstance struct {
	Name        string
	Address     string
	ReadWeight  int32
	WriteWeight int32
	Group       string
}

type EventType int

const (
	EventTypeUnknown EventType = iota
	EventTypeAdd
	EventTypeDelete
)

func (e EventType) IsAdd() bool {
	return e == EventTypeAdd
}

func (e EventType) IsDelete() bool {
	return e == EventTypeDelete
}

type Event struct {
	Error    error
	Type     EventType
	Instance ServiceInstance
}
