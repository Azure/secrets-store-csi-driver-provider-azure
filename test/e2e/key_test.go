//go:build e2e
// +build e2e

package e2e

import (
	"strings"

	"github.com/ghodss/yaml"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secret"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"

	. "github.com/onsi/gomega"
)

var _ = Describe("When deploying SecretProviderClass CRD with keys", func() {
	var (
		specName             = "key"
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
			Labels:    map[string]string{"secrets-store.csi.k8s.io/used": "true"},
		})

		keyVaultObjects := []provider.KeyVaultObject{
			{
				ObjectName: "key1",
				ObjectType: provider.VaultObjectTypeKey,
			},
			{
				ObjectName:  "key1",
				ObjectType:  provider.VaultObjectTypeKey,
				ObjectAlias: "KEY_1",
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
	})

	AfterEach(func() {
		Cleanup(CleanupInput{
			Namespace: ns,
			Getter:    kubeClient,
			Lister:    kubeClient,
			Deleter:   kubeClient,
		})
	})

	It("should read key from pod", func() {
		p = pod.Create(pod.CreateInput{
			Creator:                  kubeClient,
			Config:                   config,
			Name:                     "busybox-secrets-store-inline-crd",
			Namespace:                ns.Name,
			SecretProviderClassName:  spc.Name,
			NodePublishSecretRefName: nodePublishSecretRef.Name,
		})

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/key1")
		key, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(key).To(ContainSubstring(config.KeyValue))
	})

	It("should read key from pod with alias", func() {
		p = pod.Create(pod.CreateInput{
			Creator:                  kubeClient,
			Config:                   config,
			Name:                     "busybox-secrets-store-inline-crd",
			Namespace:                ns.Name,
			SecretProviderClassName:  spc.Name,
			NodePublishSecretRefName: nodePublishSecretRef.Name,
		})

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/KEY_1")
		key, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(key).To(ContainSubstring(config.KeyValue))
	})

	It("should read RSA-HSM key from pod", func() {
		if config.ImageVersion <= "0.0.14" {
			Skip("functionality not yet supported in release version")
		}

		// update the secretproviderclass to reference rsa-hsm keys
		keyVaultObjects := []provider.KeyVaultObject{
			{
				ObjectName: "rsahsmkey1",
				ObjectType: provider.VaultObjectTypeKey,
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

		spc.Spec.Parameters["objects"] = string(objects)
		spc = secretproviderclass.Update(secretproviderclass.UpdateInput{
			Updater:             kubeClient,
			SecretProviderClass: spc,
		})

		p = pod.Create(pod.CreateInput{
			Creator:                  kubeClient,
			Config:                   config,
			Name:                     "busybox-secrets-store-inline-crd",
			Namespace:                ns.Name,
			SecretProviderClassName:  spc.Name,
			NodePublishSecretRefName: nodePublishSecretRef.Name,
		})

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/rsahsmkey1")
		key, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(key).NotTo(BeEmpty())
		Expect(key).To(ContainSubstring("BEGIN PUBLIC KEY"))
		Expect(key).To(ContainSubstring("END PUBLIC KEY"))
	})

	It("should read EC-HSM key from pod", func() {
		if config.ImageVersion <= "0.0.14" {
			Skip("functionality not yet supported in release version")
		}

		// update the secretproviderclass to reference rsa-hsm keys
		keyVaultObjects := []provider.KeyVaultObject{
			{
				ObjectName: "echsmkey1",
				ObjectType: provider.VaultObjectTypeKey,
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

		spc.Spec.Parameters["objects"] = string(objects)
		spc = secretproviderclass.Update(secretproviderclass.UpdateInput{
			Updater:             kubeClient,
			SecretProviderClass: spc,
		})

		p = pod.Create(pod.CreateInput{
			Creator:                  kubeClient,
			Config:                   config,
			Name:                     "busybox-secrets-store-inline-crd",
			Namespace:                ns.Name,
			SecretProviderClassName:  spc.Name,
			NodePublishSecretRefName: nodePublishSecretRef.Name,
		})

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/echsmkey1")
		key, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(key).NotTo(BeEmpty())
		Expect(key).To(ContainSubstring("BEGIN PUBLIC KEY"))
		Expect(key).To(ContainSubstring("END PUBLIC KEY"))
	})
})
