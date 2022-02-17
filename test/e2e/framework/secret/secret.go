//go:build e2e
// +build e2e

package secret

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator   framework.Creator
	Name      string
	Namespace string
	Data      map[string][]byte
	Labels    map[string]string
}

// Create creates a Secret resource.
func Create(input CreateInput) *v1.Secret {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for Secret.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for Secret.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for Secret.Create")
	Expect(input.Data).NotTo(BeEmpty(), "input.Data is required for Secret.Create")

	By(fmt.Sprintf("Creating Secret \"%s\"", input.Name))

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
			Labels:    input.Labels,
		},
		Data: input.Data,
	}

	Expect(input.Creator.Create(context.TODO(), secret)).Should(Succeed())
	return secret
}

// GetInput is the input for Get.
type GetInput struct {
	Getter    framework.Getter
	Name      string
	Namespace string
}

func Get(input GetInput) *v1.Secret {
	Expect(input.Getter).NotTo(BeNil(), "input.Getter is required for Secret.Get")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for Secret.Get")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for Secret.Get")

	By(fmt.Sprintf("Getting Secret \"%s\"", input.Name))

	secret := &v1.Secret{}
	Expect(input.Getter.Get(context.TODO(), types.NamespacedName{Name: input.Name, Namespace: input.Namespace}, secret)).Should(Succeed())
	return secret
}

// DeleteInput is the input for Delete.
type DeleteInput struct {
	Deleter framework.Deleter
	Secret  *v1.Secret
}

// Delete deletes a Secret resource.
func Delete(input DeleteInput) {
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for Secret.Delete")
	Expect(input.Secret).NotTo(BeNil(), "input.Secret is required for Secret.Delete")

	By(fmt.Sprintf("Deleting Secret \"%s\"", input.Secret.Name))
	Expect(input.Deleter.Delete(context.TODO(), input.Secret)).Should(Succeed())
}
