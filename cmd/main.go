package main

import (
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/server"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/utils"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	k8spb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"

	"google.golang.org/grpc"

	"k8s.io/klog"

	"github.com/Azure/go-autorest/autorest/adal"

	"github.com/spf13/pflag"
)

var (
	versionInfo = pflag.Bool("version", false, "prints the version information")
	endpoint    = pflag.String("endpoint", "unix:///tmp/azure.sock", "CSI gRPC endpoint")
)

func main() {
	klog.InitFlags(nil)
	pflag.Parse()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

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
	proto, addr, err := utils.ParseEndpoint(*endpoint)
	if err != nil {
		klog.Fatalf("failed to parse endpoint, err: %+v", err)
	}

	if proto == "unix" {
		if runtime.GOOS != "windows" {
			addr = "/" + addr
		}
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.Fatalf("failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.LogGRPC),
	}
	s := grpc.NewServer(opts...)
	k8spb.RegisterCSIDriverProviderServer(s, &server.CSIDriverProviderServer{})

	klog.Infof("Listening for connections on address: %v", listener.Addr())
	go s.Serve(listener)

	<-signalChan
	// gracefully stop the grpc server
	klog.Infof("terminating the server")
	s.GracefulStop()
}
