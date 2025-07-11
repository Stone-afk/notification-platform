// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: config/v1/config.proto

package configv1

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// RetryConfig represents retry policy configuration
type RetryConfig struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	MaxAttempts       int32                  `protobuf:"varint,1,opt,name=max_attempts,json=maxAttempts,proto3" json:"max_attempts,omitempty"`
	InitialBackoffMs  int32                  `protobuf:"varint,2,opt,name=initial_backoff_ms,json=initialBackoffMs,proto3" json:"initial_backoff_ms,omitempty"`
	MaxBackoffMs      int32                  `protobuf:"varint,3,opt,name=max_backoff_ms,json=maxBackoffMs,proto3" json:"max_backoff_ms,omitempty"`
	BackoffMultiplier float64                `protobuf:"fixed64,4,opt,name=backoff_multiplier,json=backoffMultiplier,proto3" json:"backoff_multiplier,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *RetryConfig) Reset() {
	*x = RetryConfig{}
	mi := &file_config_v1_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RetryConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RetryConfig) ProtoMessage() {}

func (x *RetryConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RetryConfig.ProtoReflect.Descriptor instead.
func (*RetryConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{0}
}

func (x *RetryConfig) GetMaxAttempts() int32 {
	if x != nil {
		return x.MaxAttempts
	}
	return 0
}

func (x *RetryConfig) GetInitialBackoffMs() int32 {
	if x != nil {
		return x.InitialBackoffMs
	}
	return 0
}

func (x *RetryConfig) GetMaxBackoffMs() int32 {
	if x != nil {
		return x.MaxBackoffMs
	}
	return 0
}

func (x *RetryConfig) GetBackoffMultiplier() float64 {
	if x != nil {
		return x.BackoffMultiplier
	}
	return 0
}

// ChannelItem represents a notification channel with priority settings
type ChannelItem struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Channel       string                 `protobuf:"bytes,1,opt,name=channel,proto3" json:"channel,omitempty"`
	Priority      int32                  `protobuf:"varint,2,opt,name=priority,proto3" json:"priority,omitempty"`
	Enabled       bool                   `protobuf:"varint,3,opt,name=enabled,proto3" json:"enabled,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChannelItem) Reset() {
	*x = ChannelItem{}
	mi := &file_config_v1_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChannelItem) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChannelItem) ProtoMessage() {}

func (x *ChannelItem) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChannelItem.ProtoReflect.Descriptor instead.
func (*ChannelItem) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{1}
}

func (x *ChannelItem) GetChannel() string {
	if x != nil {
		return x.Channel
	}
	return ""
}

func (x *ChannelItem) GetPriority() int32 {
	if x != nil {
		return x.Priority
	}
	return 0
}

func (x *ChannelItem) GetEnabled() bool {
	if x != nil {
		return x.Enabled
	}
	return false
}

// ChannelConfig represents channel configuration
type ChannelConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Channels      []*ChannelItem         `protobuf:"bytes,1,rep,name=channels,proto3" json:"channels,omitempty"`
	RetryPolicy   *RetryConfig           `protobuf:"bytes,2,opt,name=retry_policy,json=retryPolicy,proto3" json:"retry_policy,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChannelConfig) Reset() {
	*x = ChannelConfig{}
	mi := &file_config_v1_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChannelConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChannelConfig) ProtoMessage() {}

func (x *ChannelConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChannelConfig.ProtoReflect.Descriptor instead.
func (*ChannelConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{2}
}

func (x *ChannelConfig) GetChannels() []*ChannelItem {
	if x != nil {
		return x.Channels
	}
	return nil
}

func (x *ChannelConfig) GetRetryPolicy() *RetryConfig {
	if x != nil {
		return x.RetryPolicy
	}
	return nil
}

// TxnConfig represents transaction configuration
type TxnConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ServiceName   string                 `protobuf:"bytes,1,opt,name=service_name,json=serviceName,proto3" json:"service_name,omitempty"`
	InitialDelay  int32                  `protobuf:"varint,2,opt,name=initial_delay,json=initialDelay,proto3" json:"initial_delay,omitempty"`
	RetryPolicy   *RetryConfig           `protobuf:"bytes,3,opt,name=retry_policy,json=retryPolicy,proto3" json:"retry_policy,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TxnConfig) Reset() {
	*x = TxnConfig{}
	mi := &file_config_v1_config_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TxnConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TxnConfig) ProtoMessage() {}

