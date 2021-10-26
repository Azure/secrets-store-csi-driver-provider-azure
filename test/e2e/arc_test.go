//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework/daemonset"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("When extension arguments are manually overridden", func() {
	var (
		daemonSet                    *appsv1.DaemonSet
		newRotationPollIntervalValue = "--rotation-poll-interval=1m"
		secretStoreCSIDriverName     = "secrets-store-csi-driver"
	)

	It("should reconcile them to original values", func() {
		if !config.IsArcTest {
			Skip("test case only runs while testing arc extension")
		}

		daemonSet = daemonset.Get(daemonset.GetInput{
			Namespace: framework.NamespaceKubeSystem,
			Name:      secretStoreCSIDriverName,
			Getter:    kubeClient,
		})
		Expect(daemonSet).NotTo(BeNil())

		daemonSet.Spec.Template.Spec.Containers[1].Args = append(daemonSet.Spec.Template.Spec.Containers[1].Args, newRotationPollIntervalValue)
		daemonSet = daemonset.Update(daemonset.UpdateInput{
			Updater:   kubeClient,
			DaemonSet: daemonSet,
		})
		Expect(daemonSet).NotTo(BeNil())

		// waiting for 300 seconds since 'reconcilerIntervalInSeconds' is set to this value in extension configuration
		By("Waiting for arc extension to reconcile the arguments")
		time.Sleep(time.Second * 300)

		daemonSet = daemonset.Get(daemonset.GetInput{
			Namespace: framework.NamespaceKubeSystem,
			Name:      secretStoreCSIDriverName,
			Getter:    kubeClient,
		})
		Expect(daemonSet).NotTo(BeNil())
		con, _ := json.MarshalIndent(daemonSet, "", "  ")
		fmt.Printf("%s\n", con)

		for _, arg := range daemonSet.Spec.Template.Spec.Containers[1].Args {
			if arg == newRotationPollIntervalValue {
				// Manually overridden value should be reverted by arc extension reconciliation
				Expect(arg).NotTo(Equal(newRotationPollIntervalValue))
			}
		}
	})
})
