package domain

type Quota struct {
	BizID   int64
	Quota   int32
	Channel Channel
}
