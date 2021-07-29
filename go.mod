module github.com/Azure/secrets-store-csi-driver-provider-azure

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v52.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.1
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/google/go-cmp v0.5.4
	github.com/kubernetes-csi/csi-lib-utils v0.7.1
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	google.golang.org/grpc v1.31.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/component-base v0.20.2
	k8s.io/klog/v2 v2.8.0
	sigs.k8s.io/secrets-store-csi-driver v0.0.21
)
