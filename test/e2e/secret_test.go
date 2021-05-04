// +build e2e

package e2e

import (
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secret"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("When deploying SecretProviderClass CRD with secrets", func() {
	var (
		specName             = "secret"
		spc                  *v1alpha1.SecretProviderClass
		ns                   *corev1.Namespace
		nodePublishSecretRef *corev1.Secret
		p                    *corev1.Pod
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		nodePublishSecretRef = secret.Create(secret.CreateInput{
			Creator:   kubeClient,
			Name:      "secrets-store-creds",
			Namespace: ns.Name,
			Data:      map[string][]byte{"clientid": []byte(config.AzureClientID), "clientsecret": []byte(config.AzureClientSecret)},
		})

		keyVaultObjects := []provider.KeyVaultObject{
			{
				ObjectName: "secret1",
				ObjectType: provider.VaultObjectTypeSecret,
			},
			{
				ObjectName:  "secret1",
				ObjectType:  provider.VaultObjectTypeSecret,
				ObjectAlias: "SECRET_1",
			},
		}

		yamlArray := provider.StringArray{Array: []string{}}
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
					"keyvaultName": config.KeyvaultName,
					"tenantId":     config.TenantID,
					"objects":      string(objects),
				},
			},
		})

		p = pod.Create(pod.CreateInput{
			Creator:                  kubeClient,
			Config:                   config,
			Name:                     "busybox-secrets-store-inline-crd",
			Namespace:                ns.Name,
			SecretProviderClassName:  spc.Name,
			NodePublishSecretRefName: nodePublishSecretRef.Name,
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

	It("should read secret from pod", func() {
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

	It("should read secret from pod with alias", func() {
		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/SECRET_1")
		secret, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(secret).To(Equal(config.SecretValue))
	})
})
