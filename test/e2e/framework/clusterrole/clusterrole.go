//go:build e2e
// +build e2e

package clusterrole

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator framework.Creator
	Name    string
	Rules   []rbacv1.PolicyRule
}

// Create creates a ClusterRole resource.
func Create(input CreateInput) *rbacv1.ClusterRole {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for ClusterRole.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for ClusterRole.Create")

	By(fmt.Sprintf("Creating ClusterRole \"%s\"", input.Name))
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Name,
		},
		Rules: input.Rules,
	}

	Expect(input.Creator.Create(context.TODO(), cr)).Should(Succeed())
	return cr
}

// DeleteInput is the input for Delete.
type DeleteInput struct {
	Deleter     framework.Deleter
	ClusterRole *rbacv1.ClusterRole
}

// Delete deletes a ClusterRole resource.
func Delete(input DeleteInput) {
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for ClusterRole.Delete")
	Expect(input.ClusterRole).NotTo(BeNil(), "input.ClusterRole is required for ClusterRole.Delete")

	By(fmt.Sprintf("Deleting ClusterRole \"%s\"", input.ClusterRole.Name))
	Expect(input.Deleter.Delete(context.TODO(), input.ClusterRole)).Should(Succeed())
}
