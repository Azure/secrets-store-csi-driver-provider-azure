package types

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// GetKeyVaultName returns the key vault name
func GetKeyVaultName(parameters map[string]string) string {
	return strings.TrimSpace(parameters[KeyVaultNameParameter])
}

// GetCloudName returns the cloud name
func GetCloudName(parameters map[string]string) string {
	return strings.TrimSpace(parameters[CloudNameParameter])
}

// GetUsePodIdentity returns if pod identity is enabled
func GetUsePodIdentity(parameters map[string]string) (bool, error) {
	str := strings.TrimSpace(parameters[UsePodIdentityParameter])
	if str == "" {
		return false, nil
	}
	return strconv.ParseBool(str)
}

// GetUseVMManagedIdentity returns if VM managed identity is enabled
func GetUseVMManagedIdentity(parameters map[string]string) (bool, error) {
	str := strings.TrimSpace(parameters[UseVMManagedIdentityParameter])
	if str == "" {
		return false, nil
	}
	return strconv.ParseBool(str)
}

// GetUserAssignedIdentityID returns the user assigned identity ID
func GetUserAssignedIdentityID(parameters map[string]string) string {
	return strings.TrimSpace(parameters[UserAssignedIdentityIDParameter])
}

// GetTenantID returns the tenant ID
func GetTenantID(parameters map[string]string) string {
	// ref: https://github.com/Azure/secrets-store-csi-driver-provider-azure/issues/857
	tenantID := strings.TrimSpace(parameters["tenantID"])
	if tenantID != "" {
		return tenantID
	}
	klog.V(3).Info("tenantId is deprecated and will be removed in a future release. Use 'tenantID' instead")
	return strings.TrimSpace(parameters[TenantIDParameter])
}

// GetCloudEnvFileName returns the cloud env file name
func GetCloudEnvFileName(parameters map[string]string) string {
	return strings.TrimSpace(parameters[CloudEnvFileNameParameter])
}

// GetPodName returns the pod name
func GetPodName(parameters map[string]string) string {
	return strings.TrimSpace(parameters[CSIAttributePodName])
}

// GetPodNamespace returns the pod namespace
func GetPodNamespace(parameters map[string]string) string {
	return strings.TrimSpace(parameters[CSIAttributePodNamespace])
}

// GetClientID returns the client ID
func GetClientID(parameters map[string]string) string {
	return strings.TrimSpace(parameters[ClientIDParameter])
}

// GetServiceAccountTokens returns the service account tokens
func GetServiceAccountTokens(parameters map[string]string) string {
	return strings.TrimSpace(parameters[CSIAttributeServiceAccountTokens])
}

// GetObjects returns the key vault objects
func GetObjects(parameters map[string]string) string {
	return strings.TrimSpace(parameters[ObjectsParameter])
}

// GetObjectsArray returns the key vault objects array
func GetObjectsArray(objects string) (StringArray, error) {
	var a StringArray
	err := yaml.Unmarshal([]byte(objects), &a)
	return a, err
}

// IsSyncingSingleVersion returns true if the object is configured
// to only sync a single specific version of the secret
func (kv KeyVaultObject) IsSyncingSingleVersion() bool {
	return kv.ObjectVersionHistory <= 1
}

// GetFileName returns the file name for the secret
// 1. If the object alias is specified, it will be used
// 2. If the object alias is not specified, the object name will be used
func (kv KeyVaultObject) GetFileName() string {
	if kv.ObjectAlias != "" {
		return kv.ObjectAlias
	}
	return kv.ObjectName
}

// GetFilePermission returns the file permission and error if any
func (kv KeyVaultObject) GetFilePermission(defaultFilePermission os.FileMode) (int32, error) {
	if kv.FilePermission == "" {
		return int32(defaultFilePermission), nil
	}
	permission, err := strconv.ParseInt(kv.FilePermission, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("file permission must be a valid octal number: %w", err)
	}
	return int32(permission), nil
}
