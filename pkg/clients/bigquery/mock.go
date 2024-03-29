// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

// Package bigquery is a generated GoMock package.
package bigquery

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// CheckIfDatasetExists mocks base method.
func (m *MockClient) CheckIfDatasetExists(ctx context.Context) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckIfDatasetExists", ctx)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CheckIfDatasetExists indicates an expected call of CheckIfDatasetExists.
func (mr *MockClientMockRecorder) CheckIfDatasetExists(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckIfDatasetExists", reflect.TypeOf((*MockClient)(nil).CheckIfDatasetExists), ctx)
}

// CheckIfTableExists mocks base method.
func (m *MockClient) CheckIfTableExists(ctx context.Context, table string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckIfTableExists", ctx, table)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CheckIfTableExists indicates an expected call of CheckIfTableExists.
func (mr *MockClientMockRecorder) CheckIfTableExists(ctx, table interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckIfTableExists", reflect.TypeOf((*MockClient)(nil).CheckIfTableExists), ctx, table)
}

// CreateTable mocks base method.
func (m *MockClient) CreateTable(ctx context.Context, table string, typeForSchema interface{}, partitionField string, waitReady bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTable", ctx, table, typeForSchema, partitionField, waitReady)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTable indicates an expected call of CreateTable.
func (mr *MockClientMockRecorder) CreateTable(ctx, table, typeForSchema, partitionField, waitReady interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTable", reflect.TypeOf((*MockClient)(nil).CreateTable), ctx, table, typeForSchema, partitionField, waitReady)
}

// Init mocks base method.
func (m *MockClient) Init(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Init indicates an expected call of Init.
func (mr *MockClientMockRecorder) Init(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockClient)(nil).Init), ctx)
}

// InsertBuildEvent mocks base method.
func (m *MockClient) InsertBuildEvent(ctx context.Context, event PipelineBuildEvent) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertBuildEvent", ctx, event)
	ret0, _ := ret[0].(error)
	return ret0
}

// InsertBuildEvent indicates an expected call of InsertBuildEvent.
func (mr *MockClientMockRecorder) InsertBuildEvent(ctx, event interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertBuildEvent", reflect.TypeOf((*MockClient)(nil).InsertBuildEvent), ctx, event)
}

// InsertReleaseEvent mocks base method.
func (m *MockClient) InsertReleaseEvent(ctx context.Context, event PipelineReleaseEvent) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertReleaseEvent", ctx, event)
	ret0, _ := ret[0].(error)
	return ret0
}

// InsertReleaseEvent indicates an expected call of InsertReleaseEvent.
func (mr *MockClientMockRecorder) InsertReleaseEvent(ctx, event interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertReleaseEvent", reflect.TypeOf((*MockClient)(nil).InsertReleaseEvent), ctx, event)
}

// UpdateTableSchema mocks base method.
func (m *MockClient) UpdateTableSchema(ctx context.Context, table string, typeForSchema interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTableSchema", ctx, table, typeForSchema)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateTableSchema indicates an expected call of UpdateTableSchema.
func (mr *MockClientMockRecorder) UpdateTableSchema(ctx, table, typeForSchema interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTableSchema", reflect.TypeOf((*MockClient)(nil).UpdateTableSchema), ctx, table, typeForSchema)
}
