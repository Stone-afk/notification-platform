package notificationv1

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ecodeclub/ekit/slice"
)

func (x *Notification) FindReceivers() []string {
	receivers := x.Receivers
	if x.Receiver != "" {
		receivers = append(receivers, x.Receiver)
	}
	return receivers
}

// CustomValidate 你加方法，可以做很多事情
func (x *Notification) CustomValidate() error {
	switch val := x.Strategy.StrategyType.(type) {
	case *SendStrategy_Delayed:
		// 延迟时间超过 1 小时，你就返回错误
		if time.Duration(val.Delayed.DelaySeconds)*time.Second > time.Hour*24 {
			return errors.New("延迟太久了")
		}
	}
	return nil
}

// ReceiversAsUid 比如说站内信之类，receivers 其实是 uid
func (x *Notification) ReceiversAsUid() ([]int64, error) {
	receivers := x.FindReceivers()
	result := make([]int64, 0, len(receivers))
	for _, r := range receivers {
		val, err := strconv.ParseInt(r, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("必须是数字 %w", err)
		}
		result = append(result, val)
	}
	return result, nil
}

type NotificationCarrier interface {
	GetNotifications() []*Notification
}

func (x *SendNotificationRequest) GetNotifications() []*Notification {
	n := x.GetNotification()
	if n != nil {
		return []*Notification{n}
	}
	return nil
}

func (x *SendNotificationAsyncRequest) GetNotifications() []*Notification {
	n := x.GetNotification()
	if n != nil {
		return []*Notification{n}
	}
	return nil
}

type IdempotencyCarrier interface {
	GetIdempotencyKeys() []string
}

func (x *SendNotificationRequest) GetIdempotencyKeys() []string {
	n := x.GetNotification()
	if n != nil {
		return []string{n.Key}
	}
	return nil
}

func (x *SendNotificationAsyncRequest) GetIdempotencyKeys() []string {
	n := x.GetNotification()
	if n != nil {
		return []string{n.Key}
	}
	return nil
}

func (x *BatchSendNotificationsAsyncRequest) GetIdempotencyKeys() []string {
	if x != nil {
		notifications := x.GetNotifications()
		return slice.Map(notifications, func(idx int, src *Notification) string {
			return src.Key
		})
	}
	return nil
}

func (x *BatchSendNotificationsRequest) GetIdempotencyKeys() []string {
	if x != nil {
		notifications := x.GetNotifications()
		return slice.Map(notifications, func(idx int, src *Notification) string {
			return src.Key
		})
	}
	return nil
}
