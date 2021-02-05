// +build e2e

package framework

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/secrets-store-csi-driver/apis/v1alpha1"
)

const (
	// NamespaceKubeSystem is the name of kube-system namespace.
	NamespaceKubeSystem = "kube-system"
)

// TryAddDefaultSchemes tries to add various schemes.
func TryAddDefaultSchemes(scheme *runtime.Scheme) {
	// Add the core schemes.
	_ = corev1.AddToScheme(scheme)

	// Add the apps schemes.
	_ = appsv1.AddToScheme(scheme)

	// Add the api extensions (CRD) to the scheme.
	_ = apiextensionsv1beta.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)

	// Add rbac to the scheme.
	_ = rbacv1.AddToScheme(scheme)

	scheme.AddKnownTypes(schema.GroupVersion{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version},
		&v1alpha1.SecretProviderClass{},
		&v1alpha1.SecretProviderClassList{},
	)
	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version})
}
