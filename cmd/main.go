package main

import (
	"os"

	"k8s.io/klog"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	"github.com/spf13/pflag"
)

var (
	versionInfo = pflag.Bool("version", false, "prints the version information")
	endpoint    = pflag.String("endpoint", "unix://tmp/azure.sock", "CSI gRPC endpoint")
)

func main() {
	klog.InitFlags(nil)
	pflag.Parse()

	if *versionInfo {
		if err := version.PrintVersion(); err != nil {
			klog.Fatalf("failed to print version, err: %+v", err)
		}
		os.Exit(0)
	}
	// Add csi-secrets-store user agent to adal requests
	if err := adal.AddToUserAgent(version.GetUserAgent()); err != nil {
		klog.Fatalf("failed to add user agent to adal: %+v", err)
	}
	// Initialize and run the GRPC server
	server, err := provider.New(*endpoint)
	if err != nil {
		klog.Fatalf("failed to create new server, err: %+v", err)
	}

	server.Run()
}
