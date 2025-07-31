//go:build e2e
// +build e2e

package deploy

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/auth"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"

	"github.com/ghodss/yaml"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
)

var (
	driverResourcePath        = "https://raw.githubusercontent.com/kubernetes-sigs/secrets-store-csi-driver/v1.5.3/deploy"
	providerResourceDirectory = "manifest_staging/deployment"

	driverResources = []string{
		// "csidriver.yaml" will be downloaded, modified and installed in deployDriver()
		"rbac-secretproviderclass.yaml",
		"rbac-secretproviderrotation.yaml",
		"rbac-secretprovidersyncing.yaml",
		"rbac-secretprovidertokenrequest.yaml",
		"secrets-store-csi-driver-windows.yaml",
		"secrets-store-csi-driver.yaml",
		"secrets-store.csi.x-k8s.io_secretproviderclasses.yaml",
		"secrets-store.csi.x-k8s.io_secretproviderclasspodstatuses.yaml",
	}

	providerResources = []string{
		"provider-azure-installer.yaml",
	}
)

// InstallManifest install driver and provider manifests from yaml files.
func InstallManifest(kubeconfigPath string, config *framework.Config) {
	deployDriver(kubeconfigPath, config)

	wd, err := os.Getwd()
	Expect(err).To(BeNil())

	providerResourceAbsolutePath, err := filepath.Abs(fmt.Sprintf("%s/../../%s", wd, providerResourceDirectory))
	Expect(err).To(BeNil())

	for _, resource := range providerResources {
		file, err := os.Open(fmt.Sprintf("%s/%s", providerResourceAbsolutePath, resource))
		Expect(err).To(BeNil())

		fileBytes, err := io.ReadAll(file)
		Expect(err).To(BeNil())

		// resource yaml file contains both SA and DS configuration. In order to update DS, extract DS yaml
		fileContent := string(fileBytes)
		subString := "---"
		pos := strings.LastIndex(fileContent, subString)
		if pos == -1 {
			return
		}
		adjustedPos := pos + len(subString)
		if adjustedPos >= len(fileContent) {
			return
		}
		dsYAML := fileContent[adjustedPos:]

		ds := &appsv1.DaemonSet{}
		err = yaml.Unmarshal([]byte(dsYAML), ds)
		Expect(err).To(BeNil())

		// If it's windows, then skip DS update as we are building linux image only for kind test
		if ds.Spec.Template.Spec.NodeSelector["kubernetes.io/os"] == "windows" {
			continue
		}

		// Update the image
		ds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("%s/%s:%s", config.Registry, config.ImageName, config.ImageVersion)

		// Add Volume and Volume mount required for testing
		ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "cloudenvfile-vol",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/kubernetes/custom_environment.json",
				},
			},
		})

		ds.Spec.Template.Spec.Containers[0].VolumeMounts = append(ds.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "cloudenvfile-vol",
			MountPath: "/etc/kubernetes/custom_environment.json",
		})

		// Configure higher log verbosity for debugging CI failures
		ds.Spec.Template.Spec.Containers[0].Args = append(ds.Spec.Template.Spec.Containers[0].Args, "-v=5")
		// Configure writeCertAndKeyInSeparateFiles to true as it's feature on top of default behavior
		ds.Spec.Template.Spec.Containers[0].Args = append(ds.Spec.Template.Spec.Containers[0].Args, "--write-cert-and-key-in-separate-files=true")

		updatedDS, err := yaml.Marshal(ds)
		Expect(err).To(BeNil())

		err = os.WriteFile(fmt.Sprintf("%s/updated-%s", providerResourceAbsolutePath, resource), updatedDS, 0644)
		Expect(err).To(BeNil())

		// Run original yaml to install SA
		err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", providerResourceAbsolutePath, resource)})
		Expect(err).To(BeNil())

		// Update DS with new configuration
		err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/updated-%s", providerResourceAbsolutePath, resource)})
		Expect(err).To(BeNil())
	}
}

func deployDriver(kubeconfigPath string, config *framework.Config) {
	resp, err := http.Get(fmt.Sprintf("%s/%s", driverResourcePath, "csidriver.yaml"))
	Expect(err).To(BeNil())

	csiDriverYAML, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	Expect(err).To(BeNil())

	csiDriver := &storagev1.CSIDriver{}
	err = yaml.Unmarshal(csiDriverYAML, csiDriver)
	Expect(err).To(BeNil())

	// Modify the CSI driver spec to include token requests
	// With this we can enable workload identity tests with manifests in addition to helm
	csiDriver.Spec.TokenRequests = []storagev1.TokenRequest{
		{
			Audience: auth.DefaultTokenAudience,
		},
	}

	updatedCSIDriver, err := yaml.Marshal(csiDriver)
	Expect(err).To(BeNil())

	updateCSIDriverYAMLFile := filepath.Join(os.TempDir(), driverResources[0])
	err = os.WriteFile(updateCSIDriverYAMLFile, updatedCSIDriver, 0644)
	Expect(err).To(BeNil())

	// Install the CSIDriver
	err = exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", updateCSIDriverYAMLFile})
	Expect(err).To(BeNil())

	// Install the remaining driver resources
	for _, resource := range driverResources {
		err := exec.KubectlApply(kubeconfigPath, framework.NamespaceKubeSystem, []string{"-f", fmt.Sprintf("%s/%s", driverResourcePath, resource)})
		Expect(err).To(BeNil())
	}
}
