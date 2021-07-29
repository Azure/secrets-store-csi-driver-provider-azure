// +build e2e

package deploy

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"

	. "github.com/onsi/gomega"
)

var (
	driverResourcePath        = "https://raw.githubusercontent.com/kubernetes-sigs/secrets-store-csi-driver/master/deploy"
	providerResourceDirectory = "manifest_staging/deployment"

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

// InstallManifest install driver and provider manifests from yaml files.
func InstallManifest(kubeconfigPath string, config *framework.Config) {
	for _, resource := range driverResources {
		err := exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", driverResourcePath, resource)})
		Expect(err).To(BeNil())
	}

	wd, err := os.Getwd()
	Expect(err).To(BeNil())

	providerResourceAbsolutePath, err := filepath.Abs(fmt.Sprintf("%s/../../%s", wd, providerResourceDirectory))
	Expect(err).To(BeNil())

	for _, resource := range providerResources {
		file, err := os.Open(fmt.Sprintf("%s/%s", providerResourceAbsolutePath, resource))
		Expect(err).To(BeNil())

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}

		ds := &appsv1.DaemonSet{}
		err = yaml.Unmarshal(fileBytes, ds)
		Expect(err).To(BeNil())

		ds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", config.Registry, config.ImageName, config.ImageVersion)
		updatedDS, err := yaml.Marshal(ds)
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(fmt.Sprintf("%s/updated-%s", providerResourceAbsolutePath, resource), updatedDS, 0444)
		Expect(err).To(BeNil())

		err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", providerResourceAbsolutePath, resource)})
		Expect(err).To(BeNil())

		err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/updated-%s", providerResourceAbsolutePath, resource)})
		Expect(err).To(BeNil())
	}
}
