package serviceaccount

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator   framework.Creator
	Name      string
	Namespace string
}

// Create creates a ServiceAccount resource.
func Create(input CreateInput) *corev1.ServiceAccount {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for ServiceAccount.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for ServiceAccount.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for ServiceAccount.Create")

	By(fmt.Sprintf("Creating ServiceAccount \"%s/%s\"", input.Namespace, input.Name))
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
	}

	Expect(input.Creator.Create(context.TODO(), sa)).Should(Succeed())
	return sa
}
