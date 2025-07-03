package domain

type CallbackLogStatus string

const (
	CallbackLogStatusInit    CallbackLogStatus = "INIT"
	CallbackLogStatusPending CallbackLogStatus = "PENDING"
	CallbackLogStatusSuccess CallbackLogStatus = "SUCCEEDED"
	CallbackLogStatusFailed  CallbackLogStatus = "FAILED"
)

func (c CallbackLogStatus) String() string {
	return string(c)
}

type CallbackLog struct {
	ID            int64
	Notification  Notification
	RetryCount    int32
	NextRetryTime int64
	Status        CallbackLogStatus
}
