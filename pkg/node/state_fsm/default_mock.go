// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/node/state_fsm/default.go

// Package state_fsm is a generated GoMock package.
package state_fsm

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	. "github.com/wavesplatform/gowaves/pkg/p2p/peer"
)

// MockDefault is a mock of Default interface.
type MockDefault struct {
	ctrl     *gomock.Controller
	recorder *MockDefaultMockRecorder
}

// MockDefaultMockRecorder is the mock recorder for MockDefault.
type MockDefaultMockRecorder struct {
	mock *MockDefault
}

// NewMockDefault creates a new mock instance.
func NewMockDefault(ctrl *gomock.Controller) *MockDefault {
	mock := &MockDefault{ctrl: ctrl}
	mock.recorder = &MockDefaultMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDefault) EXPECT() *MockDefaultMockRecorder {
	return m.recorder
}

// NewPeer mocks base method.
func (m *MockDefault) NewPeer(fsm FSM, p Peer, info BaseInfo) (FSM, Async, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewPeer", fsm, p, info)
	ret0, _ := ret[0].(FSM)
	ret1, _ := ret[1].(Async)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// NewPeer indicates an expected call of NewPeer.
func (mr *MockDefaultMockRecorder) NewPeer(fsm, p, info interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewPeer", reflect.TypeOf((*MockDefault)(nil).NewPeer), fsm, p, info)
}

// Noop mocks base method.
func (m *MockDefault) Noop(arg0 FSM) (FSM, Async, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Noop", arg0)
	ret0, _ := ret[0].(FSM)
	ret1, _ := ret[1].(Async)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Noop indicates an expected call of Noop.
func (mr *MockDefaultMockRecorder) Noop(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Noop", reflect.TypeOf((*MockDefault)(nil).Noop), arg0)
}

// PeerError mocks base method.
func (m *MockDefault) PeerError(fsm FSM, p Peer, baseInfo BaseInfo, arg3 error) (FSM, Async, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PeerError", fsm, p, baseInfo, arg3)
	ret0, _ := ret[0].(FSM)
	ret1, _ := ret[1].(Async)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// PeerError indicates an expected call of PeerError.
func (mr *MockDefaultMockRecorder) PeerError(fsm, p, baseInfo, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PeerError", reflect.TypeOf((*MockDefault)(nil).PeerError), fsm, p, baseInfo, arg3)
}
