module github.com/Azure/secrets-store-csi-driver-provider-azure/test/e2e

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v52.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.1
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/secrets-store-csi-driver-provider-azure v0.0.0-00010101000000-000000000000
	github.com/ghodss/yaml v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.2
	sigs.k8s.io/secrets-store-csi-driver v0.0.21
)

replace (
	github.com/Azure/secrets-store-csi-driver-provider-azure => ../..
	// fixes CVE-2020-29652
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
)
