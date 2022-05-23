//go:build e2e
// +build e2e

package e2e

import (
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/serviceaccount"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("CSI inline volume test with workload identity", func() {
	var (
		specName = "workloadidentity"
		spc      *v1alpha1.SecretProviderClass
		ns       *corev1.Namespace
		p        *corev1.Pod
		sa       *corev1.ServiceAccount
	)

	BeforeEach(func() {
		// The preconfigured federated identity credential is for namespace: workloadidentity
		// service account name: workload-identity-sa. So creating the namespace with the
		// fixed name here.
		ns = namespace.CreateWithName(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: "secret1",
				ObjectType: types.VaultObjectTypeSecret,
			},
			{
				ObjectName: "key1",
				ObjectType: types.VaultObjectTypeKey,
			},
		}

		yamlArray := types.StringArray{Array: []string{}}
		for _, object := range keyVaultObjects {
			obj, err := yaml.Marshal(object)
			Expect(err).To(BeNil())
			yamlArray.Array = append(yamlArray.Array, string(obj))
		}

		objects, err := yaml.Marshal(yamlArray)
		Expect(err).To(BeNil())

		spc = secretproviderclass.Create(secretproviderclass.CreateInput{
			Creator:   kubeClient,
			Config:    config,
			Name:      "azure",
			Namespace: ns.Name,
			Spec: v1alpha1.SecretProviderClassSpec{
				Provider: "azure",
				Parameters: map[string]string{
					types.KeyVaultNameParameter:         config.KeyvaultName,
					types.TenantIDParameter:             config.TenantID,
					types.ObjectsParameter:              string(objects),
					types.UsePodIdentityParameter:       "false",
					types.UseVMManagedIdentityParameter: "false",
					types.ClientIDParameter:             config.AzureClientID,
				},
			},
		})

		sa = serviceaccount.Create(serviceaccount.CreateInput{
			Creator:   kubeClient,
			Name:      "workload-identity-sa",
			Namespace: ns.Name,
		})

		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline-wi",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
			ServiceAccountName:      sa.Name,
		})
	})

	AfterEach(func() {
		Cleanup(CleanupInput{
			Namespace: ns,
			Getter:    kubeClient,
			Lister:    kubeClient,
			Deleter:   kubeClient,
		})
	})

	It("should read secret, key from pod", func() {
		if config.IsArcTest {
			Skip("test is not supported in Arc cluster")
		}
		if !(config.IsKindCluster || config.IsSoakTest) {
			Skip("test case currently supported for kind and soak cluster only")
		}

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/secret1")
		secret, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(secret).To(Equal(config.SecretValue))

		cmd = getPodExecCommand("cat /mnt/secrets-store/key1")
		key, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(key).To(ContainSubstring(config.KeyValue))
	})
})
