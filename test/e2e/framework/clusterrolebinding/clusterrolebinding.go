//go:build e2e
// +build e2e

package clusterrolebinding

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
	Creator  framework.Creator
	Name     string
	RoleRef  rbacv1.RoleRef
	Subjects []rbacv1.Subject
}

// Create creates a ClusterRoleBinding resource.
func Create(input CreateInput) *rbacv1.ClusterRoleBinding {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for ClusterRoleBinding.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for ClusterRoleBinding.Create")

	By(fmt.Sprintf("Creating ClusterRoleBinding \"%s\"", input.Name))
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Name,
		},
		RoleRef:  input.RoleRef,
		Subjects: input.Subjects,
	}

	Expect(input.Creator.Create(context.TODO(), crb)).Should(Succeed())
	return crb
}

// DeleteInput is the input for Delete.
type DeleteInput struct {
	Deleter            framework.Deleter
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
}

// Delete deletes a ClusterRoleBinding resource.
func Delete(input DeleteInput) {
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for ClusterRoleBinding.Delete")
	Expect(input.ClusterRoleBinding).NotTo(BeNil(), "input.ClusterRoleBinding is required for ClusterRoleBinding.Delete")

	By(fmt.Sprintf("Deleting ClusterRoleBinding \"%s\"", input.ClusterRoleBinding.Name))
	Expect(input.Deleter.Delete(context.TODO(), input.ClusterRoleBinding)).Should(Succeed())
}
