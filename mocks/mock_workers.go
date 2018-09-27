// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/SamsungSLAV/boruta (interfaces: Workers,Superviser)

// Package mocks is a generated GoMock package.
package mocks

import (
	boruta "github.com/SamsungSLAV/boruta"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockWorkers is a mock of Workers interface
type MockWorkers struct {
	ctrl     *gomock.Controller
	recorder *MockWorkersMockRecorder
}

// MockWorkersMockRecorder is the mock recorder for MockWorkers
type MockWorkersMockRecorder struct {
	mock *MockWorkers
}

// NewMockWorkers creates a new mock instance
func NewMockWorkers(ctrl *gomock.Controller) *MockWorkers {
	mock := &MockWorkers{ctrl: ctrl}
	mock.recorder = &MockWorkersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWorkers) EXPECT() *MockWorkersMockRecorder {
	return m.recorder
}

// Deregister mocks base method
func (m *MockWorkers) Deregister(arg0 boruta.WorkerUUID) error {
	ret := m.ctrl.Call(m, "Deregister", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Deregister indicates an expected call of Deregister
func (mr *MockWorkersMockRecorder) Deregister(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Deregister", reflect.TypeOf((*MockWorkers)(nil).Deregister), arg0)
}

// GetWorkerInfo mocks base method
func (m *MockWorkers) GetWorkerInfo(arg0 boruta.WorkerUUID) (boruta.WorkerInfo, error) {
	ret := m.ctrl.Call(m, "GetWorkerInfo", arg0)
	ret0, _ := ret[0].(boruta.WorkerInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWorkerInfo indicates an expected call of GetWorkerInfo
func (mr *MockWorkersMockRecorder) GetWorkerInfo(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWorkerInfo", reflect.TypeOf((*MockWorkers)(nil).GetWorkerInfo), arg0)
}

// ListWorkers mocks base method
func (m *MockWorkers) ListWorkers(arg0 boruta.Groups, arg1 boruta.Capabilities, arg2 *boruta.SortInfo) ([]boruta.WorkerInfo, error) {
	ret := m.ctrl.Call(m, "ListWorkers", arg0, arg1, arg2)
	ret0, _ := ret[0].([]boruta.WorkerInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListWorkers indicates an expected call of ListWorkers
func (mr *MockWorkersMockRecorder) ListWorkers(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListWorkers", reflect.TypeOf((*MockWorkers)(nil).ListWorkers), arg0, arg1, arg2)
}

// SetGroups mocks base method
func (m *MockWorkers) SetGroups(arg0 boruta.WorkerUUID, arg1 boruta.Groups) error {
	ret := m.ctrl.Call(m, "SetGroups", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetGroups indicates an expected call of SetGroups
func (mr *MockWorkersMockRecorder) SetGroups(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetGroups", reflect.TypeOf((*MockWorkers)(nil).SetGroups), arg0, arg1)
}

// SetState mocks base method
func (m *MockWorkers) SetState(arg0 boruta.WorkerUUID, arg1 boruta.WorkerState) error {
	ret := m.ctrl.Call(m, "SetState", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetState indicates an expected call of SetState
func (mr *MockWorkersMockRecorder) SetState(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetState", reflect.TypeOf((*MockWorkers)(nil).SetState), arg0, arg1)
}

// MockSuperviser is a mock of Superviser interface
type MockSuperviser struct {
	ctrl     *gomock.Controller
	recorder *MockSuperviserMockRecorder
}

// MockSuperviserMockRecorder is the mock recorder for MockSuperviser
type MockSuperviserMockRecorder struct {
	mock *MockSuperviser
}

// NewMockSuperviser creates a new mock instance
func NewMockSuperviser(ctrl *gomock.Controller) *MockSuperviser {
	mock := &MockSuperviser{ctrl: ctrl}
	mock.recorder = &MockSuperviserMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSuperviser) EXPECT() *MockSuperviserMockRecorder {
	return m.recorder
}

// Register mocks base method
func (m *MockSuperviser) Register(arg0 boruta.Capabilities, arg1, arg2 string) error {
	ret := m.ctrl.Call(m, "Register", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Register indicates an expected call of Register
func (mr *MockSuperviserMockRecorder) Register(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Register", reflect.TypeOf((*MockSuperviser)(nil).Register), arg0, arg1, arg2)
}

// SetFail mocks base method
func (m *MockSuperviser) SetFail(arg0 boruta.WorkerUUID, arg1 string) error {
	ret := m.ctrl.Call(m, "SetFail", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetFail indicates an expected call of SetFail
func (mr *MockSuperviserMockRecorder) SetFail(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetFail", reflect.TypeOf((*MockSuperviser)(nil).SetFail), arg0, arg1)
}
