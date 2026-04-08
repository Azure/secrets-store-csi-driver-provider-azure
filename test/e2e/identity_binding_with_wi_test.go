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

var _ = Describe("CSI inline volume test with both workload identity and identity binding", func() {
	var (
		// Static namespace and SA name — must match the FIC configured in create-fic.yaml
		nsName = "identitybindingwithwi"
		saName = "dual-identity-sa"

		spcWI  *v1alpha1.SecretProviderClass
		spcIDB *v1alpha1.SecretProviderClass
		ns     *corev1.Namespace
		p      *corev1.Pod
		sa     *corev1.ServiceAccount
		cr     *rbacv1.ClusterRole
		crb    *rbacv1.ClusterRoleBinding
	)

	BeforeEach(func() {
		if config.IsArcTest {
			Skip("test is not supported in Arc cluster")
		}
		if config.IsKindCluster {
			Skip("test is not supported on kind cluster")
		}

		// Use a fixed namespace name so the FIC subject matches
		ns = namespace.CreateWithName(namespace.CreateInput{
			Creator: kubeClient,
			Name:    nsName,
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

		// SPC #1: Standard workload identity
		spcWI = secretproviderclass.Create(secretproviderclass.CreateInput{
			Creator:   kubeClient,
			Config:    config,
			Name:      "azure-wi",
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

		// SPC #2: Identity binding via Azure Token Proxy
		spcIDB = secretproviderclass.Create(secretproviderclass.CreateInput{
			Creator:   kubeClient,
			Config:    config,
			Name:      "azure-idb",
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
					types.UseAzureTokenProxyParameter: "true",
				},
			},
		})

		// Create service account with workload identity annotations
		sa = serviceaccount.Create(serviceaccount.CreateInput{
			Creator:   kubeClient,
			Name:      saName,
			Namespace: ns.Name,
			Annotations: map[string]string{
				"azure.workload.identity/client-id": config.AzureClientID,
			},
		})

		// Create ClusterRole to allow using the managed identity (required for identity binding)
		cr = clusterrole.Create(clusterrole.CreateInput{
			Creator: kubeClient,
			Name:    "use-mi-dual-" + config.AzureClientID + "-" + ns.Name,
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
			Name:    "use-mi-dual-" + config.AzureClientID + "-" + ns.Name,
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

		// Create pod with two volumes: workload identity + identity binding
		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-dual-identity",
			Namespace:               ns.Name,
			SecretProviderClassName: spcWI.Name,
			ServiceAccountName:      sa.Name,
			AdditionalVolumes: []pod.AdditionalVolume{
				{
					SecretProviderClassName: spcIDB.Name,
					VolumeName:              "secrets-store-idb",
					MountPath:               "/mnt/secrets-store-idb",
				},
			},
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

	It("should read secrets from both workload identity and identity binding volumes", func() {
		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		// Verify workload identity volume (default mount path)
		cmd := getPodExecCommand("cat /mnt/secrets-store/secret1")
		secret, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(secret).To(Equal(config.SecretValue))

		// Verify identity binding volume (additional mount path)
		cmd = getPodExecCommand("cat /mnt/secrets-store-idb/secret1")
		secret, err = exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(secret).To(Equal(config.SecretValue))
	})
})
