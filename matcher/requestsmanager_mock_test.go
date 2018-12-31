// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/SamsungSLAV/boruta/matcher (interfaces: RequestsManager)

package matcher

import (
	boruta "github.com/SamsungSLAV/boruta"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
	time "time"
)

// MockRequestsManager is a mock of RequestsManager interface
type MockRequestsManager struct {
	ctrl     *gomock.Controller
	recorder *MockRequestsManagerMockRecorder
}

// MockRequestsManagerMockRecorder is the mock recorder for MockRequestsManager
type MockRequestsManagerMockRecorder struct {
	mock *MockRequestsManager
}

// NewMockRequestsManager creates a new mock instance
func NewMockRequestsManager(ctrl *gomock.Controller) *MockRequestsManager {
	mock := &MockRequestsManager{ctrl: ctrl}
	mock.recorder = &MockRequestsManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRequestsManager) EXPECT() *MockRequestsManagerMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockRequestsManager) Close(arg0 boruta.ReqID) error {
	ret := m.ctrl.Call(m, "Close", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockRequestsManagerMockRecorder) Close(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockRequestsManager)(nil).Close), arg0)
}

// Get mocks base method
func (m *MockRequestsManager) Get(arg0 boruta.ReqID) (boruta.ReqInfo, error) {
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].(boruta.ReqInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockRequestsManagerMockRecorder) Get(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockRequestsManager)(nil).Get), arg0)
}

// InitIteration mocks base method
func (m *MockRequestsManager) InitIteration() error {
	ret := m.ctrl.Call(m, "InitIteration")
	ret0, _ := ret[0].(error)
	return ret0
}

// InitIteration indicates an expected call of InitIteration
func (mr *MockRequestsManagerMockRecorder) InitIteration() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitIteration", reflect.TypeOf((*MockRequestsManager)(nil).InitIteration))
}

// Next mocks base method
func (m *MockRequestsManager) Next() (boruta.ReqID, bool) {
	ret := m.ctrl.Call(m, "Next")
	ret0, _ := ret[0].(boruta.ReqID)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Next indicates an expected call of Next
func (mr *MockRequestsManagerMockRecorder) Next() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockRequestsManager)(nil).Next))
}

// Run mocks base method
func (m *MockRequestsManager) Run(arg0 boruta.ReqID, arg1 boruta.WorkerUUID) error {
	ret := m.ctrl.Call(m, "Run", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run
func (mr *MockRequestsManagerMockRecorder) Run(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockRequestsManager)(nil).Run), arg0, arg1)
}

// TerminateIteration mocks base method
func (m *MockRequestsManager) TerminateIteration() {
	m.ctrl.Call(m, "TerminateIteration")
}

// TerminateIteration indicates an expected call of TerminateIteration
func (mr *MockRequestsManagerMockRecorder) TerminateIteration() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TerminateIteration", reflect.TypeOf((*MockRequestsManager)(nil).TerminateIteration))
}

// Timeout mocks base method
func (m *MockRequestsManager) Timeout(arg0 boruta.ReqID) error {
	ret := m.ctrl.Call(m, "Timeout", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Timeout indicates an expected call of Timeout
func (mr *MockRequestsManagerMockRecorder) Timeout(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Timeout", reflect.TypeOf((*MockRequestsManager)(nil).Timeout), arg0)
}

// VerifyIfReady mocks base method
func (m *MockRequestsManager) VerifyIfReady(arg0 boruta.ReqID, arg1 time.Time) bool {
	ret := m.ctrl.Call(m, "VerifyIfReady", arg0, arg1)
	ret0, _ := ret[0].(bool)
	return ret0
}

// VerifyIfReady indicates an expected call of VerifyIfReady
func (mr *MockRequestsManagerMockRecorder) VerifyIfReady(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyIfReady", reflect.TypeOf((*MockRequestsManager)(nil).VerifyIfReady), arg0, arg1)
}