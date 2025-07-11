// Code generated by MockGen. DO NOT EDIT.
// Source: ./manage.go
//
// Generated by this command:
//
//	mockgen -source=./manage.go -destination=../mocks/manage.mock.go -package=templatemocks -typed ChannelTemplateService
//

// Package templatemocks is a generated GoMock package.
package templatemocks

import (
	context "context"
	reflect "reflect"

	domain "notification-platform/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

// MockChannelTemplateService is a mock of ChannelTemplateService interface.
type MockChannelTemplateService struct {
	ctrl     *gomock.Controller
	recorder *MockChannelTemplateServiceMockRecorder
	isgomock struct{}
}

// MockChannelTemplateServiceMockRecorder is the mock recorder for MockChannelTemplateService.
type MockChannelTemplateServiceMockRecorder struct {
	mock *MockChannelTemplateService
}

// NewMockChannelTemplateService creates a new mock instance.
func NewMockChannelTemplateService(ctrl *gomock.Controller) *MockChannelTemplateService {
	mock := &MockChannelTemplateService{ctrl: ctrl}
	mock.recorder = &MockChannelTemplateServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChannelTemplateService) EXPECT() *MockChannelTemplateServiceMockRecorder {
	return m.recorder
}

// BatchQueryAndUpdateProviderAuditInfo mocks base method.
func (m *MockChannelTemplateService) BatchQueryAndUpdateProviderAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchQueryAndUpdateProviderAuditInfo", ctx, providers)
	ret0, _ := ret[0].(error)
	return ret0
}

// BatchQueryAndUpdateProviderAuditInfo indicates an expected call of BatchQueryAndUpdateProviderAuditInfo.
func (mr *MockChannelTemplateServiceMockRecorder) BatchQueryAndUpdateProviderAuditInfo(ctx, providers any) *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchQueryAndUpdateProviderAuditInfo", reflect.TypeOf((*MockChannelTemplateService)(nil).BatchQueryAndUpdateProviderAuditInfo), ctx, providers)
	return &MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall{Call: call}
}

// MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall wrap *gomock.Call
type MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall) Return(arg0 error) *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall) Do(f func(context.Context, []domain.ChannelTemplateProvider) error) *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall) DoAndReturn(f func(context.Context, []domain.ChannelTemplateProvider) error) *MockChannelTemplateServiceBatchQueryAndUpdateProviderAuditInfoCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// BatchSubmitForProviderReview mocks base method.
func (m *MockChannelTemplateService) BatchSubmitForProviderReview(ctx context.Context, versionID []int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchSubmitForProviderReview", ctx, versionID)
	ret0, _ := ret[0].(error)
	return ret0
}

// BatchSubmitForProviderReview indicates an expected call of BatchSubmitForProviderReview.
func (mr *MockChannelTemplateServiceMockRecorder) BatchSubmitForProviderReview(ctx, versionID any) *MockChannelTemplateServiceBatchSubmitForProviderReviewCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchSubmitForProviderReview", reflect.TypeOf((*MockChannelTemplateService)(nil).BatchSubmitForProviderReview), ctx, versionID)
	return &MockChannelTemplateServiceBatchSubmitForProviderReviewCall{Call: call}
}

// MockChannelTemplateServiceBatchSubmitForProviderReviewCall wrap *gomock.Call
type MockChannelTemplateServiceBatchSubmitForProviderReviewCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceBatchSubmitForProviderReviewCall) Return(arg0 error) *MockChannelTemplateServiceBatchSubmitForProviderReviewCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceBatchSubmitForProviderReviewCall) Do(f func(context.Context, []int64) error) *MockChannelTemplateServiceBatchSubmitForProviderReviewCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceBatchSubmitForProviderReviewCall) DoAndReturn(f func(context.Context, []int64) error) *MockChannelTemplateServiceBatchSubmitForProviderReviewCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// BatchUpdateVersionAuditStatus mocks base method.
func (m *MockChannelTemplateService) BatchUpdateVersionAuditStatus(ctx context.Context, versions []domain.ChannelTemplateVersion) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchUpdateVersionAuditStatus", ctx, versions)
	ret0, _ := ret[0].(error)
	return ret0
}

// BatchUpdateVersionAuditStatus indicates an expected call of BatchUpdateVersionAuditStatus.
func (mr *MockChannelTemplateServiceMockRecorder) BatchUpdateVersionAuditStatus(ctx, versions any) *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchUpdateVersionAuditStatus", reflect.TypeOf((*MockChannelTemplateService)(nil).BatchUpdateVersionAuditStatus), ctx, versions)
	return &MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall{Call: call}
}

// MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall wrap *gomock.Call
type MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall) Return(arg0 error) *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall) Do(f func(context.Context, []domain.ChannelTemplateVersion) error) *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall) DoAndReturn(f func(context.Context, []domain.ChannelTemplateVersion) error) *MockChannelTemplateServiceBatchUpdateVersionAuditStatusCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CreateTemplate mocks base method.
func (m *MockChannelTemplateService) CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTemplate", ctx, template)
	ret0, _ := ret[0].(domain.ChannelTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTemplate indicates an expected call of CreateTemplate.
func (mr *MockChannelTemplateServiceMockRecorder) CreateTemplate(ctx, template any) *MockChannelTemplateServiceCreateTemplateCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTemplate", reflect.TypeOf((*MockChannelTemplateService)(nil).CreateTemplate), ctx, template)
	return &MockChannelTemplateServiceCreateTemplateCall{Call: call}
}

// MockChannelTemplateServiceCreateTemplateCall wrap *gomock.Call
type MockChannelTemplateServiceCreateTemplateCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceCreateTemplateCall) Return(arg0 domain.ChannelTemplate, arg1 error) *MockChannelTemplateServiceCreateTemplateCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceCreateTemplateCall) Do(f func(context.Context, domain.ChannelTemplate) (domain.ChannelTemplate, error)) *MockChannelTemplateServiceCreateTemplateCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceCreateTemplateCall) DoAndReturn(f func(context.Context, domain.ChannelTemplate) (domain.ChannelTemplate, error)) *MockChannelTemplateServiceCreateTemplateCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// ForkVersion mocks base method.
func (m *MockChannelTemplateService) ForkVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ForkVersion", ctx, versionID)
	ret0, _ := ret[0].(domain.ChannelTemplateVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForkVersion indicates an expected call of ForkVersion.
func (mr *MockChannelTemplateServiceMockRecorder) ForkVersion(ctx, versionID any) *MockChannelTemplateServiceForkVersionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForkVersion", reflect.TypeOf((*MockChannelTemplateService)(nil).ForkVersion), ctx, versionID)
	return &MockChannelTemplateServiceForkVersionCall{Call: call}
}

// MockChannelTemplateServiceForkVersionCall wrap *gomock.Call
type MockChannelTemplateServiceForkVersionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceForkVersionCall) Return(arg0 domain.ChannelTemplateVersion, arg1 error) *MockChannelTemplateServiceForkVersionCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceForkVersionCall) Do(f func(context.Context, int64) (domain.ChannelTemplateVersion, error)) *MockChannelTemplateServiceForkVersionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceForkVersionCall) DoAndReturn(f func(context.Context, int64) (domain.ChannelTemplateVersion, error)) *MockChannelTemplateServiceForkVersionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetPendingOrInReviewProviders mocks base method.
func (m *MockChannelTemplateService) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) ([]domain.ChannelTemplateProvider, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPendingOrInReviewProviders", ctx, offset, limit, utime)
	ret0, _ := ret[0].([]domain.ChannelTemplateProvider)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetPendingOrInReviewProviders indicates an expected call of GetPendingOrInReviewProviders.
func (mr *MockChannelTemplateServiceMockRecorder) GetPendingOrInReviewProviders(ctx, offset, limit, utime any) *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPendingOrInReviewProviders", reflect.TypeOf((*MockChannelTemplateService)(nil).GetPendingOrInReviewProviders), ctx, offset, limit, utime)
	return &MockChannelTemplateServiceGetPendingOrInReviewProvidersCall{Call: call}
}

// MockChannelTemplateServiceGetPendingOrInReviewProvidersCall wrap *gomock.Call
type MockChannelTemplateServiceGetPendingOrInReviewProvidersCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall) Return(providers []domain.ChannelTemplateProvider, total int64, err error) *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall {
	c.Call = c.Call.Return(providers, total, err)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall) Do(f func(context.Context, int, int, int64) ([]domain.ChannelTemplateProvider, int64, error)) *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall) DoAndReturn(f func(context.Context, int, int, int64) ([]domain.ChannelTemplateProvider, int64, error)) *MockChannelTemplateServiceGetPendingOrInReviewProvidersCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetTemplateByID mocks base method.
