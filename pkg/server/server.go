package server

import (
	"encoding/json"
	"fmt"
	"os"

	provider2 "github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"

	"k8s.io/klog"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

// CSIDriverProviderServer providers a Secrets Store CSI Driver provider implementation
type CSIDriverProviderServer struct {
	*grpc.Server
}

// Mount executes the mount operation in the provider. The provider fetches the objects from Key Vault
// writes the contents to the pod mount and returns the object versions as part of MountResponse
func (s *CSIDriverProviderServer) Mount(ctx context.Context, req *v1alpha1.MountRequest) (*v1alpha1.MountResponse, error) {
	var attrib, secret map[string]string
	var filePermission os.FileMode
	var err error

	err = json.Unmarshal([]byte(req.GetAttributes()), &attrib)
	if err != nil {
		klog.Errorf("failed to unmarshal attributes, error: %+v", err)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to unmarshal attributes, error: %v", err)
	}
	err = json.Unmarshal([]byte(req.GetSecrets()), &secret)
	if err != nil {
		klog.Errorf("failed to unmarshal secrets, error: %+v", err)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to unmarshal secrets, error: %v", err)
	}
	err = json.Unmarshal([]byte(req.GetPermission()), &filePermission)
	if err != nil {
		klog.Errorf("failed to unmarshal file permission, error: %+v", err)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to unmarshal file permission, error: %v", err)
	}
	provider, err := provider2.NewProvider()
	if err != nil {
		klog.Errorf("failed to initialize new provider, error: %v", err)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to initialize new provider, error: %v", err)
	}

	objectVersions, err := provider.MountSecretsStoreObjectContent(ctx, attrib, secret, req.GetTargetPath(), filePermission)
	if err != nil {
		klog.Errorf("failed to process mount request, error: %+v", err)
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to mount objects, error: %v", err)
	}
	var ov []*v1alpha1.ObjectVersion
	for k, v := range objectVersions {
		ov = append(ov, &v1alpha1.ObjectVersion{Id: k, Version: v})
	}

	return &v1alpha1.MountResponse{ObjectVersion: ov}, nil
}

func (s *CSIDriverProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Version not implemented")
}
