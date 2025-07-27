package notification

import "notification-platform/internal/domain"

const (
	EventName = "notification_events"
)

// Event 相对于 EventV1 来说，就是 EventV1 的聚合消息
type Event struct {
	Notifications []domain.Notification `json:"notifications"`
}

// EventV1 最开始可能涉及成这个样子
type EventV1 struct {
	Notification domain.Notification `json:"notification"`
}