func (m *MockChannelTemplateService) GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplateByID", ctx, templateID)
	ret0, _ := ret[0].(domain.ChannelTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplateByID indicates an expected call of GetTemplateByID.
func (mr *MockChannelTemplateServiceMockRecorder) GetTemplateByID(ctx, templateID any) *MockChannelTemplateServiceGetTemplateByIDCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplateByID", reflect.TypeOf((*MockChannelTemplateService)(nil).GetTemplateByID), ctx, templateID)
	return &MockChannelTemplateServiceGetTemplateByIDCall{Call: call}
}

// MockChannelTemplateServiceGetTemplateByIDCall wrap *gomock.Call
type MockChannelTemplateServiceGetTemplateByIDCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceGetTemplateByIDCall) Return(arg0 domain.ChannelTemplate, arg1 error) *MockChannelTemplateServiceGetTemplateByIDCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceGetTemplateByIDCall) Do(f func(context.Context, int64) (domain.ChannelTemplate, error)) *MockChannelTemplateServiceGetTemplateByIDCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceGetTemplateByIDCall) DoAndReturn(f func(context.Context, int64) (domain.ChannelTemplate, error)) *MockChannelTemplateServiceGetTemplateByIDCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetTemplateByIDAndProviderInfo mocks base method.
func (m *MockChannelTemplateService) GetTemplateByIDAndProviderInfo(ctx context.Context, templateID int64, providerName string, channel domain.Channel) (domain.ChannelTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplateByIDAndProviderInfo", ctx, templateID, providerName, channel)
	ret0, _ := ret[0].(domain.ChannelTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplateByIDAndProviderInfo indicates an expected call of GetTemplateByIDAndProviderInfo.
func (mr *MockChannelTemplateServiceMockRecorder) GetTemplateByIDAndProviderInfo(ctx, templateID, providerName, channel any) *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplateByIDAndProviderInfo", reflect.TypeOf((*MockChannelTemplateService)(nil).GetTemplateByIDAndProviderInfo), ctx, templateID, providerName, channel)
	return &MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall{Call: call}
}

// MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall wrap *gomock.Call
type MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall) Return(arg0 domain.ChannelTemplate, arg1 error) *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall) Do(f func(context.Context, int64, string, domain.Channel) (domain.ChannelTemplate, error)) *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall) DoAndReturn(f func(context.Context, int64, string, domain.Channel) (domain.ChannelTemplate, error)) *MockChannelTemplateServiceGetTemplateByIDAndProviderInfoCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetTemplatesByOwner mocks base method.
func (m *MockChannelTemplateService) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTemplatesByOwner", ctx, ownerID, ownerType)
	ret0, _ := ret[0].([]domain.ChannelTemplate)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTemplatesByOwner indicates an expected call of GetTemplatesByOwner.
func (mr *MockChannelTemplateServiceMockRecorder) GetTemplatesByOwner(ctx, ownerID, ownerType any) *MockChannelTemplateServiceGetTemplatesByOwnerCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTemplatesByOwner", reflect.TypeOf((*MockChannelTemplateService)(nil).GetTemplatesByOwner), ctx, ownerID, ownerType)
	return &MockChannelTemplateServiceGetTemplatesByOwnerCall{Call: call}
}

// MockChannelTemplateServiceGetTemplatesByOwnerCall wrap *gomock.Call
type MockChannelTemplateServiceGetTemplatesByOwnerCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceGetTemplatesByOwnerCall) Return(arg0 []domain.ChannelTemplate, arg1 error) *MockChannelTemplateServiceGetTemplatesByOwnerCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceGetTemplatesByOwnerCall) Do(f func(context.Context, int64, domain.OwnerType) ([]domain.ChannelTemplate, error)) *MockChannelTemplateServiceGetTemplatesByOwnerCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceGetTemplatesByOwnerCall) DoAndReturn(f func(context.Context, int64, domain.OwnerType) ([]domain.ChannelTemplate, error)) *MockChannelTemplateServiceGetTemplatesByOwnerCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// PublishTemplate mocks base method.
func (m *MockChannelTemplateService) PublishTemplate(ctx context.Context, templateID, versionID int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PublishTemplate", ctx, templateID, versionID)
	ret0, _ := ret[0].(error)
	return ret0
}

