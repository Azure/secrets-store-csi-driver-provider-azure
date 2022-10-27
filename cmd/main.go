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
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/server"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/utils"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/pkg/version"

	"github.com/Azure/go-autorest/autorest/adal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	logsapi "k8s.io/component-base/logs/api/v1"
	json "k8s.io/component-base/logs/json"
	"k8s.io/klog/v2"
	k8spb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

const (
	readHeaderTimeout = 5 * time.Second
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

	metricsBackend = flag.String("metrics-backend", "Prometheus", "Backend used for metrics")
	prometheusPort = flag.Int("prometheus-port", 8898, "Prometheus port for metrics backend")

	constructPEMChain              = flag.Bool("construct-pem-chain", true, "explicitly reconstruct the pem chain in the order: SERVER, INTERMEDIATE, ROOT")
	writeCertAndKeyInSeparateFiles = flag.Bool("write-cert-and-key-in-separate-files", false,
		"Write cert and key in separate files. The individual files will be named as <secret-name>.crt and <secret-name>.key. These files will be created in addition to the single file.")
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.Parse()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	if *logFormatJSON {
		jsonFactory := json.Factory{}
		logger, _ := jsonFactory.Create(logsapi.LoggingConfiguration{Format: "json"})
		klog.SetLogger(logger)
	}

	if *versionInfo {
		if err := version.PrintVersion(); err != nil {
			klog.ErrorS(err, "failed to print version")
			os.Exit(1)
		}
		os.Exit(0)
	}
	klog.InfoS("Starting Azure Key Vault Provider", "version", version.BuildVersion)

	if *enableProfile {
		klog.InfoS("Starting profiling", "port", *profilePort)
		go func() {
			server := &http.Server{
				Addr:              fmt.Sprintf("%s:%d", "localhost", *profilePort),
				ReadHeaderTimeout: readHeaderTimeout,
			}
			klog.ErrorS(server.ListenAndServe(), "unable to start profiling server")
		}()
	}
	// initialize metrics exporter before creating measurements
	err := metrics.InitMetricsExporter(*metricsBackend, *prometheusPort)
	if err != nil {
		klog.ErrorS(err, "failed to initialize metrics exporter")
		os.Exit(1)
	}

	if *constructPEMChain {
		klog.Infof("construct pem chain feature enabled")
	}
	if *writeCertAndKeyInSeparateFiles {
		klog.Infof("write cert and key in separate files feature enabled")
	}
	// Add csi-secrets-store user agent to adal requests
	if err := adal.AddToUserAgent(version.GetUserAgent()); err != nil {
		klog.ErrorS(err, "failed to add user agent to adal")
		os.Exit(1)
	}
	// Initialize and run the gRPC server
	proto, addr, err := utils.ParseEndpoint(*endpoint)
	if err != nil {
		klog.ErrorS(err, "failed to parse endpoint")
		os.Exit(1)
	}

	if proto == "unix" {
		if runtime.GOOS != "windows" {
			addr = "/" + addr
		}
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.ErrorS(err, "failed to remove socket", "addr", addr)
			os.Exit(1)
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.ErrorS(err, "failed to listen", "proto", proto, "addr", addr)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.LogInterceptor()),
	}
	s := grpc.NewServer(opts...)
	csiDriverProviderServer := server.New(*constructPEMChain, *writeCertAndKeyInSeparateFiles)
	k8spb.RegisterCSIDriverProviderServer(s, csiDriverProviderServer)
	// Register the health service.
	grpc_health_v1.RegisterHealthServer(s, csiDriverProviderServer)

	klog.InfoS("Listening for connections", "address", listener.Addr())
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