func (x *TxnConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TxnConfig.ProtoReflect.Descriptor instead.
func (*TxnConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{3}
}

func (x *TxnConfig) GetServiceName() string {
	if x != nil {
		return x.ServiceName
	}
	return ""
}

func (x *TxnConfig) GetInitialDelay() int32 {
	if x != nil {
		return x.InitialDelay
	}
	return 0
}

func (x *TxnConfig) GetRetryPolicy() *RetryConfig {
	if x != nil {
		return x.RetryPolicy
	}
	return nil
}

// MonthlyConfig represents monthly quotas for different channels
type MonthlyConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Sms           int32                  `protobuf:"varint,1,opt,name=sms,proto3" json:"sms,omitempty"`
	Email         int32                  `protobuf:"varint,2,opt,name=email,proto3" json:"email,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MonthlyConfig) Reset() {
	*x = MonthlyConfig{}
	mi := &file_config_v1_config_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MonthlyConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MonthlyConfig) ProtoMessage() {}

func (x *MonthlyConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MonthlyConfig.ProtoReflect.Descriptor instead.
func (*MonthlyConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{4}
}

func (x *MonthlyConfig) GetSms() int32 {
	if x != nil {
		return x.Sms
	}
	return 0
}

func (x *MonthlyConfig) GetEmail() int32 {
	if x != nil {
		return x.Email
	}
	return 0
}

// QuotaConfig represents quota configuration
type QuotaConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Monthly       *MonthlyConfig         `protobuf:"bytes,1,opt,name=monthly,proto3" json:"monthly,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *QuotaConfig) Reset() {
	*x = QuotaConfig{}
	mi := &file_config_v1_config_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *QuotaConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuotaConfig) ProtoMessage() {}

func (x *QuotaConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuotaConfig.ProtoReflect.Descriptor instead.
func (*QuotaConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{5}
}

func (x *QuotaConfig) GetMonthly() *MonthlyConfig {
	if x != nil {
		return x.Monthly
	}
	return nil
}

// CallbackConfig represents callback configuration
type CallbackConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ServiceName   string                 `protobuf:"bytes,1,opt,name=service_name,json=serviceName,proto3" json:"service_name,omitempty"`
	RetryPolicy   *RetryConfig           `protobuf:"bytes,2,opt,name=retry_policy,json=retryPolicy,proto3" json:"retry_policy,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CallbackConfig) Reset() {
	*x = CallbackConfig{}
	mi := &file_config_v1_config_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CallbackConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CallbackConfig) ProtoMessage() {}

func (x *CallbackConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CallbackConfig.ProtoReflect.Descriptor instead.
func (*CallbackConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{6}
}

func (x *CallbackConfig) GetServiceName() string {
	if x != nil {
		return x.ServiceName
	}
	return ""
}

func (x *CallbackConfig) GetRetryPolicy() *RetryConfig {
	if x != nil {
		return x.RetryPolicy
	}
	return nil
}

// BusinessConfig represents the configuration for a business entity
type BusinessConfig struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	OwnerId        int64                  `protobuf:"varint,1,opt,name=owner_id,json=ownerId,proto3" json:"owner_id,omitempty"`
	OwnerType      string                 `protobuf:"bytes,2,opt,name=owner_type,json=ownerType,proto3" json:"owner_type,omitempty"`
	ChannelConfig  *ChannelConfig         `protobuf:"bytes,3,opt,name=channel_config,json=channelConfig,proto3" json:"channel_config,omitempty"`
	TxnConfig      *TxnConfig             `protobuf:"bytes,4,opt,name=txn_config,json=txnConfig,proto3" json:"txn_config,omitempty"`
	RateLimit      int32                  `protobuf:"varint,5,opt,name=rate_limit,json=rateLimit,proto3" json:"rate_limit,omitempty"`
	Quota          *QuotaConfig           `protobuf:"bytes,6,opt,name=quota,proto3" json:"quota,omitempty"`
	CallbackConfig *CallbackConfig        `protobuf:"bytes,7,opt,name=callback_config,json=callbackConfig,proto3" json:"callback_config,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *BusinessConfig) Reset() {
	*x = BusinessConfig{}
	mi := &file_config_v1_config_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BusinessConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BusinessConfig) ProtoMessage() {}

