// +build e2e

package helm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	chartName            = "csi"
	podIdentityChartName = "pi"
)

// InstallInput is the input for Install.
type InstallInput struct {
	Config         *framework.Config
	NamespacedMode bool
}

// Install installs csi-secrets-store-provider-azure via Helm 3.
func Install(input InstallInput) {
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for Helm.Install")

	cwd, err := os.Getwd()
	Expect(err).To(BeNil())

	// Change current working directory to repo root
	// Before installing csi-secrets-store-provider-azure through Helm
	os.Chdir("../..")
	defer os.Chdir(cwd)

	chartDir := input.Config.HelmChartDir

	args := append([]string{
		"install",
		chartName,
		chartDir,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		fmt.Sprintf("--set=secrets-store-csi-driver.enableSecretRotation=true"),
		fmt.Sprintf("--set=secrets-store-csi-driver.rotationPollInterval=30s"),
		fmt.Sprintf("--set=logVerbosity=1"),
		fmt.Sprintf("--set=linux.customUserAgent=csi-e2e"),
		fmt.Sprintf("--set=windows.customUserAgent=csi-e2e"),
		"--dependency-update",
		"--wait",
		"--timeout=5m",
		"--debug",
	})

	if input.Config.IsWindowsTest {
		args = append(args,
			fmt.Sprintf("--set=windows.enabled=true"),
			fmt.Sprintf("--set=secrets-store-csi-driver.windows.enabled=true"))
	}

	args = append(args, generateValueArgs(input.Config)...)

	err = helm(args)
	Expect(err).To(BeNil())
}

// UpgradeInput is the input for helm upgrade.
type UpgradeInput struct {
	Config *framework.Config
}

//Upgrade upgrades csi-secrets-store-provider-azure to chart specified in config
func Upgrade(input UpgradeInput) {
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for Helm upgrade")

	cwd, err := os.Getwd()
	Expect(err).To(BeNil())

	// Change current working directory to repo root
	// Before installing csi-secrets-store-provider-azure through Helm
	os.Chdir("../..")
	defer os.Chdir(cwd)

	chartDir := input.Config.HelmChartDir

	//resolve helm dependency
	dependencyArgs := append([]string{
		"dependency",
		"update",
		chartDir,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"--debug",
	})
	err = helm(dependencyArgs)
	Expect(err).To(BeNil())

	args := append([]string{
		"upgrade",
		chartName,
		chartDir,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		fmt.Sprintf("--set=secrets-store-csi-driver.enableSecretRotation=true"),
		fmt.Sprintf("--set=secrets-store-csi-driver.rotationPollInterval=30s"),
		fmt.Sprintf("--set=logVerbosity=1"),
		fmt.Sprintf("--set=linux.customUserAgent=csi-e2e"),
		fmt.Sprintf("--set=windows.customUserAgent=csi-e2e"),
		"--wait",
		"--timeout=5m",
		"--debug",
	})

	if input.Config.IsWindowsTest {
		args = append(args,
			fmt.Sprintf("--set=windows.enabled=true"),
			fmt.Sprintf("--set=secrets-store-csi-driver.windows.enabled=true"))
	}

	args = append(args, generateValueArgs(input.Config)...)

	err = helm(args)
	Expect(err).To(BeNil())
}

// Uninstall uninstalls csi-secrets-store-provider-azure via Helm 3.
func Uninstall() {
	args := []string{
		"uninstall",
		chartName,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"--debug",
	}

	// ignore error to allow cleanup completion
	_ = helm(args)
}

// ReleaseExists checks if csi release exists
func ReleaseExists() bool {
	args := []string{
		"status",
		chartName,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
	}

	err := helm(args)
	// chart not found error
	return err == nil
}

func generateValueArgs(config *framework.Config) []string {
	args := []string{
		fmt.Sprintf("--set=linux.image.repository=%s/%s", config.Registry, config.ImageName),
		fmt.Sprintf("--set=windows.image.repository=%s/%s", config.Registry, config.ImageName),
	}

	//Set image.tag only if there is an image version provided. Else rely on default values.
	if config.ImageVersion != "" {
		args = append(args, fmt.Sprintf("--set=linux.image.tag=%s", config.ImageVersion), fmt.Sprintf("--set=windows.image.tag=%s", config.ImageVersion))
	}

	if config.DriverWriteSecrets {
		args = append(args, fmt.Sprintf("--set=driverWriteSecrets=true"))
	}

	// add the custom env file volume and mount if exists
	if config.AzureEnvironmentFilePath != "" {
		args = append(args,
			fmt.Sprintf("--set=linux.volumes[0].name=cloudenvfile-vol,linux.volumes[0].hostPath.path=%s,linux.volumes[0].hostPath.type=File", config.AzureEnvironmentFilePath),
			fmt.Sprintf("--set=linux.volumeMounts[0].name=cloudenvfile-vol,linux.volumeMounts[0].mountPath=%s", config.AzureEnvironmentFilePath),
		)
	}
	return args
}

func helm(args []string) error {
	By(fmt.Sprintf("helm %s", strings.Join(args, " ")))

	cmd := exec.Command("helm", args...)
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s", stdoutStderr)

	return err
}

// InstallPodIdentity installs aad-pod-identity via Helm3
func InstallPodIdentity() {
	// add aad-pod-identity chart repo
	args := append([]string{
		"repo", "add", "aad-pod-identity", "https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts",
	})
	err := helm(args)
	Expect(err).To(BeNil())

	// update helm repo
	args = append([]string{
		"repo", "update",
	})
	err = helm(args)
	Expect(err).To(BeNil())

	// Install aad-pod-identity helm chart
	args = append([]string{
		"install",
		podIdentityChartName,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"aad-pod-identity/aad-pod-identity", "--set", "nmi.allowNetworkPluginKubenet=true", "--wait", "--timeout=5m", "--debug",
	})
	err = helm(args)
	Expect(err).To(BeNil())
}

// UninstallPodIdentity uninstalls aad-pod-identity
func UninstallPodIdentity() {
	args := []string{
		"uninstall",
		podIdentityChartName,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"--debug",
	}

	// ignore error to allow cleanup completion
	_ = helm(args)
}
