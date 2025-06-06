//go:build e2e
// +build e2e

package framework

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config holds global test configuration translated from environment variables
type Config struct {
	SubscriptionID                    string `envconfig:"SUBSCRIPTION_ID"`
	AzureClientID                     string `envconfig:"AZURE_CLIENT_ID"`
	TenantID                          string `envconfig:"TENANT_ID"`
	KeyvaultName                      string `envconfig:"KEYVAULT_NAME"`
	Registry                          string `envconfig:"REGISTRY" default:"mcr.microsoft.com/oss/v2/azure/secrets-store"`
	ImageName                         string `envconfig:"IMAGE_NAME" default:"provider-azure"`
	ImageVersion                      string `envconfig:"IMAGE_VERSION" default:"v1.7.0"`
	IsSoakTest                        bool   `envconfig:"IS_SOAK_TEST" default:"false"`
	IsWindowsTest                     bool   `envconfig:"TEST_WINDOWS" default:"false"`
	IsGPUTest                         bool   `envconfig:"TEST_GPU" default:"false"`
	IsKindCluster                     bool   `envconfig:"CI_KIND_CLUSTER" default:"false"`
	SecretValue                       string `envconfig:"SECRET_VALUE" default:"test"`
	KeyValue                          string `envconfig:"KEY_VALUE" default:"uiPCav0xdIq"`
	UserAssignedIdentityID            string `envconfig:"USER_ASSIGN_IDENTITY_ID"`
	PodIdentityUserMSIName            string `envconfig:"POD_IDENTITY_USER_MSI_NAME"`
	PodIdentityUserAssignedIdentityID string `envconfig:"POD_IDENTITY_USER_ASSIGN_IDENTITY_ID"`
	ResourceGroup                     string `envconfig:"RESOURCE_GROUP"`
	IsUpgradeTest                     bool   `envconfig:"IS_UPGRADE_TEST"`
	IsHelmTest                        bool   `envconfig:"IS_HELM_TEST" default:"true"`
	HelmChartDir                      string `envconfig:"HELM_CHART_DIR" default:"manifest_staging/charts/csi-secrets-store-provider-azure"`
	IsClusterUpgraded                 bool   `envconfig:"IS_CLUSTER_UPGRADED"`
	IsBackwardCompatibilityTest       bool   `envconfig:"IS_BACKWARD_COMPATIBILITY_TEST"`
	AzureEnvironmentFilePath          string `envconfig:"AZURE_ENVIRONMENT_FILEPATH"`
	IsArcTest                         bool   `envconfig:"IS_ARC_TEST" default:"false"`

	// KeyvaultClientID is the client ID of the service principal used to access the keyvault
	KeyvaultClientID string `envconfig:"KEYVAULT_CLIENT_ID" default:"878afdc6-3fc3-4c3e-be5c-f28377892326"`
}

func (c *Config) DeepCopy() *Config {
	copy := new(Config)
	copy.SubscriptionID = c.SubscriptionID
	copy.AzureClientID = c.AzureClientID
	copy.TenantID = c.TenantID
	copy.KeyvaultName = c.KeyvaultName
	copy.Registry = c.Registry
	copy.ImageName = c.ImageName
	copy.ImageVersion = c.ImageVersion
	copy.IsSoakTest = c.IsSoakTest
	copy.IsWindowsTest = c.IsWindowsTest
	copy.IsGPUTest = c.IsGPUTest
	copy.IsKindCluster = c.IsKindCluster
	copy.SecretValue = c.SecretValue
	copy.KeyValue = c.KeyValue
	copy.UserAssignedIdentityID = c.UserAssignedIdentityID
	copy.PodIdentityUserMSIName = c.PodIdentityUserMSIName
	copy.PodIdentityUserAssignedIdentityID = c.PodIdentityUserAssignedIdentityID
	copy.ResourceGroup = c.ResourceGroup
	copy.IsUpgradeTest = c.IsUpgradeTest
	copy.HelmChartDir = c.HelmChartDir
	copy.IsClusterUpgraded = c.IsClusterUpgraded
	copy.IsBackwardCompatibilityTest = c.IsBackwardCompatibilityTest
	copy.AzureEnvironmentFilePath = c.AzureEnvironmentFilePath
	copy.IsHelmTest = c.IsHelmTest
	copy.IsArcTest = c.IsArcTest
	copy.KeyvaultClientID = c.KeyvaultClientID

	return copy
}

// ParseConfig parses the needed environment variables for running the tests
func ParseConfig() (*Config, error) {
	c := new(Config)
	if err := envconfig.Process("config", c); err != nil {
		return c, err
	}
	return c, nil
}

func (c *Config) GetOsSpecificVersionedFilePath(baseFileName string, versionIndex int32) string {
	if c.IsWindowsTest {
		return fmt.Sprintf("%s\\%d", baseFileName, versionIndex)
	}

	return fmt.Sprintf("%s/%d", baseFileName, versionIndex)
}