func (x *BusinessConfig) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BusinessConfig.ProtoReflect.Descriptor instead.
func (*BusinessConfig) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{7}
}

func (x *BusinessConfig) GetOwnerId() int64 {
	if x != nil {
		return x.OwnerId
	}
	return 0
}

func (x *BusinessConfig) GetOwnerType() string {
	if x != nil {
		return x.OwnerType
	}
	return ""
}

func (x *BusinessConfig) GetChannelConfig() *ChannelConfig {
	if x != nil {
		return x.ChannelConfig
	}
	return nil
}

func (x *BusinessConfig) GetTxnConfig() *TxnConfig {
	if x != nil {
		return x.TxnConfig
	}
	return nil
}

func (x *BusinessConfig) GetRateLimit() int32 {
	if x != nil {
		return x.RateLimit
	}
	return 0
}

func (x *BusinessConfig) GetQuota() *QuotaConfig {
	if x != nil {
		return x.Quota
	}
	return nil
}

func (x *BusinessConfig) GetCallbackConfig() *CallbackConfig {
	if x != nil {
		return x.CallbackConfig
	}
	return nil
}

// GetByIDsRequest represents the request for GetByIDs method
type GetByIDsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Ids           []int64                `protobuf:"varint,1,rep,packed,name=ids,proto3" json:"ids,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetByIDsRequest) Reset() {
	*x = GetByIDsRequest{}
	mi := &file_config_v1_config_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetByIDsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetByIDsRequest) ProtoMessage() {}

func (x *GetByIDsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetByIDsRequest.ProtoReflect.Descriptor instead.
func (*GetByIDsRequest) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{8}
}

func (x *GetByIDsRequest) GetIds() []int64 {
	if x != nil {
		return x.Ids
	}
	return nil
}

// GetByIDsResponse represents the response for GetByIDs method
type GetByIDsResponse struct {
	state         protoimpl.MessageState    `protogen:"open.v1"`
	Configs       map[int64]*BusinessConfig `protobuf:"bytes,1,rep,name=configs,proto3" json:"configs,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetByIDsResponse) Reset() {
	*x = GetByIDsResponse{}
	mi := &file_config_v1_config_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetByIDsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetByIDsResponse) ProtoMessage() {}

func (x *GetByIDsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetByIDsResponse.ProtoReflect.Descriptor instead.
func (*GetByIDsResponse) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{9}
}

func (x *GetByIDsResponse) GetConfigs() map[int64]*BusinessConfig {
	if x != nil {
		return x.Configs
	}
	return nil
}

// GetByIDRequest represents the request for GetByID method
type GetByIDRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            int64                  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetByIDRequest) Reset() {
	*x = GetByIDRequest{}
	mi := &file_config_v1_config_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetByIDRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetByIDRequest) ProtoMessage() {}

func (x *GetByIDRequest) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetByIDRequest.ProtoReflect.Descriptor instead.
func (*GetByIDRequest) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{10}
}

func (x *GetByIDRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

// GetByIDResponse represents the response for GetByID method
type GetByIDResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Config        *BusinessConfig        `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetByIDResponse) Reset() {
	*x = GetByIDResponse{}
	mi := &file_config_v1_config_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetByIDResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetByIDResponse) ProtoMessage() {}

func (x *GetByIDResponse) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetByIDResponse.ProtoReflect.Descriptor instead.
func (*GetByIDResponse) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{11}
}

func (x *GetByIDResponse) GetConfig() *BusinessConfig {
	if x != nil {
		return x.Config
	}
	return nil
}

// DeleteRequest represents the request for Delete method
type DeleteRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            int64                  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	mi := &file_config_v1_config_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRequest) ProtoMessage() {}

func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteRequest.ProtoReflect.Descriptor instead.
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{12}
}

func (x *DeleteRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

// DeleteResponse represents the response for Delete method
type DeleteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteResponse) Reset() {
	*x = DeleteResponse{}
	mi := &file_config_v1_config_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteResponse) ProtoMessage() {}

func (x *DeleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteResponse.ProtoReflect.Descriptor instead.
func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{13}
}

func (x *DeleteResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

// SaveConfigRequest represents the request for SaveConfig method
type SaveConfigRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Config        *BusinessConfig        `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveConfigRequest) Reset() {
	*x = SaveConfigRequest{}
	mi := &file_config_v1_config_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveConfigRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveConfigRequest) ProtoMessage() {}

