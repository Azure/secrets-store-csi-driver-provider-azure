//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/deploy"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/helm"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/keyvault"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
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
	// anySpecFailed is set to true by an AfterEach hook whenever a spec
	// fails. The post-suite stability check uses this to avoid burying
	// the real failure under cascading assertions.
	anySpecFailed bool
)

const (
	// podStartTimeout is how long to wait for the pod to be started.
	podStartTimeout = 5 * time.Minute

	// defaultPostSuiteSoak is the soak window applied after all specs
	// complete before re-asserting that driver and provider pods are
	// still healthy with zero container restarts. The window covers the
	// default Windows livenessProbe cadence (failureThreshold 3 x
	// periodSeconds 30 = ~90s) plus headroom, so a probe regression like
	// issue #2029 surfaces here even if it does not fire during the
	// regular suite.
	defaultPostSuiteSoak = 2 * time.Minute

	// postSuiteSoakEnv overrides defaultPostSuiteSoak. Accepts any
	// time.ParseDuration value. Set to "0" to skip the soak entirely
	// (useful for local iteration).
	postSuiteSoakEnv = "E2E_POST_SUITE_SOAK"
)

// providerAndDriverAppLabels is the set of `app` label values used by the
// driver and Azure provider DaemonSets. Kept as a package var so the
// BeforeSuite readiness wait and the AfterSuite stability check stay in
// sync.
var providerAndDriverAppLabels = []string{
	"secrets-store-csi-driver",
	"csi-secrets-store-provider-azure",
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "sscdproviderazure")
}

var _ = AfterEach(func() {
	if CurrentSpecReport().Failed() {
		anySpecFailed = true
	}
})

var _ = BeforeSuite(func(ctx context.Context) {
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

	// Ensure driver and provider pods are running and ready before starting tests
	By("Waiting for driver and provider pods to be running and ready before starting tests.")
	driverAndProviderLabels, err := labels.NewRequirement("app", selection.In, providerAndDriverAppLabels)
	Expect(err).To(BeNil())

	podSelector := labels.NewSelector().Add(*driverAndProviderLabels)

	if _, err := e2epod.WaitForPodsWithLabelRunningReady(ctx, clientSet, framework.NamespaceKubeSystem, podSelector, 1, podStartTimeout); err != nil {
		e2eframework.Failf("error waiting for driver and provider pods to be running and ready: %v", err)
	}
})

var _ = AfterSuite(func(ctx context.Context) {
	// Run uninstall LAST and dumpLogs SECOND-TO-LAST so they execute
	// even if the post-suite stability check fails — otherwise a
	// regression like issue #2029 would tear down the cluster before
	// we capture the evidence needed to debug it.
	defer func() {
		// uninstall if it's not Soak Test, not backward compatibility test and if cluster is already upgraded or it's not cluster upgrade test
		if !config.IsSoakTest && !config.IsArcTest && !config.IsBackwardCompatibilityTest && (!config.IsUpgradeTest || config.IsClusterUpgraded) {
			if helm.ReleaseExists() {
				By("Uninstalling Secrets Store CSI Driver and Azure Key Vault Provider via Helm")
				helm.Uninstall()
			}
		}
	}()
	defer dumpLogs()

	runPostSuiteStabilityCheck(ctx)
})

// runPostSuiteStabilityCheck asserts that the driver and provider
// DaemonSets remained healthy across the suite and stay healthy through
// a short soak window. It is the safety net for liveness/readiness
// regressions (see issue #2029, where the provider's healthz dial was
// broken on Windows and crash-looped the pod every ~90s).
//
// It deliberately skips:
//   - runs where a spec already failed (the real failure is upstream,
//     and we do not want to bury it with cascading assertions);
//   - test modes that legitimately churn provider pods (soak, cluster
//     upgrade) or do not install the provider in the usual shape (arc);
//   - runs against a previously-published chart (IsReleasedVersionTest),
//     since we do not want a known bug shipped in v1.x to permanently
//     red-light this safety net on every PR. Regressions on the
//     candidate change are still caught by the "Run e2e test with New
//     Version" invocation, which does not set IS_RELEASED_VERSION_TEST.
func runPostSuiteStabilityCheck(ctx context.Context) {
	if anySpecFailed {
		By("Skipping post-suite stability check: at least one spec already failed")
		return
	}
	if config.IsSoakTest || config.IsArcTest || config.IsReleasedVersionTest || config.IsUpgradeTest {
		By("Skipping post-suite stability check: not applicable for this test mode")
		return
	}

	soak := defaultPostSuiteSoak
	if v, ok := os.LookupEnv(postSuiteSoakEnv); ok {
		parsed, err := time.ParseDuration(v)
		Expect(err).To(BeNil(), "invalid %s=%q", postSuiteSoakEnv, v)
		soak = parsed
	}

	By("Verifying driver and provider pods have zero container restarts after the suite")
	pod.AssertNoRestarts(pod.AssertNoRestartsInput{
		Lister:         kubeClient,
		KubeconfigPath: kubeconfigPath,
		Namespace:      framework.NamespaceKubeSystem,
		AppLabels:      providerAndDriverAppLabels,
	})

	if soak <= 0 {
		return
	}

	By(fmt.Sprintf("Soaking for %s to catch slow-burn liveness/healthz regressions", soak))
	select {
	case <-time.After(soak):
	case <-ctx.Done():
		return
	}

	By("Verifying driver and provider pods are still Ready with zero container restarts after soak")
	pod.AssertNoRestarts(pod.AssertNoRestartsInput{
		Lister:         kubeClient,
		KubeconfigPath: kubeconfigPath,
		Namespace:      framework.NamespaceKubeSystem,
		AppLabels:      providerAndDriverAppLabels,
		VerifyReady:    true,
	})
}

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
