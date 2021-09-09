package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // #nosec
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/metrics"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/provider"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/server"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/utils"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	"github.com/Azure/go-autorest/autorest/adal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	json "k8s.io/component-base/logs/json"
	"k8s.io/klog/v2"
	k8spb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

var (
	versionInfo   = flag.Bool("version", false, "prints the version information")
	endpoint      = flag.String("endpoint", "unix:///tmp/azure.sock", "CSI gRPC endpoint")
	logFormatJSON = flag.Bool("log-format-json", false, "set log formatter to json")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling")

	healthzPort    = flag.Int("healthz-port", 8989, "port for health check")
	healthzPath    = flag.String("healthz-path", "/healthz", "path for health check")
	healthzTimeout = flag.Duration("healthz-timeout", 5*time.Second, "RPC timeout for health check")

	// driverWriteSecrets feature is enabled by default in v0.1.0 release. All writes to the pod filesystem will now be done by the CSI driver instead of provider.
	// this flag will be removed in the future.
	driverWriteSecrets = flag.Bool("driver-write-secrets", true, "[DEPRECATED] Return secrets in gRPC response to the driver (supported in driver v0.0.21+) instead of writing to filesystem")

	metricsBackend = flag.String("metrics-backend", "Prometheus", "Backend used for metrics")
	prometheusPort = flag.Int("prometheus-port", 8898, "Prometheus port for metrics backend")
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.Parse()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	if *logFormatJSON {
		klog.SetLogger(json.JSONLogger)
	}

	if *versionInfo {
		if err := version.PrintVersion(); err != nil {
			klog.Fatalf("failed to print version, err: %+v", err)
		}
		os.Exit(0)
	}
	klog.Infof("Starting Azure Key Vault Provider version: %s", version.BuildVersion)

	if *enableProfile {
		klog.Infof("Starting profiling on port %d", *profilePort)
		go func() {
			addr := fmt.Sprintf("%s:%d", "localhost", *profilePort)
			klog.ErrorS(http.ListenAndServe(addr, nil), "unable to start profiling server")
		}()
	}
	// initialize metrics exporter before creating measurements
	err := metrics.InitMetricsExporter(*metricsBackend, *prometheusPort)
	if err != nil {
		klog.Fatalf("failed to initialize metrics exporter, error: %+v", err)
	}

	if *provider.ConstructPEMChain {
		klog.Infof("construct pem chain feature enabled")
	}
	if !*driverWriteSecrets {
		klog.Infof("driver write secrets feature can't be disabled. The --driver-write-secret flag will be removed in future releases.")
	}
	// Add csi-secrets-store user agent to adal requests
	if err := adal.AddToUserAgent(version.GetUserAgent()); err != nil {
		klog.Fatalf("failed to add user agent to adal: %+v", err)
	}
	// Initialize and run the gRPC server
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
		grpc.UnaryInterceptor(utils.LogInterceptor()),
	}
	s := grpc.NewServer(opts...)
	csiDriverProviderServer := server.New()
	k8spb.RegisterCSIDriverProviderServer(s, csiDriverProviderServer)
	// Register the health service.
	grpc_health_v1.RegisterHealthServer(s, csiDriverProviderServer)

	klog.Infof("Listening for connections on address: %v", listener.Addr())
	go s.Serve(listener)

	healthz := &server.HealthZ{
		HealthCheckURL: &url.URL{
			Host: net.JoinHostPort("", strconv.FormatUint(uint64(*healthzPort), 10)),
			Path: *healthzPath,
		},
		UnixSocketPath: listener.Addr().String(),
		RPCTimeout:     *healthzTimeout,
	}
	go healthz.Serve()

	<-signalChan
	// gracefully stop the grpc server
	klog.Infof("terminating the server")
	s.GracefulStop()
}
