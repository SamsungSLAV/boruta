// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/SamsungSLAV/boruta/matcher (interfaces: JobsManager)

package requests

import (
	boruta "github.com/SamsungSLAV/boruta"
	workers "github.com/SamsungSLAV/boruta/workers"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockJobsManager is a mock of JobsManager interface
type MockJobsManager struct {
	ctrl     *gomock.Controller
	recorder *MockJobsManagerMockRecorder
}

// MockJobsManagerMockRecorder is the mock recorder for MockJobsManager
type MockJobsManagerMockRecorder struct {
	mock *MockJobsManager
}

// NewMockJobsManager creates a new mock instance
func NewMockJobsManager(ctrl *gomock.Controller) *MockJobsManager {
	mock := &MockJobsManager{ctrl: ctrl}
	mock.recorder = &MockJobsManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockJobsManager) EXPECT() *MockJobsManagerMockRecorder {
	return m.recorder
}

// Create mocks base method
func (m *MockJobsManager) Create(arg0 boruta.ReqID, arg1 boruta.WorkerUUID) error {
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockJobsManagerMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockJobsManager)(nil).Create), arg0, arg1)
}

// Finish mocks base method
func (m *MockJobsManager) Finish(arg0 boruta.WorkerUUID, arg1 bool) error {
	ret := m.ctrl.Call(m, "Finish", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Finish indicates an expected call of Finish
func (mr *MockJobsManagerMockRecorder) Finish(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Finish", reflect.TypeOf((*MockJobsManager)(nil).Finish), arg0, arg1)
}

// Get mocks base method
func (m *MockJobsManager) Get(arg0 boruta.WorkerUUID) (*workers.Job, error) {
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(*workers.Job)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockJobsManagerMockRecorder) Get(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockJobsManager)(nil).Get), arg0)
}
