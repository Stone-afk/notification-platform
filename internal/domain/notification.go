package domain

import (
	"encoding/json"
	"fmt"
	"notification-platform/internal/errs"
	"time"
)

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

// ReplaceAsyncImmediate 如果是是立刻发送，就修改为默认的策略
func (n *Notification) ReplaceAsyncImmediate() {
	if n.IsImmediate() {
		n.SendStrategyConfig.DeadlineTime = time.Now().Add(time.Minute)
		n.SendStrategyConfig.Type = SendStrategyDeadline
	}
}

func (n *Notification) Validate() error {
	if n.BizID <= 0 {
		return fmt.Errorf("%w: BizID = %d", errs.ErrInvalidParameter, n.BizID)
	}

	if n.Key == "" {
		return fmt.Errorf("%w: Key = %q", errs.ErrInvalidParameter, n.Key)
	}

	if len(n.Receivers) == 0 {
		return fmt.Errorf("%w: Receivers= %v", errs.ErrInvalidParameter, n.Receivers)
	}

	if !n.Channel.IsValid() {
		return fmt.Errorf("%w: Channel = %q", errs.ErrInvalidParameter, n.Channel)
	}

	if n.Template.ID <= 0 {
		return fmt.Errorf("%w: Template.ID = %d", errs.ErrInvalidParameter, n.Template.ID)
	}

	if n.Template.VersionID <= 0 {
		return fmt.Errorf("%w: Template.VersionID = %d", errs.ErrInvalidParameter, n.Template.VersionID)
	}

	if len(n.Template.Params) == 0 {
		return fmt.Errorf("%w: Template.Params = %q", errs.ErrInvalidParameter, n.Template.Params)
	}

	if err := n.SendStrategyConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (n *Notification) IsValidBizID() error {
	if n.BizID <= 0 {
		return fmt.Errorf("%w: BizID = %d", errs.ErrInvalidParameter, n.BizID)
	}
	return nil
}

func (n *Notification) MarshalReceivers() (string, error) {
	return n.marshal(n.Receivers)
}

func (n *Notification) MarshalTemplateParams() (string, error) {
	return n.marshal(n.Template.Params)
}

func (n *Notification) marshal(v any) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
