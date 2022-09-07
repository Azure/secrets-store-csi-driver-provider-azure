//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secret"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("When user provides file permission for secrets", func() {
	var (
		specName               = "secret-file-permission"
		spc                    *v1alpha1.SecretProviderClass
		ns                     *corev1.Namespace
		nodePublishSecretRef   *corev1.Secret
		p                      *corev1.Pod
		expectedFilePermission = "755"
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

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName:     "secret1",
				ObjectType:     types.VaultObjectTypeSecret,
				FilePermission: fmt.Sprintf("0%s", expectedFilePermission),
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

	It("should mount secret file with given permission", func() {
		if !config.IsKindCluster {
			Skip("test case currently supported for kind cluster only")
		}

		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("stat -c '%a' /mnt/secrets-store/..data/secret1")
		filePermission, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		Expect(strings.Trim(filePermission, "'")).To(Equal(expectedFilePermission))
	})
})
