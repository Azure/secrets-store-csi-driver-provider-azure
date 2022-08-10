package server

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/mock_provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	"github.com/golang/mock/gomock"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

func TestMountError(t *testing.T) {
	cases := []struct {
		desc         string
		mountRequest *v1alpha1.MountRequest
	}{
		{
			desc:         "failed to unmarshal attributes",
			mountRequest: &v1alpha1.MountRequest{},
		},
		{
			desc: "failed to unmarshal secrets",
			mountRequest: &v1alpha1.MountRequest{
				Attributes: `{"keyvaultName":"kv"}`,
			},
		},
		{
			desc: "failed to unmarshal file permission",
			mountRequest: &v1alpha1.MountRequest{
				Attributes: `{"keyvaultName":"kv"}`,
				Secrets:    `{"clientid":"foo","clientsecret":"bar"}`,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			testServer := &CSIDriverProviderServer{
				provider: mock_provider.NewMockInterface(ctrl),
			}
			if _, err := testServer.Mount(context.TODO(), tc.mountRequest); err == nil {
				t.Fatalf("Mount() expected error, got nil")
			}
		})
	}
}

func TestMount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testServer := &CSIDriverProviderServer{}
	mockProvider := mock_provider.NewMockInterface(ctrl)
	mockProvider.EXPECT().GetSecretsStoreObjectContent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		[]types.SecretFile{
			{
				Content: []byte("foo"),
				Path:    "foo.txt",
				Version: "1",
			},
			{
				Content: []byte("bar"),
				Path:    "bar.txt",
				Version: "2",
			},
		}, nil,
	)
	testServer.provider = mockProvider
	response, err := testServer.Mount(context.TODO(), &v1alpha1.MountRequest{
		Attributes: `{"keyvaultName":"kv"}`,
		Secrets:    `{"clientid":"foo","clientsecret":"bar"}`,
		Permission: "420",
	})
	if err != nil {
		t.Fatalf("Mount() expected no error, got %v", err)
	}
	if len(response.Files) != 2 {
		t.Fatalf("Mount() expected 2 files, got %v", len(response.Files))
	}
	if len(response.ObjectVersion) != 2 {
		t.Fatalf("Mount() expected 2 object versions, got %v", len(response.ObjectVersion))
	}
}

func TestVersion(t *testing.T) {
	testServer := &CSIDriverProviderServer{}
	version.BuildVersion = "test"
	resp, err := testServer.Version(context.TODO(), &v1alpha1.VersionRequest{})
	if err != nil {
		t.Fatalf("expected error to be nil")
	}
	expectedVersionResponse := &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeVersion: "test",
		RuntimeName:    "secrets-store-csi-driver-provider-azure",
	}
	if !reflect.DeepEqual(resp, expectedVersionResponse) {
		t.Fatalf("expected resp: %v, got: %v", expectedVersionResponse, resp)
	}
}
