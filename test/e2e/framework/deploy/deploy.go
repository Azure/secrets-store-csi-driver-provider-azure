// +build e2e

package deploy

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	appsv1 "k8s.io/api/apps/v1"

	"sigs.k8s.io/yaml"

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
		Expect(err).To(BeNil())

		fileContent := fmt.Sprintf("%s", fileBytes)
		pos := strings.LastIndex(fileContent, "---")
		if pos == -1 {
			return
		}
		adjustedPos := pos + len("---")
		if adjustedPos >= len(fileContent) {
			return
		}
		dsYaml := fileContent[adjustedPos:len(fileContent)]

		ds := &appsv1.DaemonSet{}
		err = yaml.Unmarshal([]byte(dsYaml), ds)
		Expect(err).To(BeNil())

		ds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", config.Registry, config.ImageName, config.ImageVersion)	
		updatedDS, err := yaml.Marshal(ds)
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(fmt.Sprintf("%s/updated-%s", providerResourceAbsolutePath, resource), updatedDS, 0644)
		Expect(err).To(BeNil())

		err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", providerResourceAbsolutePath, resource)})
		Expect(err).To(BeNil())

		err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/updated-%s", providerResourceAbsolutePath, resource)})
		Expect(err).To(BeNil())
	}
}
