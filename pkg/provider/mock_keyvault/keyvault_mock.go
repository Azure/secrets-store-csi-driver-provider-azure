// Code generated by MockGen. DO NOT EDIT.
// Source: ../keyvault.go

// Package mock_keyvault is a generated GoMock package.
package mock_keyvault

import (
	context "context"
	reflect "reflect"

	azcertificates "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azcertificates"
	azkeys "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azkeys"
	azsecrets "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	types "github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	gomock "github.com/golang/mock/gomock"
)

// MockKeyVault is a mock of KeyVault interface.
type MockKeyVault struct {
	ctrl     *gomock.Controller
	recorder *MockKeyVaultMockRecorder
}

// MockKeyVaultMockRecorder is the mock recorder for MockKeyVault.
type MockKeyVaultMockRecorder struct {
	mock *MockKeyVault
}

// NewMockKeyVault creates a new mock instance.
func NewMockKeyVault(ctrl *gomock.Controller) *MockKeyVault {
	mock := &MockKeyVault{ctrl: ctrl}
	mock.recorder = &MockKeyVaultMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKeyVault) EXPECT() *MockKeyVaultMockRecorder {
	return m.recorder
}

// GetCertificate mocks base method.
func (m *MockKeyVault) GetCertificate(ctx context.Context, name, version string) (*azcertificates.CertificateBundle, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificate", ctx, name, version)
	ret0, _ := ret[0].(*azcertificates.CertificateBundle)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCertificate indicates an expected call of GetCertificate.
func (mr *MockKeyVaultMockRecorder) GetCertificate(ctx, name, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificate", reflect.TypeOf((*MockKeyVault)(nil).GetCertificate), ctx, name, version)
}

// GetCertificateVersions mocks base method.
func (m *MockKeyVault) GetCertificateVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificateVersions", ctx, name)
	ret0, _ := ret[0].([]types.KeyVaultObjectVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCertificateVersions indicates an expected call of GetCertificateVersions.
func (mr *MockKeyVaultMockRecorder) GetCertificateVersions(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificateVersions", reflect.TypeOf((*MockKeyVault)(nil).GetCertificateVersions), ctx, name)
}

// GetKey mocks base method.
func (m *MockKeyVault) GetKey(ctx context.Context, name, version string) (*azkeys.KeyBundle, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKey", ctx, name, version)
	ret0, _ := ret[0].(*azkeys.KeyBundle)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKey indicates an expected call of GetKey.
func (mr *MockKeyVaultMockRecorder) GetKey(ctx, name, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKey", reflect.TypeOf((*MockKeyVault)(nil).GetKey), ctx, name, version)
}

// GetKeyVersions mocks base method.
func (m *MockKeyVault) GetKeyVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKeyVersions", ctx, name)
	ret0, _ := ret[0].([]types.KeyVaultObjectVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKeyVersions indicates an expected call of GetKeyVersions.
func (mr *MockKeyVaultMockRecorder) GetKeyVersions(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKeyVersions", reflect.TypeOf((*MockKeyVault)(nil).GetKeyVersions), ctx, name)
}

// GetSecret mocks base method.
func (m *MockKeyVault) GetSecret(ctx context.Context, name, version string) (*azsecrets.SecretBundle, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecret", ctx, name, version)
	ret0, _ := ret[0].(*azsecrets.SecretBundle)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecret indicates an expected call of GetSecret.
func (mr *MockKeyVaultMockRecorder) GetSecret(ctx, name, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecret", reflect.TypeOf((*MockKeyVault)(nil).GetSecret), ctx, name, version)
}

// GetSecretVersions mocks base method.
func (m *MockKeyVault) GetSecretVersions(ctx context.Context, name string) ([]types.KeyVaultObjectVersion, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecretVersions", ctx, name)
	ret0, _ := ret[0].([]types.KeyVaultObjectVersion)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecretVersions indicates an expected call of GetSecretVersions.
func (mr *MockKeyVaultMockRecorder) GetSecretVersions(ctx, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecretVersions", reflect.TypeOf((*MockKeyVault)(nil).GetSecretVersions), ctx, name)
}
