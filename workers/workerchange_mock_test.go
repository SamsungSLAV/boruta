// Code generated by MockGen. DO NOT EDIT.
// Source: git.tizen.org/tools/boruta/workers (interfaces: WorkerChange)

package workers

import (
	boruta "git.tizen.org/tools/boruta"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockWorkerChange is a mock of WorkerChange interface
type MockWorkerChange struct {
	ctrl     *gomock.Controller
	recorder *MockWorkerChangeMockRecorder
}

// MockWorkerChangeMockRecorder is the mock recorder for MockWorkerChange
type MockWorkerChangeMockRecorder struct {
	mock *MockWorkerChange
}

// NewMockWorkerChange creates a new mock instance
func NewMockWorkerChange(ctrl *gomock.Controller) *MockWorkerChange {
	mock := &MockWorkerChange{ctrl: ctrl}
	mock.recorder = &MockWorkerChangeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWorkerChange) EXPECT() *MockWorkerChangeMockRecorder {
	return m.recorder
}

// OnWorkerFail mocks base method
func (m *MockWorkerChange) OnWorkerFail(arg0 boruta.WorkerUUID) {
	m.ctrl.Call(m, "OnWorkerFail", arg0)
}

// OnWorkerFail indicates an expected call of OnWorkerFail
func (mr *MockWorkerChangeMockRecorder) OnWorkerFail(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnWorkerFail", reflect.TypeOf((*MockWorkerChange)(nil).OnWorkerFail), arg0)
}

// OnWorkerIdle mocks base method
func (m *MockWorkerChange) OnWorkerIdle(arg0 boruta.WorkerUUID) {
	m.ctrl.Call(m, "OnWorkerIdle", arg0)
}

// OnWorkerIdle indicates an expected call of OnWorkerIdle
func (mr *MockWorkerChangeMockRecorder) OnWorkerIdle(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnWorkerIdle", reflect.TypeOf((*MockWorkerChange)(nil).OnWorkerIdle), arg0)
}