// PublishTemplate indicates an expected call of PublishTemplate.
func (mr *MockChannelTemplateServiceMockRecorder) PublishTemplate(ctx, templateID, versionID any) *MockChannelTemplateServicePublishTemplateCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PublishTemplate", reflect.TypeOf((*MockChannelTemplateService)(nil).PublishTemplate), ctx, templateID, versionID)
	return &MockChannelTemplateServicePublishTemplateCall{Call: call}
}

// MockChannelTemplateServicePublishTemplateCall wrap *gomock.Call
type MockChannelTemplateServicePublishTemplateCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServicePublishTemplateCall) Return(arg0 error) *MockChannelTemplateServicePublishTemplateCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServicePublishTemplateCall) Do(f func(context.Context, int64, int64) error) *MockChannelTemplateServicePublishTemplateCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServicePublishTemplateCall) DoAndReturn(f func(context.Context, int64, int64) error) *MockChannelTemplateServicePublishTemplateCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SubmitForInternalReview mocks base method.
func (m *MockChannelTemplateService) SubmitForInternalReview(ctx context.Context, versionID int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubmitForInternalReview", ctx, versionID)
	ret0, _ := ret[0].(error)
	return ret0
}

// SubmitForInternalReview indicates an expected call of SubmitForInternalReview.
func (mr *MockChannelTemplateServiceMockRecorder) SubmitForInternalReview(ctx, versionID any) *MockChannelTemplateServiceSubmitForInternalReviewCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubmitForInternalReview", reflect.TypeOf((*MockChannelTemplateService)(nil).SubmitForInternalReview), ctx, versionID)
	return &MockChannelTemplateServiceSubmitForInternalReviewCall{Call: call}
}

// MockChannelTemplateServiceSubmitForInternalReviewCall wrap *gomock.Call
type MockChannelTemplateServiceSubmitForInternalReviewCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceSubmitForInternalReviewCall) Return(arg0 error) *MockChannelTemplateServiceSubmitForInternalReviewCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceSubmitForInternalReviewCall) Do(f func(context.Context, int64) error) *MockChannelTemplateServiceSubmitForInternalReviewCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceSubmitForInternalReviewCall) DoAndReturn(f func(context.Context, int64) error) *MockChannelTemplateServiceSubmitForInternalReviewCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UpdateTemplate mocks base method.
func (m *MockChannelTemplateService) UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTemplate", ctx, template)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateTemplate indicates an expected call of UpdateTemplate.
func (mr *MockChannelTemplateServiceMockRecorder) UpdateTemplate(ctx, template any) *MockChannelTemplateServiceUpdateTemplateCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTemplate", reflect.TypeOf((*MockChannelTemplateService)(nil).UpdateTemplate), ctx, template)
	return &MockChannelTemplateServiceUpdateTemplateCall{Call: call}
}

// MockChannelTemplateServiceUpdateTemplateCall wrap *gomock.Call
type MockChannelTemplateServiceUpdateTemplateCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceUpdateTemplateCall) Return(arg0 error) *MockChannelTemplateServiceUpdateTemplateCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceUpdateTemplateCall) Do(f func(context.Context, domain.ChannelTemplate) error) *MockChannelTemplateServiceUpdateTemplateCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceUpdateTemplateCall) DoAndReturn(f func(context.Context, domain.ChannelTemplate) error) *MockChannelTemplateServiceUpdateTemplateCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// UpdateVersion mocks base method.
func (m *MockChannelTemplateService) UpdateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateVersion", ctx, version)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateVersion indicates an expected call of UpdateVersion.
func (mr *MockChannelTemplateServiceMockRecorder) UpdateVersion(ctx, version any) *MockChannelTemplateServiceUpdateVersionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateVersion", reflect.TypeOf((*MockChannelTemplateService)(nil).UpdateVersion), ctx, version)
	return &MockChannelTemplateServiceUpdateVersionCall{Call: call}
}

// MockChannelTemplateServiceUpdateVersionCall wrap *gomock.Call
type MockChannelTemplateServiceUpdateVersionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockChannelTemplateServiceUpdateVersionCall) Return(arg0 error) *MockChannelTemplateServiceUpdateVersionCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockChannelTemplateServiceUpdateVersionCall) Do(f func(context.Context, domain.ChannelTemplateVersion) error) *MockChannelTemplateServiceUpdateVersionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockChannelTemplateServiceUpdateVersionCall) DoAndReturn(f func(context.Context, domain.ChannelTemplateVersion) error) *MockChannelTemplateServiceUpdateVersionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
