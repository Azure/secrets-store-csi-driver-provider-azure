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

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("When deploying SecretProviderClass CRD with keys", func() {
	var (
		specName = "key-test"
		spc      *v1alpha1.SecretProviderClass
		ns       *corev1.Namespace
		p        *corev1.Pod
	)

	BeforeEach(func() {
		ns = namespace.CreateWithName(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: "key1",
				ObjectType: types.VaultObjectTypeKey,
			},
			{
				ObjectName:  "key1",
				ObjectType:  types.VaultObjectTypeKey,
				ObjectAlias: "KEY_1",
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
					types.KeyVaultNameParameter: config.KeyvaultName,
					types.TenantIDParameter:     config.TenantID,
					types.ObjectsParameter:      string(objects),
					types.ClientIDParameter:     config.AzureClientID,
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
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline-crd",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
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
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline-crd",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
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
		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: "rsahsmkey1",
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

		spc.Spec.Parameters[types.ObjectsParameter] = string(objects)
		spc = secretproviderclass.Update(secretproviderclass.UpdateInput{
			Updater:             kubeClient,
			SecretProviderClass: spc,
		})

		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline-crd",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
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
		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: "echsmkey1",
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

		spc.Spec.Parameters[types.ObjectsParameter] = string(objects)
		spc = secretproviderclass.Update(secretproviderclass.UpdateInput{
			Updater:             kubeClient,
			SecretProviderClass: spc,
		})

		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline-crd",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
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
