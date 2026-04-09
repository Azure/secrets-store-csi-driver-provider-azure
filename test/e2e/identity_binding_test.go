//go:build e2e
// +build e2e

package e2e

import (
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/clusterrole"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/clusterrolebinding"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/serviceaccount"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("CSI inline volume test with identity binding", func() {
	var (
		specName = "identitybinding"
		spc      *v1alpha1.SecretProviderClass
		ns       *corev1.Namespace
		p        *corev1.Pod
		sa       *corev1.ServiceAccount
		cr       *rbacv1.ClusterRole
		crb      *rbacv1.ClusterRoleBinding
	)

	BeforeEach(func() {
		if config.IsArcTest {
			Skip("test is not supported in Arc cluster")
		}
		if config.IsKindCluster {
			Skip("test is not supported on kind cluster")
		}

		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: "secret1",
				ObjectType: types.VaultObjectTypeSecret,
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
					// Enable identity binding
					types.UseAzureTokenProxyParameter: "true",
				},
			},
		})

		// Create service account with workload identity annotations
		sa = serviceaccount.Create(serviceaccount.CreateInput{
			Creator:   kubeClient,
			Name:      "identity-binding-sa",
			Namespace: ns.Name,
			Annotations: map[string]string{
				"azure.workload.identity/client-id": config.AzureClientID,
			},
		})

		// Create ClusterRole to allow using the managed identity
		// API group: cid.wi.aks.azure.com
		// Resource: the client ID of the managed identity
		// Verb: use-managed-identity
		cr = clusterrole.Create(clusterrole.CreateInput{
			Creator: kubeClient,
			Name:    "use-mi-" + config.AzureClientID + "-" + ns.Name,
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"cid.wi.aks.azure.com"},
					Resources: []string{config.AzureClientID},
					Verbs:     []string{"use-managed-identity"},
				},
			},
		})

		// Create ClusterRoleBinding to bind the role to the service account
		crb = clusterrolebinding.Create(clusterrolebinding.CreateInput{
			Creator: kubeClient,
			Name:    "use-mi-" + config.AzureClientID + "-" + ns.Name,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     cr.Name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      sa.Name,
					Namespace: ns.Name,
				},
			},
		})

		// Create pod for identity binding
		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline-identity-binding",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
			ServiceAccountName:      sa.Name,
		})
	})

	AfterEach(func() {
		// Clean up ClusterRole and ClusterRoleBinding first since they are cluster-scoped
		if crb != nil {
			clusterrolebinding.Delete(clusterrolebinding.DeleteInput{
				Deleter:            kubeClient,
				ClusterRoleBinding: crb,
			})
		}
		if cr != nil {
			clusterrole.Delete(clusterrole.DeleteInput{
				Deleter:     kubeClient,
				ClusterRole: cr,
			})
		}

		Cleanup(CleanupInput{
			Namespace: ns,
			Getter:    kubeClient,
			Lister:    kubeClient,
			Deleter:   kubeClient,
		})
	})

	It("should read secret from pod using identity binding", func() {
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
	})
})