func (x *SaveConfigRequest) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaveConfigRequest.ProtoReflect.Descriptor instead.
func (*SaveConfigRequest) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{14}
}

func (x *SaveConfigRequest) GetConfig() *BusinessConfig {
	if x != nil {
		return x.Config
	}
	return nil
}

// SaveConfigResponse represents the response for SaveConfig method
type SaveConfigResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveConfigResponse) Reset() {
	*x = SaveConfigResponse{}
	mi := &file_config_v1_config_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveConfigResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveConfigResponse) ProtoMessage() {}

func (x *SaveConfigResponse) ProtoReflect() protoreflect.Message {
	mi := &file_config_v1_config_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaveConfigResponse.ProtoReflect.Descriptor instead.
func (*SaveConfigResponse) Descriptor() ([]byte, []int) {
	return file_config_v1_config_proto_rawDescGZIP(), []int{15}
}

func (x *SaveConfigResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

var File_config_v1_config_proto protoreflect.FileDescriptor

const file_config_v1_config_proto_rawDesc = "" +
	"\n" +
	"\x16config/v1/config.proto\x12\tconfig.v1\"\xb3\x01\n" +
	"\vRetryConfig\x12!\n" +
	"\fmax_attempts\x18\x01 \x01(\x05R\vmaxAttempts\x12,\n" +
	"\x12initial_backoff_ms\x18\x02 \x01(\x05R\x10initialBackoffMs\x12$\n" +
	"\x0emax_backoff_ms\x18\x03 \x01(\x05R\fmaxBackoffMs\x12-\n" +
	"\x12backoff_multiplier\x18\x04 \x01(\x01R\x11backoffMultiplier\"]\n" +
	"\vChannelItem\x12\x18\n" +
	"\achannel\x18\x01 \x01(\tR\achannel\x12\x1a\n" +
	"\bpriority\x18\x02 \x01(\x05R\bpriority\x12\x18\n" +
	"\aenabled\x18\x03 \x01(\bR\aenabled\"~\n" +
	"\rChannelConfig\x122\n" +
	"\bchannels\x18\x01 \x03(\v2\x16.config.v1.ChannelItemR\bchannels\x129\n" +
	"\fretry_policy\x18\x02 \x01(\v2\x16.config.v1.RetryConfigR\vretryPolicy\"\x8e\x01\n" +
	"\tTxnConfig\x12!\n" +
	"\fservice_name\x18\x01 \x01(\tR\vserviceName\x12#\n" +
	"\rinitial_delay\x18\x02 \x01(\x05R\finitialDelay\x129\n" +
	"\fretry_policy\x18\x03 \x01(\v2\x16.config.v1.RetryConfigR\vretryPolicy\"7\n" +
	"\rMonthlyConfig\x12\x10\n" +
	"\x03sms\x18\x01 \x01(\x05R\x03sms\x12\x14\n" +
	"\x05email\x18\x02 \x01(\x05R\x05email\"A\n" +
	"\vQuotaConfig\x122\n" +
	"\amonthly\x18\x01 \x01(\v2\x18.config.v1.MonthlyConfigR\amonthly\"n\n" +
	"\x0eCallbackConfig\x12!\n" +
	"\fservice_name\x18\x01 \x01(\tR\vserviceName\x129\n" +
	"\fretry_policy\x18\x02 \x01(\v2\x16.config.v1.RetryConfigR\vretryPolicy\"\xd1\x02\n" +
	"\x0eBusinessConfig\x12\x19\n" +
	"\bowner_id\x18\x01 \x01(\x03R\aownerId\x12\x1d\n" +
	"\n" +
	"owner_type\x18\x02 \x01(\tR\townerType\x12?\n" +
	"\x0echannel_config\x18\x03 \x01(\v2\x18.config.v1.ChannelConfigR\rchannelConfig\x123\n" +
	"\n" +
	"txn_config\x18\x04 \x01(\v2\x14.config.v1.TxnConfigR\ttxnConfig\x12\x1d\n" +
	"\n" +
	"rate_limit\x18\x05 \x01(\x05R\trateLimit\x12,\n" +
	"\x05quota\x18\x06 \x01(\v2\x16.config.v1.QuotaConfigR\x05quota\x12B\n" +
	"\x0fcallback_config\x18\a \x01(\v2\x19.config.v1.CallbackConfigR\x0ecallbackConfig\"#\n" +
	"\x0fGetByIDsRequest\x12\x10\n" +
	"\x03ids\x18\x01 \x03(\x03R\x03ids\"\xad\x01\n" +
	"\x10GetByIDsResponse\x12B\n" +
	"\aconfigs\x18\x01 \x03(\v2(.config.v1.GetByIDsResponse.ConfigsEntryR\aconfigs\x1aU\n" +
	"\fConfigsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\x03R\x03key\x12/\n" +
	"\x05value\x18\x02 \x01(\v2\x19.config.v1.BusinessConfigR\x05value:\x028\x01\" \n" +
	"\x0eGetByIDRequest\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\x03R\x02id\"D\n" +
	"\x0fGetByIDResponse\x121\n" +
	"\x06config\x18\x01 \x01(\v2\x19.config.v1.BusinessConfigR\x06config\"\x1f\n" +
	"\rDeleteRequest\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\x03R\x02id\"*\n" +
	"\x0eDeleteResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\"F\n" +
	"\x11SaveConfigRequest\x121\n" +
	"\x06config\x18\x01 \x01(\v2\x19.config.v1.BusinessConfigR\x06config\".\n" +
	"\x12SaveConfigResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess2\xb0\x02\n" +
	"\x15BusinessConfigService\x12E\n" +
	"\bGetByIDs\x12\x1a.config.v1.GetByIDsRequest\x1a\x1b.config.v1.GetByIDsResponse\"\x00\x12B\n" +
	"\aGetByID\x12\x19.config.v1.GetByIDRequest\x1a\x1a.config.v1.GetByIDResponse\"\x00\x12?\n" +
	"\x06Delete\x12\x18.config.v1.DeleteRequest\x1a\x19.config.v1.DeleteResponse\"\x00\x12K\n" +
	"\n" +
	"SaveConfig\x12\x1c.config.v1.SaveConfigRequest\x1a\x1d.config.v1.SaveConfigResponse\"\x00B\xab\x01\n" +
	"\rcom.config.v1B\vConfigProtoP\x01ZHnotification-platform/api/proto/gen/config/v1;configv1\xa2\x02\x03CXX\xaa\x02\tConfig.V1\xca\x02\tConfig\\V1\xe2\x02\x15Config\\V1\\GPBMetadata\xea\x02\n" +
	"Config::V1b\x06proto3"

var (
	file_config_v1_config_proto_rawDescOnce sync.Once
	file_config_v1_config_proto_rawDescData []byte
)

func file_config_v1_config_proto_rawDescGZIP() []byte {
	file_config_v1_config_proto_rawDescOnce.Do(func() {
		file_config_v1_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_config_v1_config_proto_rawDesc), len(file_config_v1_config_proto_rawDesc)))
	})
	return file_config_v1_config_proto_rawDescData
}

