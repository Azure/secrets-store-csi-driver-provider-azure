package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/helm"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/utils"
)

var _ = PDescribe("Test backward compatibility", func() {
	Context("By upgrading to current release", func() {
		BeforeEach(func() {
			//Upgrade to Current Version
			if helm.ReleaseExists() {
				By("Upgrading Secrets Store CSI Driver and Azure Key Vault Provider via Helm")
				helm.Upgrade(helm.InstallInput{
					Config: config,
				})
			}

			utils.RecreateSecretPod(kubeClient, kubeconfigPath, config)
		})

		It("should read secret from pod", func() {
			cmd := getPodExecCommand("cat /mnt/secrets-store/secret1")
			secret, err := exec.KubectlExec(kubeconfigPath, utils.SecretPodName, utils.SecretNamespaeName, strings.Split(cmd, " "))
			Expect(err).To(BeNil())
			Expect(secret).To(Equal(config.SecretValue))
		})

		It("should read secret from pod with alias", func() {
			cmd := getPodExecCommand("cat /mnt/secrets-store/SECRET_1")
			secret, err := exec.KubectlExec(kubeconfigPath, utils.SecretPodName, utils.SecretNamespaeName, strings.Split(cmd, " "))
			Expect(err).To(BeNil())
			Expect(secret).To(Equal(config.SecretValue))
		})

		AfterEach(func() {
			//Upgrade to new version
			if helm.ReleaseExists() {
				By("Upgrading Secrets Store CSI Driver and Azure Key Vault Provider via Helm")
				helm.Upgrade(helm.InstallInput{
					Config:           config,
				})
			}
		})
	})
})
