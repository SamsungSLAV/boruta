// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/SamsungSLAV/boruta/tunnels (interfaces: Tunneler)

package matcher

import (
	gomock "github.com/golang/mock/gomock"
	net "net"
	reflect "reflect"
)

// MockTunneler is a mock of Tunneler interface
type MockTunneler struct {
	ctrl     *gomock.Controller
	recorder *MockTunnelerMockRecorder
}

// MockTunnelerMockRecorder is the mock recorder for MockTunneler
type MockTunnelerMockRecorder struct {
	mock *MockTunneler
}

// NewMockTunneler creates a new mock instance
func NewMockTunneler(ctrl *gomock.Controller) *MockTunneler {
	mock := &MockTunneler{ctrl: ctrl}
	mock.recorder = &MockTunnelerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockTunneler) EXPECT() *MockTunnelerMockRecorder {
	return m.recorder
}

// Addr mocks base method
func (m *MockTunneler) Addr() net.Addr {
	ret := m.ctrl.Call(m, "Addr")
	ret0, _ := ret[0].(net.Addr)
	return ret0
}

// Addr indicates an expected call of Addr
func (mr *MockTunnelerMockRecorder) Addr() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Addr", reflect.TypeOf((*MockTunneler)(nil).Addr))
}

// Close mocks base method
func (m *MockTunneler) Close() error {
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockTunnelerMockRecorder) Close() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockTunneler)(nil).Close))
}

// Create mocks base method
func (m *MockTunneler) Create(arg0 net.IP, arg1 net.TCPAddr) error {
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create
func (mr *MockTunnelerMockRecorder) Create(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockTunneler)(nil).Create), arg0, arg1)
}