var (
	file_config_v1_config_proto_msgTypes = make([]protoimpl.MessageInfo, 17)
	file_config_v1_config_proto_goTypes  = []any{
		(*RetryConfig)(nil),        // 0: config.v1.RetryConfig
		(*ChannelItem)(nil),        // 1: config.v1.ChannelItem
		(*ChannelConfig)(nil),      // 2: config.v1.ChannelConfig
		(*TxnConfig)(nil),          // 3: config.v1.TxnConfig
		(*MonthlyConfig)(nil),      // 4: config.v1.MonthlyConfig
		(*QuotaConfig)(nil),        // 5: config.v1.QuotaConfig
		(*CallbackConfig)(nil),     // 6: config.v1.CallbackConfig
		(*BusinessConfig)(nil),     // 7: config.v1.BusinessConfig
		(*GetByIDsRequest)(nil),    // 8: config.v1.GetByIDsRequest
		(*GetByIDsResponse)(nil),   // 9: config.v1.GetByIDsResponse
		(*GetByIDRequest)(nil),     // 10: config.v1.GetByIDRequest
		(*GetByIDResponse)(nil),    // 11: config.v1.GetByIDResponse
		(*DeleteRequest)(nil),      // 12: config.v1.DeleteRequest
		(*DeleteResponse)(nil),     // 13: config.v1.DeleteResponse
		(*SaveConfigRequest)(nil),  // 14: config.v1.SaveConfigRequest
		(*SaveConfigResponse)(nil), // 15: config.v1.SaveConfigResponse
		nil,                        // 16: config.v1.GetByIDsResponse.ConfigsEntry
	}
)

