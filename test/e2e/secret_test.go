// +build e2e

package e2e

import (
	"strings"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/utils"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("When deploying SecretProviderClass CRD with secrets", func() {
	BeforeEach(func() {
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
})
