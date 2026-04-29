//go:build e2e
// +build e2e

package e2e

import (
	"context"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider/types"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/namespace"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

var _ = Describe("ValidatingAdmissionPolicy for Azure SecretProviderClass", func() {
	var (
		ns *corev1.Namespace
	)

	BeforeEach(func() {
		ns = namespace.CreateWithName(namespace.CreateInput{
			Creator: kubeClient,
			Name:    "vap-test",
		})
	})

	AfterEach(func() {
		Expect(kubeClient.Delete(context.TODO(), ns)).Should(Succeed())
	})

	validSPC := func(name string, params map[string]string) *v1alpha1.SecretProviderClass {
		return &v1alpha1.SecretProviderClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SecretProviderClassSpec{
				Provider:   "azure",
				Parameters: params,
			},
		}
	}

	baseParams := func() map[string]string {
		return map[string]string{
			types.KeyVaultNameParameter: "my-test-keyvault",
			types.TenantIDParameter:     config.TenantID,
			types.ClientIDParameter:     config.AzureClientID,
		}
	}

	It("should accept a valid SecretProviderClass", func() {
		spc := validSPC("valid-spc", baseParams())
		Expect(kubeClient.Create(context.TODO(), spc)).Should(Succeed())
	})

	It("should reject SecretProviderClass with missing keyvaultName", func() {
		params := baseParams()
		delete(params, types.KeyVaultNameParameter)
		spc := validSPC("missing-kvname", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("keyvaultName"))
	})

	It("should reject SecretProviderClass with invalid keyvaultName (too short)", func() {
		params := baseParams()
		params[types.KeyVaultNameParameter] = "ab"
		spc := validSPC("short-kvname", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("keyvaultName"))
	})

	It("should reject SecretProviderClass with invalid keyvaultName (too long)", func() {
		params := baseParams()
		params[types.KeyVaultNameParameter] = "abcdefghijklmnopqrstuvwxy" // 25 chars
		spc := validSPC("long-kvname", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("keyvaultName"))
	})

	It("should reject SecretProviderClass with invalid keyvaultName (special chars)", func() {
		params := baseParams()
		params[types.KeyVaultNameParameter] = "my_vault.name"
		spc := validSPC("invalid-chars-kvname", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("keyvaultName"))
	})

	It("should reject SecretProviderClass with missing tenantId", func() {
		params := baseParams()
		delete(params, types.TenantIDParameter)
		spc := validSPC("missing-tenantid", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("tenantId"))
	})

	It("should reject SecretProviderClass with invalid usePodIdentity value", func() {
		params := baseParams()
		params[types.UsePodIdentityParameter] = "yes"
		spc := validSPC("invalid-podidentity", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("usePodIdentity"))
	})

	It("should reject SecretProviderClass with invalid useVMManagedIdentity value", func() {
		params := baseParams()
		params[types.UseVMManagedIdentityParameter] = "yes"
		spc := validSPC("invalid-vmmi", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("useVMManagedIdentity"))
	})

	It("should reject SecretProviderClass with multiple identity modes enabled", func() {
		params := baseParams()
		params[types.UsePodIdentityParameter] = "true"
		params[types.UseVMManagedIdentityParameter] = "true"
		spc := validSPC("multiple-identity", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("only one identity mode"))
	})

	It("should reject SecretProviderClass with useAzureTokenProxy but no clientID", func() {
		params := baseParams()
		params[types.UseAzureTokenProxyParameter] = "true"
		delete(params, types.ClientIDParameter)
		spc := validSPC("tokenproxy-no-clientid", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("clientID"))
	})

	It("should reject SecretProviderClass with invalid cloudName", func() {
		params := baseParams()
		params[types.CloudNameParameter] = "InvalidCloud"
		spc := validSPC("invalid-cloudname", params)
		err := kubeClient.Create(context.TODO(), spc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cloudName"))
	})

	It("should accept SecretProviderClass with valid cloudName", func() {
		params := baseParams()
		params[types.CloudNameParameter] = "AzurePublicCloud"
		spc := validSPC("valid-cloudname", params)
		Expect(kubeClient.Create(context.TODO(), spc)).Should(Succeed())
	})

	It("should not affect non-azure provider SecretProviderClass", func() {
		spc := &v1alpha1.SecretProviderClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "non-azure-spc",
				Namespace: ns.Name,
			},
			Spec: v1alpha1.SecretProviderClassSpec{
				Provider:   "vault",
				Parameters: map[string]string{},
			},
		}
		Expect(kubeClient.Create(context.TODO(), spc)).Should(Succeed())
	})
})
