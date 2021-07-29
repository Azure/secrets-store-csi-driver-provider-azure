// +build e2e

package deploy

import (
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"

	. "github.com/onsi/gomega"
)

var (
	driverResourcePath   = "https://raw.githubusercontent.com/kubernetes-sigs/secrets-store-csi-driver/master/deploy"
	providerResourcePath = "https://raw.githubusercontent.com/Azure/secrets-store-csi-driver-provider-azure/master/deployment"

	driverResources = []string{
		"csidriver.yaml",
		"rbac-secretproviderclass.yaml",
		"rbac-secretproviderrotation.yaml",
		"rbac-secretprovidersyncing.yaml",
		"secrets-store-csi-driver-windows.yaml",
		"secrets-store-csi-driver.yaml",
		"secrets-store.csi.x-k8s.io_secretproviderclasses.yaml",
		"secrets-store.csi.x-k8s.io_secretproviderclasspodstatuses.yaml",
	}

	providerResources = []string{
		"provider-azure-installer.yaml",
		"provider-azure-installer-windows.yaml",
	}
)

// InstallManifest install driver and provider manifests from yaml files
func InstallManifest(kubeconfigPath string) {
	for _, resource := range driverResources {
		err := exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", driverResourcePath, resource)})
		Expect(err).To(BeNil())
	}

	for _, resource := range providerResources {
		err := exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", providerResourcePath, resource)})
		Expect(err).To(BeNil())
	}
}
