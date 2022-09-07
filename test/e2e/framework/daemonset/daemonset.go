//go:build e2e
// +build e2e

package daemonset

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
)

// GetInput is the input for Get.
type GetInput struct {
	Getter    framework.Getter
	Name      string
	Namespace string
}

// Get gets a DaemonSet resource.
func Get(input GetInput) *appsv1.DaemonSet {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for DaemonSet.Get")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for DaemonSet.Get")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for DaemonSet.Get")

	By(fmt.Sprintf("Getting DaemonSet \"%s\"", input.Name))

	daemonSet := &appsv1.DaemonSet{}
	Expect(input.Getter.Get(context.TODO(), types.NamespacedName{Name: input.Name, Namespace: input.Namespace}, daemonSet)).Should(Succeed())
	return daemonSet
}

// UpdateInput is the input for Update.
type UpdateInput struct {
	Updater   framework.Updater
	DaemonSet *appsv1.DaemonSet
}

// Update updates a DaemonSet resource.
func Update(input UpdateInput) *appsv1.DaemonSet {
	Expect(input.Updater).NotTo(BeNil(), "input.Updater is required for DaemonSet.Update")
	Expect(input.DaemonSet).NotTo(BeNil(), "input.DaemonSet is required for DaemonSet.Update")

	By(fmt.Sprintf("Updating DaemonSet \"%s/%s\"", input.DaemonSet.Namespace, input.DaemonSet.Name))

	Expect(input.Updater.Update(context.TODO(), input.DaemonSet)).Should(Succeed())
	return input.DaemonSet
}
