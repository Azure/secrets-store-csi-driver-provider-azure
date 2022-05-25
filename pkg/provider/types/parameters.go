package types

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
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
