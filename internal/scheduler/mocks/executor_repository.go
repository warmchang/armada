// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/armadaproject/armada/internal/scheduler/database (interfaces: ExecutorRepository)

// Package schedulermocks is a generated GoMock package.
package schedulermocks

import (
	reflect "reflect"
	time "time"

	armadacontext "github.com/armadaproject/armada/internal/common/armadacontext"
	schedulerobjects "github.com/armadaproject/armada/internal/scheduler/schedulerobjects"
	gomock "github.com/golang/mock/gomock"
)

// MockExecutorRepository is a mock of ExecutorRepository interface.
type MockExecutorRepository struct {
	ctrl     *gomock.Controller
	recorder *MockExecutorRepositoryMockRecorder
}

// MockExecutorRepositoryMockRecorder is the mock recorder for MockExecutorRepository.
type MockExecutorRepositoryMockRecorder struct {
	mock *MockExecutorRepository
}

// NewMockExecutorRepository creates a new mock instance.
func NewMockExecutorRepository(ctrl *gomock.Controller) *MockExecutorRepository {
	mock := &MockExecutorRepository{ctrl: ctrl}
	mock.recorder = &MockExecutorRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutorRepository) EXPECT() *MockExecutorRepositoryMockRecorder {
	return m.recorder
}

// GetExecutorSettings mocks base method.
func (m *MockExecutorRepository) GetExecutorSettings(arg0 *armadacontext.Context) ([]*schedulerobjects.ExecutorSettings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecutorSettings", arg0)
	ret0, _ := ret[0].([]*schedulerobjects.ExecutorSettings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecutorSettings indicates an expected call of GetExecutorSettings.
func (mr *MockExecutorRepositoryMockRecorder) GetExecutorSettings(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecutorSettings", reflect.TypeOf((*MockExecutorRepository)(nil).GetExecutorSettings), arg0)
}

// GetExecutors mocks base method.
func (m *MockExecutorRepository) GetExecutors(arg0 *armadacontext.Context) ([]*schedulerobjects.Executor, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExecutors", arg0)
	ret0, _ := ret[0].([]*schedulerobjects.Executor)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExecutors indicates an expected call of GetExecutors.
func (mr *MockExecutorRepositoryMockRecorder) GetExecutors(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExecutors", reflect.TypeOf((*MockExecutorRepository)(nil).GetExecutors), arg0)
}

// GetLastUpdateTimes mocks base method.
func (m *MockExecutorRepository) GetLastUpdateTimes(arg0 *armadacontext.Context) (map[string]time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastUpdateTimes", arg0)
	ret0, _ := ret[0].(map[string]time.Time)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLastUpdateTimes indicates an expected call of GetLastUpdateTimes.
func (mr *MockExecutorRepositoryMockRecorder) GetLastUpdateTimes(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastUpdateTimes", reflect.TypeOf((*MockExecutorRepository)(nil).GetLastUpdateTimes), arg0)
}

// StoreExecutor mocks base method.
func (m *MockExecutorRepository) StoreExecutor(arg0 *armadacontext.Context, arg1 *schedulerobjects.Executor) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreExecutor", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreExecutor indicates an expected call of StoreExecutor.
func (mr *MockExecutorRepositoryMockRecorder) StoreExecutor(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreExecutor", reflect.TypeOf((*MockExecutorRepository)(nil).StoreExecutor), arg0, arg1)
}