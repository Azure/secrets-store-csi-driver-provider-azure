// +build e2e

package e2e

import (
	"html/template"
	"os"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/helm"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

const (
	azureIdentityTemplate = `
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentity
metadata:
  name: {{.AzureIdentityName}}
  namespace: {{.Namespace}}
spec:
  type: 0
  resourceID: /subscriptions/{{.SubscriptionID}}/resourcegroups/{{.ResourceGroup}}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{{.UserMSIName}}
  clientID: {{.ClientID}}`

	azureIdentityBindingTemplate = `
apiVersion: "aadpodidentity.k8s.io/v1"
kind: AzureIdentityBinding
metadata:
  name: {{.AzureIdentityName}}-binding
  namespace: {{.Namespace}}
spec:
  azureIdentity: {{.AzureIdentityName}}
  selector: {{.Selector}}`
)

var _ = Describe("CSI inline volume test with aad-pod-identity", func() {
	var (
		specName = "podidentity"
		spc      *v1alpha1.SecretProviderClass
		ns       *corev1.Namespace
		p        *corev1.Pod
	)

	BeforeEach(func() {
		ns = namespace.Create(namespace.CreateInput{
			Creator: kubeClient,
			Name:    specName,
		})

		keyVaultObjects := []provider.KeyVaultObject{
			{
				ObjectName: "secret1",
				ObjectType: provider.VaultObjectTypeSecret,
			},
			{
				ObjectName: "key1",
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

		spc = secretproviderclass.Create(secretproviderclass.CreateInput{
			Creator:   kubeClient,
			Config:    config,
			Name:      "azure",
			Namespace: ns.Name,
			Spec: v1alpha1.SecretProviderClassSpec{
				Provider: "azure",
				Parameters: map[string]string{
					"keyvaultName":         config.KeyvaultName,
					"tenantId":             config.TenantID,
					"objects":              string(objects),
					"usePodIdentity":       "true",
					"useVMManagedIdentity": "false",
				},
			},
		})

		p = pod.Create(pod.CreateInput{
			Creator:                 kubeClient,
			Config:                  config,
			Name:                    "nginx-secrets-store-inline-pi",
			Namespace:               ns.Name,
			SecretProviderClassName: spc.Name,
			Labels:                  map[string]string{"aadpodidbinding": ns.Name},
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
		if config.IsKindCluster {
			Skip("test case not supported for kind cluster")
		}
		if config.IsWindowsTest {
			Skip("test case not supported for windows cluster")
		}

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
			err = os.Remove(azureIdentityFile.Name())
			Expect(err).To(BeNil())
			err = os.Remove(azureIdentityBindingFile.Name())
			Expect(err).To(BeNil())
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
