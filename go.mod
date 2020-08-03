module github.com/Azure/secrets-store-csi-driver-provider-azure

go 1.12

require (
	github.com/Azure/azure-sdk-for-go v34.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.9.1
	github.com/Azure/go-autorest/autorest/adal v0.6.0
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/kubernetes-csi/csi-lib-utils v0.7.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20200222125558-5a598a2470a0
	google.golang.org/grpc v1.31.0
	gopkg.in/yaml.v2 v2.2.7
	k8s.io/klog v1.0.0
	sigs.k8s.io/secrets-store-csi-driver v0.0.14
)
