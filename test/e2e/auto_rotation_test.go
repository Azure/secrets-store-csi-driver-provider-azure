//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/helm"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secret"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

const (
	randomLength = 5
)

var _ = Describe("Test auto rotation of mount contents and K8s secrets", func() {
	var (
		specName = "autorotation"
		spc      *v1alpha1.SecretProviderClass
		ns       *corev1.Namespace
		p        *corev1.Pod
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
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

	It("should auto rotate mount contents with service principal", func() {
		if config.IsKindCluster {
			Skip("test case not supported for kind cluster")
		}

		nodePublishSecretRef := secret.Create(secret.CreateInput{
			Creator:   kubeClient,
			Name:      "secrets-store-creds",
			Namespace: ns.Name,
			Data:      map[string][]byte{"clientid": []byte(config.AzureClientID), "clientsecret": []byte(config.AzureClientSecret)},
			Labels:    map[string]string{"secrets-store.csi.k8s.io/used": "true"},
		})

		secretName := fmt.Sprintf("secret-sp-%s", utilrand.String(randomLength))
		// create secret in keyvault
		err := kvClient.SetSecret(secretName, "secret")
		Expect(err).To(BeNil())
		defer func() {
			err = kvClient.DeleteSecret(secretName)
			Expect(err).To(BeNil())
		}()

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: secretName,
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
				SecretObjects: []*v1alpha1.SecretObject{
					{
						SecretName: "rotationsecret",
						Type:       string(corev1.SecretTypeOpaque),
						Labels:     map[string]string{"environment": "test"},
						Data: []*v1alpha1.SecretObjectData{
							{
								ObjectName: secretName,
								Key:        "foo",
							},
						},
					},
				},
				Parameters: map[string]string{
					types.KeyVaultNameParameter:         config.KeyvaultName,
					types.TenantIDParameter:             config.TenantID,
					types.ObjectsParameter:              string(objects),
					types.UsePodIdentityParameter:       "false",
					types.UseVMManagedIdentityParameter: "false",
				},
			},
		})

		p = pod.Create(pod.CreateInput{
			Creator:                  kubeClient,
			Config:                   config,
			Name:                     "busybox-secrets-store-inline",
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

		// check the secret is correct
		cmd := getPodExecCommand(fmt.Sprintf("cat /mnt/secrets-store/%s", secretName))
		out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(out).To(Equal("secret"))

		// check K8s secret is valid
		k8sSecret := secret.Get(secret.GetInput{
			Getter:    kubeClient,
			Name:      "rotationsecret",
			Namespace: ns.Name,
		})

		Expect(k8sSecret).NotTo(BeNil())
		Expect(k8sSecret.Data).NotTo(BeNil())
		Expect(string(k8sSecret.Data["foo"])).To(Equal("secret"))

		// rotate secret in key vault
		err = kvClient.SetSecret(secretName, "rotated")
		Expect(err).To(BeNil())

		Eventually(func() bool {
			// wait for secret to be updated
			cmd = getPodExecCommand(fmt.Sprintf("cat /mnt/secrets-store/%s", secretName))
			out, err = exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			// check K8s secret is valid
			k8sSecret := secret.Get(secret.GetInput{
				Getter:    kubeClient,
				Name:      "rotationsecret",
				Namespace: ns.Name,
			})

			Expect(k8sSecret).NotTo(BeNil())
			Expect(k8sSecret.Data).NotTo(BeNil())

			return out == string(k8sSecret.Data["foo"]) && out == "rotated"
		}, 2*time.Minute, 15*time.Second).Should(BeTrue())
	})

	It("should auto rotate mount contents with user managed identity", func() {
		if config.IsKindCluster {
			Skip("test case not supported for kind cluster")
		}

		secretName := fmt.Sprintf("secret-msi-%s", utilrand.String(randomLength))
		// create secret in keyvault
		err := kvClient.SetSecret(secretName, "secret")
		Expect(err).To(BeNil())
		defer func() {
			err = kvClient.DeleteSecret(secretName)
			Expect(err).To(BeNil())
		}()

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: secretName,
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
				SecretObjects: []*v1alpha1.SecretObject{
					{
						SecretName: "rotationsecret",
						Type:       string(corev1.SecretTypeOpaque),
						Labels:     map[string]string{"environment": "test"},
						Data: []*v1alpha1.SecretObjectData{
							{
								ObjectName: secretName,
								Key:        "foo",
							},
						},
					},
				},
				Parameters: map[string]string{
					types.KeyVaultNameParameter:           config.KeyvaultName,
					types.TenantIDParameter:               config.TenantID,
					types.ObjectsParameter:                string(objects),
					types.UsePodIdentityParameter:         "false",
					types.UseVMManagedIdentityParameter:   "true",
					types.UserAssignedIdentityIDParameter: config.UserAssignedIdentityID,
				},
			},
		})

		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline",
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

		// check the secret is correct
		cmd := getPodExecCommand(fmt.Sprintf("cat /mnt/secrets-store/%s", secretName))
		out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(out).To(Equal("secret"))

		// check K8s secret is valid
		k8sSecret := secret.Get(secret.GetInput{
			Getter:    kubeClient,
			Name:      "rotationsecret",
			Namespace: ns.Name,
		})

		Expect(k8sSecret).NotTo(BeNil())
		Expect(k8sSecret.Data).NotTo(BeNil())
		Expect(string(k8sSecret.Data["foo"])).To(Equal("secret"))

		// rotate secret in key vault
		err = kvClient.SetSecret(secretName, "rotated")
		Expect(err).To(BeNil())

		Eventually(func() bool {
			// wait for secret to be updated
			cmd = getPodExecCommand(fmt.Sprintf("cat /mnt/secrets-store/%s", secretName))
			out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			// check K8s secret is valid
			k8sSecret := secret.Get(secret.GetInput{
				Getter:    kubeClient,
				Name:      "rotationsecret",
				Namespace: ns.Name,
			})

			Expect(k8sSecret).NotTo(BeNil())
			Expect(k8sSecret.Data).NotTo(BeNil())

			return out == string(k8sSecret.Data["foo"]) && out == "rotated"
		}, 2*time.Minute, 15*time.Second).Should(BeTrue())
	})

	It("should auto rotate mount contents with pod identity", func() {
		if config.IsKindCluster {
			Skip("test case not supported for kind cluster")
		}
		if config.IsWindowsTest {
			Skip("test case not supported for windows cluster")
		}
		if config.IsArcTest {
			Skip("test is not supported in Arc cluster")
		}

		secretName := fmt.Sprintf("secret-pi-%s", utilrand.String(randomLength))
		// create secret in keyvault
		err := kvClient.SetSecret(secretName, "secret")
		Expect(err).To(BeNil())
		defer func() {
			err = kvClient.DeleteSecret(secretName)
			Expect(err).To(BeNil())
		}()

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: secretName,
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
				SecretObjects: []*v1alpha1.SecretObject{
					{
						SecretName: "rotationsecret",
						Type:       string(corev1.SecretTypeOpaque),
						Labels:     map[string]string{"environment": "test"},
						Data: []*v1alpha1.SecretObjectData{
							{
								ObjectName: secretName,
								Key:        "foo",
							},
						},
					},
				},
				Parameters: map[string]string{
					types.KeyVaultNameParameter:         config.KeyvaultName,
					types.TenantIDParameter:             config.TenantID,
					types.ObjectsParameter:              string(objects),
					types.UsePodIdentityParameter:       "true",
					types.UseVMManagedIdentityParameter: "false",
				},
			},
		})

		By("Deploying aad-pod-identity")
		helm.InstallPodIdentity()
		defer helm.UninstallPodIdentity()

		By("Deploying AzureIdentity and AzureIdentityBinding")

		type identityAndBinding struct {
			AzureIdentityName string
			Namespace         string
			SubscriptionID    string
			ResourceGroup     string
			UserMSIName       string
			ClientID          string
			Selector          string
		}

		data := &identityAndBinding{
			AzureIdentityName: p.Name,
			Namespace:         ns.Name,
			SubscriptionID:    config.SubscriptionID,
			ResourceGroup:     config.ResourceGroup,
			UserMSIName:       config.PodIdentityUserMSIName,
			ClientID:          config.PodIdentityUserAssignedIdentityID,
			Selector:          ns.Name,
		}

		azureIdentityFile, err := os.CreateTemp("", "")
		Expect(err).To(BeNil())
		azureIdentityBindingFile, err := os.CreateTemp("", "")
		Expect(err).To(BeNil())
		defer func() {
			os.Remove(azureIdentityFile.Name())
			os.Remove(azureIdentityBindingFile.Name())
		}()

		tl, err := template.New("").Parse(azureIdentityTemplate)
		Expect(err).To(BeNil())
		err = tl.Execute(azureIdentityFile, &data)
		Expect(err).To(BeNil())

		tl, err = template.New("").Parse(azureIdentityBindingTemplate)
		Expect(err).To(BeNil())
		err = tl.Execute(azureIdentityBindingFile, &data)
		Expect(err).To(BeNil())

		err = exec.KubectlApply(kubeconfigPath, ns.Name, []string{"-f", azureIdentityFile.Name()})
		Expect(err).To(BeNil())
		err = exec.KubectlApply(kubeconfigPath, ns.Name, []string{"-f", azureIdentityBindingFile.Name()})
		Expect(err).To(BeNil())

		defer func() {
			err = exec.KubectlDelete(kubeconfigPath, ns.Name, []string{"-f", azureIdentityFile.Name()})
			Expect(err).To(BeNil())
			err = exec.KubectlDelete(kubeconfigPath, ns.Name, []string{"-f", azureIdentityBindingFile.Name()})
			Expect(err).To(BeNil())
		}()

		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "busybox-secrets-store-inline",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
			Labels:                  map[string]string{"aadpodidbinding": ns.Name},
		})

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		// check the secret is correct
		cmd := getPodExecCommand(fmt.Sprintf("cat /mnt/secrets-store/%s", secretName))
		out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(out).To(Equal("secret"))

		// check K8s secret is valid
		k8sSecret := secret.Get(secret.GetInput{
			Getter:    kubeClient,
			Name:      "rotationsecret",
			Namespace: ns.Name,
		})

		Expect(k8sSecret).NotTo(BeNil())
		Expect(k8sSecret.Data).NotTo(BeNil())
		Expect(string(k8sSecret.Data["foo"])).To(Equal("secret"))

		// rotate secret in key vault
		err = kvClient.SetSecret(secretName, "rotated")
		Expect(err).To(BeNil())

		Eventually(func() bool {
			// wait for secret to be updated
			cmd = getPodExecCommand(fmt.Sprintf("cat /mnt/secrets-store/%s", secretName))
			out, err = exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			// check K8s secret is valid
			k8sSecret := secret.Get(secret.GetInput{
				Getter:    kubeClient,
				Name:      "rotationsecret",
				Namespace: ns.Name,
			})

			Expect(k8sSecret).NotTo(BeNil())
			Expect(k8sSecret.Data).NotTo(BeNil())

			return out == string(k8sSecret.Data["foo"]) && out == "rotated"
		}, 2*time.Minute, 15*time.Second).Should(BeTrue())
	})
})
