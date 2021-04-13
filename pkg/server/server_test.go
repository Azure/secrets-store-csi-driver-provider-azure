package server

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

func TestMount(t *testing.T) {
	cases := []struct {
		desc         string
		mountRequest *v1alpha1.MountRequest
		expectedErr  bool
	}{
		{
			desc:         "failed to unmarshal attributes",
			mountRequest: &v1alpha1.MountRequest{},
			expectedErr:  true,
		},
		{
			desc: "failed to unmarshal secrets",
			mountRequest: &v1alpha1.MountRequest{
				Attributes: `{"keyvaultName":"kv"}`,
			},
			expectedErr: true,
		},
		{
			desc: "failed to unmarshal file permission",
			mountRequest: &v1alpha1.MountRequest{
				Attributes: `{"keyvaultName":"kv"}`,
				Secrets:    `{"clientid":"foo","clientsecret":"bar"}`,
			},
			expectedErr: true,
		},
		{
			desc: "failed to mount request",
			mountRequest: &v1alpha1.MountRequest{
				Attributes: `{"keyvaultName":"kv","tenantId":"72f988bf-86f1-41af-91ab-2d7cd011db47","objects": "array:"}`,
				Secrets:    `{"clientid":"foo","clientsecret":"bar"}`,
				Permission: "420",
			},
			expectedErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			testServer := &CSIDriverProviderServer{}
			_, err := testServer.Mount(context.TODO(), tc.mountRequest)
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
		})
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
