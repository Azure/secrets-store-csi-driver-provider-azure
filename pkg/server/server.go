package server

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
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
		klog.ErrorS(err, "failed to unmarshal attributes")
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to unmarshal attributes, error: %w", err)
	}
	err = json.Unmarshal([]byte(req.GetSecrets()), &secret)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal node publish secrets ref")
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to unmarshal secrets, error: %w", err)
	}
	err = json.Unmarshal([]byte(req.GetPermission()), &filePermission)
	if err != nil {
		klog.ErrorS(err, "failed to unmarshal file permission")
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to unmarshal file permission, error: %w", err)
	}
	provider, err := provider.NewProvider()
	if err != nil {
		klog.ErrorS(err, "failed to initialize new provider")
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to initialize new provider, error: %w", err)
	}

	files, objectVersions, err := provider.MountSecretsStoreObjectContent(ctx, attrib, secret, req.GetTargetPath(), filePermission)
	if err != nil {
		klog.ErrorS(err, "failed to process mount request")
		return &v1alpha1.MountResponse{}, fmt.Errorf("failed to mount objects, error: %w", err)
	}
	ov := []*v1alpha1.ObjectVersion{}
	for k, v := range objectVersions {
		ov = append(ov, &v1alpha1.ObjectVersion{Id: k, Version: v})
	}

	f := []*v1alpha1.File{}
	// CSI driver v0.0.21+ will write to the filesystem if the files are in the response.
	// No files in the response translates to "not implemented" in the CSI driver.
	for k, v := range files {
		f = append(f, &v1alpha1.File{
			Path:     k,
			Contents: v,
			Mode:     int32(filePermission),
		})
	}

	return &v1alpha1.MountResponse{
		ObjectVersion: ov,
		Files:         f,
	}, nil
}

func (s *CSIDriverProviderServer) Version(ctx context.Context, req *v1alpha1.VersionRequest) (*v1alpha1.VersionResponse, error) {
	return &v1alpha1.VersionResponse{
		Version:        "v1alpha1",
		RuntimeVersion: version.BuildVersion,
		RuntimeName:    "secrets-store-csi-driver-provider-azure",
	}, nil
}

func (s *CSIDriverProviderServer) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch for the serving status of the requested service.
func (s *CSIDriverProviderServer) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watch is not supported")
}
