//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"testing"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/deploy"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/helm"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/keyvault"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
	e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clusterProxy   framework.ClusterProxy
	config         *framework.Config
	kubeClient     client.Client
	clientSet      *kubernetes.Clientset
	kvClient       keyvault.Client
	kubeconfigPath string
	coreNamespaces = []string{
		framework.NamespaceKubeSystem,
	}
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
	kubeClient = clusterProxy.GetClient()
	clientSet = clusterProxy.GetClientSet()
	kubeconfigPath = clusterProxy.GetKubeconfigPath()

	By("Creating a Keyvault Client")
	kvClient = keyvault.NewClient(config)

	if config.IsSoakTest || config.IsArcTest {
		return
	}

	if !config.IsHelmTest {
		By("Installing Secrets Store CSI Driver and Azure Key Vault Provider via kubectl from deployment manifest.")
		deploy.InstallManifest(kubeconfigPath, config)

		return
	}

	// if helm release exists, it means either cluster upgrade test or backward compatibility test is underway
	if !helm.ReleaseExists() {
		By(fmt.Sprintf("Installing Secrets Store CSI Driver and Azure Key Vault Provider via Helm from - %s.", config.HelmChartDir))
		helm.Install(helm.InstallInput{
			Config: config,
		})
	} else if config.IsBackwardCompatibilityTest {
		// upgrade helm charts only if running backward compatibility tests
		By(fmt.Sprintf("Upgrading Secrets Store CSI Driver and Azure Key Vault Provider via Helm to New Version from - %s.", config.HelmChartDir))
		helm.Upgrade(helm.UpgradeInput{
			Config: config,
		})
	}

	// Ensure all pods are running and ready before starting tests
	podStartupTimeout := e2eframework.TestContext.SystemPodsStartupTimeout
	for _, namespace := range coreNamespaces {
		if err := e2epod.WaitForPodsRunningReady(clientSet, namespace, int32(e2eframework.TestContext.MinStartupPods), int32(e2eframework.TestContext.AllowedNotReadyNodes), podStartupTimeout, map[string]string{}); err != nil {
			e2eframework.DumpAllNamespaceInfo(clientSet, namespace)
			e2ekubectl.LogFailedContainers(clientSet, namespace, e2eframework.Logf)
			e2eframework.Failf("error waiting for all pods to be running and ready: %v", err)
		}
	}
})

var _ = AfterSuite(func() {
	// cleanup
	defer func() {
		// uninstall if it's not Soak Test, not backward compatibility test and if cluster is already upgraded or it's not cluster upgrade test
		if !config.IsSoakTest && !config.IsArcTest && !config.IsBackwardCompatibilityTest && (!config.IsUpgradeTest || config.IsClusterUpgraded) {
			if helm.ReleaseExists() {
				By("Uninstalling Secrets Store CSI Driver and Azure Key Vault Provider via Helm")
				helm.Uninstall()
			}
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
