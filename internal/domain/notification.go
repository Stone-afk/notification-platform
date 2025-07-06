package domain

import "time"

// SendStatus 通知状态
type SendStatus string

const (
	SendStatusPrepare   SendStatus = "PREPARE"   // 准备中
	SendStatusCanceled  SendStatus = "CANCELED"  // 已取消
	SendStatusPending   SendStatus = "PENDING"   // 待发送
	SendStatusSending   SendStatus = "SENDING"   // 待发送
	SendStatusSucceeded SendStatus = "SUCCEEDED" // 发送成功
	SendStatusFailed    SendStatus = "FAILED"    // 发送失败
)

func (s SendStatus) String() string {
	return string(s)
}

type Template struct {
	ID        int64             `json:"id"`        // 模板ID
	VersionID int64             `json:"versionId"` // 版本ID
	Params    map[string]string `json:"params"`    // 渲染模版时使用的参数

	// 只做版本兼容演示代码用，其余忽略
	Version string `json:"version"`
}

// Notification 通知领域模型
type Notification struct {
	ID                 uint64             `json:"id"`             // 通知唯一标识
	BizID              int64              `json:"bizId"`          // 业务唯一标识
	Key                string             `json:"key"`            // 业务内唯一标识
	Receivers          []string           `json:"receivers"`      // 接收者(手机/邮箱/用户ID)
	Channel            Channel            `json:"channel"`        // 发送渠道
	Template           Template           `json:"template"`       // 关联的模版
	Status             SendStatus         `json:"status"`         // 发送状态
	ScheduledSTime     time.Time          `json:"scheduledSTime"` // 计划发送开始时间
	ScheduledETime     time.Time          `json:"scheduledETime"` // 计划发送结束时间
	Version            int                `json:"version"`        // 版本号
	SendStrategyConfig SendStrategyConfig `json:"sendStrategyConfig"`
}

func (n *Notification) SetSendTime() {
	stime, etime := n.SendStrategyConfig.SendTimeWindow()
	n.ScheduledSTime = stime
	n.ScheduledETime = etime
}

func (n *Notification) IsImmediate() bool {
	return n.SendStrategyConfig.Type == SendStrategyImmediate
}
