// Code generated by MockGen. DO NOT EDIT.
// Source: service.go

// Package catalog is a generated GoMock package.
package catalog

import (
	context "context"
	estafette_ci_contracts "github.com/estafette/estafette-ci-contracts"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockService is a mock of Service interface
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CreateCatalogEntity mocks base method
func (m *MockService) CreateCatalogEntity(ctx context.Context, catalogEntity estafette_ci_contracts.CatalogEntity) (*estafette_ci_contracts.CatalogEntity, error) {
	ret := m.ctrl.Call(m, "CreateCatalogEntity", ctx, catalogEntity)
	ret0, _ := ret[0].(*estafette_ci_contracts.CatalogEntity)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateCatalogEntity indicates an expected call of CreateCatalogEntity
func (mr *MockServiceMockRecorder) CreateCatalogEntity(ctx, catalogEntity interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateCatalogEntity", reflect.TypeOf((*MockService)(nil).CreateCatalogEntity), ctx, catalogEntity)
}

// UpdateCatalogEntity mocks base method
func (m *MockService) UpdateCatalogEntity(ctx context.Context, catalogEntity estafette_ci_contracts.CatalogEntity) error {
	ret := m.ctrl.Call(m, "UpdateCatalogEntity", ctx, catalogEntity)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateCatalogEntity indicates an expected call of UpdateCatalogEntity
func (mr *MockServiceMockRecorder) UpdateCatalogEntity(ctx, catalogEntity interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCatalogEntity", reflect.TypeOf((*MockService)(nil).UpdateCatalogEntity), ctx, catalogEntity)
}

// DeleteCatalogEntity mocks base method
func (m *MockService) DeleteCatalogEntity(ctx context.Context, id string) error {
	ret := m.ctrl.Call(m, "DeleteCatalogEntity", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCatalogEntity indicates an expected call of DeleteCatalogEntity
func (mr *MockServiceMockRecorder) DeleteCatalogEntity(ctx, id interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCatalogEntity", reflect.TypeOf((*MockService)(nil).DeleteCatalogEntity), ctx, id)
}