// +build e2e

package utils

import (
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secret"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"

	. "github.com/onsi/gomega"
)

var (
	NodePublishSecretRefName = "secrets-store-creds"
	SecretNamespaeName       = "secret-az"
	SecretPodName            = "secret-busybox-secrets-store-inline-crd"
	SecretSPCName            = "secret-azure"
)

//Input provides input for util methods
type Input struct {
	Creator framework.Creator
	Config  *framework.Config
}

//SetupSecrets creates necessary resources in the cluster for testing
func SetupSecrets(input Input) {
	//Create secret namespace
	ns := namespace.Create(namespace.CreateInput{
		Creator: input.Creator,
		Name:    SecretNamespaeName,
		NS: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: SecretNamespaeName,
			},
		},
	})

	nodePublishSecretRef := secret.Create(secret.CreateInput{
		Creator:   input.Creator,
		Name:      NodePublishSecretRefName,
		Namespace: ns.Name,
		Data:      map[string][]byte{"clientid": []byte(input.Config.AzureClientID), "clientsecret": []byte(input.Config.AzureClientSecret)},
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

	spc := secretproviderclass.Create(secretproviderclass.CreateInput{
		Creator:   input.Creator,
		Config:    input.Config,
		Name:      SecretSPCName,
		Namespace: ns.Name,
		Spec: v1alpha1.SecretProviderClassSpec{
			Provider: "azure",
			Parameters: map[string]string{
				"keyvaultName": input.Config.KeyvaultName,
				"tenantId":     input.Config.TenantID,
				"objects":      string(objects),
			},
		},
	})

	pod.Create(pod.CreateInput{
		Creator:   input.Creator,
		Config:    input.Config,
		Name:      SecretPodName,
		Namespace: ns.Name,
		Labels: map[string]string{
			"secret-test": "e2e",
		},
		SecretProviderClassName:  spc.Name,
		NodePublishSecretRefName: nodePublishSecretRef.Name,
	})
}

func RecreateSecretPod(kubeClient client.Client, kubeconfigPath string, config *framework.Config) {
	//Check Is pod exists
	podList := pod.List(pod.ListInput{
		Lister:    kubeClient,
		Namespace: SecretNamespaeName,
		Labels: map[string]string{
			"secret-test": "e2e",
		},
	})
	Expect(podList.Items).NotTo(BeNil())

	if len(podList.Items) > 0 {
		if podList.Items[0].ObjectMeta.Name == SecretPodName {
			pod.Delete(pod.DeleteInput{
				Deleter: kubeClient,
				Pod:     &podList.Items[0],
			})
		}
	}

	pod.Create(pod.CreateInput{
		Creator:   kubeClient,
		Config:    config,
		Name:      SecretPodName,
		Namespace: SecretNamespaeName,
		Labels: map[string]string{
			"secret-test": "e2e",
		},
		SecretProviderClassName:  SecretSPCName,
		NodePublishSecretRefName: NodePublishSecretRefName,
	})

	pod.WaitFor(pod.WaitForInput{
		Getter:         kubeClient,
		KubeconfigPath: kubeconfigPath,
		Config:         config,
		PodName:        SecretPodName,
		Namespace:      SecretNamespaeName,
	})
}
