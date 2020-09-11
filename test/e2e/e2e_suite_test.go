// +build e2e

package e2e

import (
	"fmt"
	"testing"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/helm"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/keyvault"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"

	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clusterProxy   framework.ClusterProxy
	config         *framework.Config
	kubeClient     client.Client
	kvClient       keyvault.Client
	kubeconfigPath string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "sscdproviderazure")
}

var _ = BeforeSuite(func() {
	By("Parsing test configuration")
	var err error
	config, err = framework.ParseConfig()
	Expect(err).To(BeNil())

	By("Creating a Cluster Proxy")
	clusterProxy = framework.NewClusterProxy(initScheme())

	By("Creating a Keyvault Client")
	kvClient = keyvault.NewClient(config)

	if !config.IsSoakTest {
		By("Installing Secrets Store CSI Driver and Azure Key Vault Provider via Helm")
		helm.Install(helm.InstallInput{
			Config: config,
		})
	}

	kubeClient = clusterProxy.GetClient()
	kubeconfigPath = clusterProxy.GetKubeconfigPath()
})

var _ = AfterSuite(func() {
	defer func() {
		if !config.IsSoakTest {
			By("Uninstalling Secrets Store CSI Driver and Azure Key Vault Provider via Helm")
			helm.Uninstall()
		}
	}()

	dumpLogs()
})

func initScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	framework.TryAddDefaultSchemes(scheme)
	return scheme
}

func dumpLogs() {
	for component, containers := range map[string][]string{
		"secrets-store-csi-driver":         {"node-driver-registrar", "secrets-store", "liveness-probe"},
		"csi-secrets-store-provider-azure": {"provider-azure-installer"},
	} {
		podList := pod.List(pod.ListInput{
			Lister:    kubeClient,
			Namespace: framework.NamespaceKubeSystem,
			Labels: map[string]string{
				"app": component,
			},
		})

		for _, p := range podList.Items {
			for _, containerName := range containers {
				By(fmt.Sprintf("Dumping logs for %s scheduled to %s, container %s", p.Name, p.Spec.NodeName, containerName))
				out, err := exec.KubectlLogs(kubeconfigPath, p.Name, containerName, framework.NamespaceKubeSystem)
				Expect(err).To(BeNil())
				fmt.Println(out + "\n")
			}
		}
	}
}

// getPodExecCommand returns the exec command to use for validating mount contents
func getPodExecCommand(cmd string) string {
	if config.IsWindowsTest {
		return fmt.Sprintf("powershell.exe -Command %s", cmd)
	}
	return cmd
}
