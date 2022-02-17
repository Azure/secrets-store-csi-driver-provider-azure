//go:build e2e
// +build e2e

package secretproviderclass

import (
	"context"
	"fmt"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

// CreateInput is the input for Create.
type CreateInput struct {
	Creator   framework.Creator
	Config    *framework.Config
	Name      string
	Namespace string
	Spec      v1alpha1.SecretProviderClassSpec
}

// Create creates a SecretProviderClass resource.
func Create(input CreateInput) *v1alpha1.SecretProviderClass {
	Expect(input.Creator).NotTo(BeNil(), "input.Creator is required for SecretProviderClass.Create")
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for SecretProviderClass.Create")
	Expect(input.Name).NotTo(BeEmpty(), "input.Name is required for SecretProviderClass.Create")
	Expect(input.Namespace).NotTo(BeEmpty(), "input.Namespace is required for SecretProviderClass.Create")
	Expect(input.Spec).NotTo(BeNil(), "input.Spec is required for SecretProviderClass.Create")

	By(fmt.Sprintf("Creating SecretProviderClass \"%s/%s\"", input.Namespace, input.Name))

	spc := &v1alpha1.SecretProviderClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: input.Spec,
	}

	Expect(input.Creator.Create(context.TODO(), spc)).Should(Succeed())
	return spc
}

// DeleteInput is the input for Delete.
type DeleteInput struct {
	Deleter             framework.Deleter
	SecretProviderClass *v1alpha1.SecretProviderClass
}

// Delete deletes a SecretProviderClass resource.
func Delete(input DeleteInput) {
	Expect(input.Deleter).NotTo(BeNil(), "input.Deleter is required for SecretProviderClass.Delete")
	Expect(input.SecretProviderClass).NotTo(BeNil(), "input.SecretProviderClass is required for SecretProviderClass.Delete")

	By(fmt.Sprintf("Deleting SecretProviderClass \"%s\"", input.SecretProviderClass.Name))
	Expect(input.Deleter.Delete(context.TODO(), input.SecretProviderClass)).Should(Succeed())
}

// UpdateInput is the input for Update.
type UpdateInput struct {
	Updater             framework.Updater
	SecretProviderClass *v1alpha1.SecretProviderClass
}

// Update updates a SecretProviderClass resource.
func Update(input UpdateInput) *v1alpha1.SecretProviderClass {
	Expect(input.Updater).NotTo(BeNil(), "input.Updater is required for SecretProviderClass.Update")
	Expect(input.SecretProviderClass).NotTo(BeNil(), "input.SecretProviderClass is required for SecretProviderClass.Update")

	By(fmt.Sprintf("Updating SecretProviderClass \"%s/%s\"", input.SecretProviderClass.Namespace, input.SecretProviderClass.Name))

	Expect(input.Updater.Update(context.TODO(), input.SecretProviderClass)).Should(Succeed())
	return input.SecretProviderClass
}
