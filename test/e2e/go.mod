module github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e

go 1.14

require (
	github.com/Azure/azure-sdk-for-go v34.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.6
	github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Azure/secrets-store-csi-driver-provider-azure v0.0.0-00010101000000-000000000000
	github.com/ghodss/yaml v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/secrets-store-csi-driver v0.0.18
)

replace github.com/Azure/secrets-store-csi-driver-provider-azure => ../..
