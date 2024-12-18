// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/armadaproject/armada/internal/scheduler/database (interfaces: JobRepository)

// Package schedulermocks is a generated GoMock package.
package schedulermocks

import (
	reflect "reflect"

	armadacontext "github.com/armadaproject/armada/internal/common/armadacontext"
	database "github.com/armadaproject/armada/internal/scheduler/database"
	armadaevents "github.com/armadaproject/armada/pkg/armadaevents"
	gomock "github.com/golang/mock/gomock"
	uuid "github.com/google/uuid"
)

// MockJobRepository is a mock of JobRepository interface.
type MockJobRepository struct {
	ctrl     *gomock.Controller
	recorder *MockJobRepositoryMockRecorder
}

// MockJobRepositoryMockRecorder is the mock recorder for MockJobRepository.
type MockJobRepositoryMockRecorder struct {
	mock *MockJobRepository
}

// NewMockJobRepository creates a new mock instance.
func NewMockJobRepository(ctrl *gomock.Controller) *MockJobRepository {
	mock := &MockJobRepository{ctrl: ctrl}
	mock.recorder = &MockJobRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockJobRepository) EXPECT() *MockJobRepositoryMockRecorder {
	return m.recorder
}

// CountReceivedPartitions mocks base method.
func (m *MockJobRepository) CountReceivedPartitions(arg0 *armadacontext.Context, arg1 uuid.UUID) (uint32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountReceivedPartitions", arg0, arg1)
	ret0, _ := ret[0].(uint32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CountReceivedPartitions indicates an expected call of CountReceivedPartitions.
func (mr *MockJobRepositoryMockRecorder) CountReceivedPartitions(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountReceivedPartitions", reflect.TypeOf((*MockJobRepository)(nil).CountReceivedPartitions), arg0, arg1)
}

// FetchInitialJobs mocks base method.
func (m *MockJobRepository) FetchInitialJobs(arg0 *armadacontext.Context) ([]database.Job, []database.Run, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchInitialJobs", arg0)
	ret0, _ := ret[0].([]database.Job)
	ret1, _ := ret[1].([]database.Run)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// FetchInitialJobs indicates an expected call of FetchInitialJobs.
func (mr *MockJobRepositoryMockRecorder) FetchInitialJobs(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchInitialJobs", reflect.TypeOf((*MockJobRepository)(nil).FetchInitialJobs), arg0)
}

// FetchJobRunErrors mocks base method.
func (m *MockJobRepository) FetchJobRunErrors(arg0 *armadacontext.Context, arg1 []string) (map[string]*armadaevents.Error, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchJobRunErrors", arg0, arg1)
	ret0, _ := ret[0].(map[string]*armadaevents.Error)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchJobRunErrors indicates an expected call of FetchJobRunErrors.
func (mr *MockJobRepositoryMockRecorder) FetchJobRunErrors(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchJobRunErrors", reflect.TypeOf((*MockJobRepository)(nil).FetchJobRunErrors), arg0, arg1)
}

// FetchJobRunLeases mocks base method.
func (m *MockJobRepository) FetchJobRunLeases(arg0 *armadacontext.Context, arg1 string, arg2 uint, arg3 []string) ([]*database.JobRunLease, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchJobRunLeases", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]*database.JobRunLease)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchJobRunLeases indicates an expected call of FetchJobRunLeases.
func (mr *MockJobRepositoryMockRecorder) FetchJobRunLeases(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchJobRunLeases", reflect.TypeOf((*MockJobRepository)(nil).FetchJobRunLeases), arg0, arg1, arg2, arg3)
}

// FetchJobUpdates mocks base method.
func (m *MockJobRepository) FetchJobUpdates(arg0 *armadacontext.Context, arg1, arg2 int64) ([]database.Job, []database.Run, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FetchJobUpdates", arg0, arg1, arg2)
	ret0, _ := ret[0].([]database.Job)
	ret1, _ := ret[1].([]database.Run)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// FetchJobUpdates indicates an expected call of FetchJobUpdates.
func (mr *MockJobRepositoryMockRecorder) FetchJobUpdates(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchJobUpdates", reflect.TypeOf((*MockJobRepository)(nil).FetchJobUpdates), arg0, arg1, arg2)
}

// FindInactiveRuns mocks base method.
func (m *MockJobRepository) FindInactiveRuns(arg0 *armadacontext.Context, arg1 []string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindInactiveRuns", arg0, arg1)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindInactiveRuns indicates an expected call of FindInactiveRuns.
func (mr *MockJobRepositoryMockRecorder) FindInactiveRuns(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindInactiveRuns", reflect.TypeOf((*MockJobRepository)(nil).FindInactiveRuns), arg0, arg1)
}
