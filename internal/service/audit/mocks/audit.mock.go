// Code generated by MockGen. DO NOT EDIT.
// Source: ./audit.go
//
// Generated by this command:
//
//	mockgen -source=./audit.go -destination=./mocks/audit.mock.go -package=auditmocks -typed Service
//

// Package auditmocks is a generated GoMock package.
package auditmocks

import (
	context "context"
	reflect "reflect"

	domain "notification-platform/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
	isgomock struct{}
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CreateAudit mocks base method.
func (m *MockService) CreateAudit(ctx context.Context, req domain.Audit) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAudit", ctx, req)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAudit indicates an expected call of CreateAudit.
func (mr *MockServiceMockRecorder) CreateAudit(ctx, req any) *MockServiceCreateAuditCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAudit", reflect.TypeOf((*MockService)(nil).CreateAudit), ctx, req)
	return &MockServiceCreateAuditCall{Call: call}
}

// MockServiceCreateAuditCall wrap *gomock.Call
type MockServiceCreateAuditCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceCreateAuditCall) Return(arg0 int64, arg1 error) *MockServiceCreateAuditCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceCreateAuditCall) Do(f func(context.Context, domain.Audit) (int64, error)) *MockServiceCreateAuditCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceCreateAuditCall) DoAndReturn(f func(context.Context, domain.Audit) (int64, error)) *MockServiceCreateAuditCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
