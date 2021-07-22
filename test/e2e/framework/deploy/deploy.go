// +build e2e

package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"

	. "github.com/onsi/gomega"
)

var (
	driverResourcePath        = "https://raw.githubusercontent.com/kubernetes-sigs/secrets-store-csi-driver/master/deploy"
	providerResourceDirectory = "deployment"

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

func InstallManifest(kubeconfigPath string) {
	for _, resource := range driverResources {
		exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", driverResourcePath, resource)})
	}

	wd, err := os.Getwd()
	Expect(err).To(BeNil())

	providerResourceAbsolutePath, err := filepath.Abs(fmt.Sprintf("%s/../../%s", wd, providerResourceDirectory))
	Expect(err).To(BeNil())

	for _, resource := range providerResources {
		exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", providerResourceAbsolutePath, resource)})
	}
}
