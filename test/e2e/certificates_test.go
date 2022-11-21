//go:build e2e
// +build e2e

package e2e

import (
	"encoding/base64"
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/certificates"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/openssl"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/pod"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secret"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/secretproviderclass"

	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("When fetching certificates and private key from Key Vault", func() {
	var (
		specName             = "certificates"
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

		keyVaultObjects := []types.KeyVaultObject{
			{
				ObjectName: "pemcert1",
				ObjectType: types.VaultObjectTypeCertificate,
			},
			{
				ObjectName: "pkcs12cert1",
				ObjectType: types.VaultObjectTypeCertificate,
			},
			{
				ObjectName: "ecccert1",
				ObjectType: types.VaultObjectTypeCertificate,
			},
			{
				ObjectName:  "pemcert1",
				ObjectType:  types.VaultObjectTypeKey,
				ObjectAlias: "pemcert1-pub-key",
			},
			{
				ObjectName:  "pkcs12cert1",
				ObjectType:  types.VaultObjectTypeKey,
				ObjectAlias: "pkcs12cert1-pub-key",
			},
			{
				ObjectName:  "ecccert1",
				ObjectType:  types.VaultObjectTypeKey,
				ObjectAlias: "ecccert1-pub-key",
			},
			{
				ObjectName:  "pemcert1",
				ObjectType:  types.VaultObjectTypeSecret,
				ObjectAlias: "pemcert1-secret",
			},
			{
				ObjectName:  "pkcs12cert1",
				ObjectType:  types.VaultObjectTypeSecret,
				ObjectAlias: "pkcs12cert1-secret",
			},
			{
				ObjectName:  "ecccert1",
				ObjectType:  types.VaultObjectTypeSecret,
				ObjectAlias: "ecccert1-secret",
			},
			{
				ObjectName:   "pkcs12cert1",
				ObjectType:   types.VaultObjectTypeSecret,
				ObjectAlias:  "pkcs12cert1-secret-pfx",
				ObjectFormat: "pfx",
			},
			{
				ObjectName:   "ecccert1",
				ObjectType:   types.VaultObjectTypeSecret,
				ObjectAlias:  "ecccert1-secret-pfx",
				ObjectFormat: "pfx",
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
			Name:      "azure-certs",
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
			Name:                     "busybox-secrets-store-inline-crd-certs",
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

	It("should read pem cert, private and public key from pod", func() {
		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/pemcert1")
		cert, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		certificates.ValidateCert(cert, "test.domain.com")

		cmd = getPodExecCommand("cat /mnt/secrets-store/pemcert1-pub-key")
		pubKey, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())

		cmd = getPodExecCommand("cat /mnt/secrets-store/pemcert1-secret")
		out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		certificates.ValidateCertBundle(out, pubKey, out, "test.domain.com")
	})

	It("should read pkcs12 cert, private and public key from pod", func() {
		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/pkcs12cert1")
		cert, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		certificates.ValidateCert(cert, "test.domain.com")

		cmd = getPodExecCommand("cat /mnt/secrets-store/pkcs12cert1-pub-key")
		pubKey, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())

		cmd = getPodExecCommand("cat /mnt/secrets-store/pkcs12cert1-secret")
		out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		certificates.ValidateCertBundle(out, pubKey, out, "test.domain.com")

		cmd = getPodExecCommand("cat /mnt/secrets-store/pkcs12cert1-secret-pfx")
		out, err = exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		// base64 decode the pfx data
		pfxRaw, err := base64.StdEncoding.DecodeString(out)
		Expect(err).To(BeNil())
		// Convert pfx data to PEM
		pem, err := openssl.ParsePKCS12(string(pfxRaw), "")
		Expect(err).To(BeNil())

		certificates.ValidateCertBundle(pem, pubKey, pem, "test.domain.com")
	})

	It("should read ecc cert, private and public key from pod", func() {
		pod.WaitFor(pod.WaitForInput{
			Getter:         kubeClient,
			KubeconfigPath: kubeconfigPath,
			Config:         config,
			PodName:        p.Name,
			Namespace:      ns.Name,
		})

		cmd := getPodExecCommand("cat /mnt/secrets-store/ecccert1")
		cert, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		certificates.ValidateCert(cert, "")

		cmd = getPodExecCommand("cat /mnt/secrets-store/ecccert1-pub-key")
		pubKey, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())

		cmd = getPodExecCommand("cat /mnt/secrets-store/ecccert1-secret")
		out, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		certificates.ValidateCertBundle(out, pubKey, out, "")

		cmd = getPodExecCommand("cat /mnt/secrets-store/ecccert1-secret-pfx")
		out, err = exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
		Expect(err).To(BeNil())
		// base64 decode the pfx data
		pfxRaw, err := base64.StdEncoding.DecodeString(out)
		Expect(err).To(BeNil())
		// Convert pfx data to PEM
		pem, err := openssl.ParsePKCS12(string(pfxRaw), "")
		Expect(err).To(BeNil())

		certificates.ValidateCertBundle(pem, pubKey, pem, "")
	})

	Describe("[Feature:WriteCertAndKeyInSeparateFiles] Writing certificates and private key in separate files", func() {
		It("should write cert and key in separate files", func() {
			pod.WaitFor(pod.WaitForInput{
				Getter:         kubeClient,
				KubeconfigPath: kubeconfigPath,
				Config:         config,
				PodName:        p.Name,
				Namespace:      ns.Name,
			})

			// validate pemcert1
			cmd := getPodExecCommand("cat /mnt/secrets-store/pemcert1-secret.crt")
			cert, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())
			certificates.ValidateCert(cert, "test.domain.com")

			cmd = getPodExecCommand("cat /mnt/secrets-store/pemcert1-pub-key")
			pubKey, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			cmd = getPodExecCommand("cat /mnt/secrets-store/pemcert1-secret.key")
			privKey, err := exec.KubectlExec(kubeconfigPath, p.Name, p.Namespace, strings.Split(cmd, " "))
			Expect(err).To(BeNil())

			certificates.ValidateCertBundle(cert, pubKey, privKey, "test.domain.com")
		})
	})
})