var file_config_v1_config_proto_depIdxs = []int32{
	1,  // 0: config.v1.ChannelConfig.channels:type_name -> config.v1.ChannelItem
	0,  // 1: config.v1.ChannelConfig.retry_policy:type_name -> config.v1.RetryConfig
	0,  // 2: config.v1.TxnConfig.retry_policy:type_name -> config.v1.RetryConfig
	4,  // 3: config.v1.QuotaConfig.monthly:type_name -> config.v1.MonthlyConfig
	0,  // 4: config.v1.CallbackConfig.retry_policy:type_name -> config.v1.RetryConfig
	2,  // 5: config.v1.BusinessConfig.channel_config:type_name -> config.v1.ChannelConfig
	3,  // 6: config.v1.BusinessConfig.txn_config:type_name -> config.v1.TxnConfig
	5,  // 7: config.v1.BusinessConfig.quota:type_name -> config.v1.QuotaConfig
	6,  // 8: config.v1.BusinessConfig.callback_config:type_name -> config.v1.CallbackConfig
	16, // 9: config.v1.GetByIDsResponse.configs:type_name -> config.v1.GetByIDsResponse.ConfigsEntry
	7,  // 10: config.v1.GetByIDResponse.config:type_name -> config.v1.BusinessConfig
	7,  // 11: config.v1.SaveConfigRequest.config:type_name -> config.v1.BusinessConfig
	7,  // 12: config.v1.GetByIDsResponse.ConfigsEntry.value:type_name -> config.v1.BusinessConfig
	8,  // 13: config.v1.BusinessConfigService.GetByIDs:input_type -> config.v1.GetByIDsRequest
	10, // 14: config.v1.BusinessConfigService.GetByID:input_type -> config.v1.GetByIDRequest
	12, // 15: config.v1.BusinessConfigService.Delete:input_type -> config.v1.DeleteRequest
	14, // 16: config.v1.BusinessConfigService.SaveConfig:input_type -> config.v1.SaveConfigRequest
	9,  // 17: config.v1.BusinessConfigService.GetByIDs:output_type -> config.v1.GetByIDsResponse
	11, // 18: config.v1.BusinessConfigService.GetByID:output_type -> config.v1.GetByIDResponse
	13, // 19: config.v1.BusinessConfigService.Delete:output_type -> config.v1.DeleteResponse
	15, // 20: config.v1.BusinessConfigService.SaveConfig:output_type -> config.v1.SaveConfigResponse
	17, // [17:21] is the sub-list for method output_type
	13, // [13:17] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_config_v1_config_proto_init() }
func file_config_v1_config_proto_init() {
	if File_config_v1_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_config_v1_config_proto_rawDesc), len(file_config_v1_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   17,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_config_v1_config_proto_goTypes,
		DependencyIndexes: file_config_v1_config_proto_depIdxs,
		MessageInfos:      file_config_v1_config_proto_msgTypes,
	}.Build()
	File_config_v1_config_proto = out.File
	file_config_v1_config_proto_goTypes = nil
	file_config_v1_config_proto_depIdxs = nil
}
