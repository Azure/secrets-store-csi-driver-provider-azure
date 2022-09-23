package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"k8s.io/klog/v2"
)

const (
	readHeaderTimeout = 5 * time.Second
)

type HealthZ struct {
	HealthCheckURL *url.URL
	UnixSocketPath string
	RPCTimeout     time.Duration
}

// Serve creates the http handler for serving health requests
func (h *HealthZ) Serve() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc(h.HealthCheckURL.EscapedPath(), h.ServeHTTP)
	server := &http.Server{
		Addr:              h.HealthCheckURL.Host,
		ReadHeaderTimeout: readHeaderTimeout,
		Handler:           serveMux,
	}
	if err := server.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
		klog.ErrorS(err, "failed to start health check server")
		os.Exit(1)
	}
}

func (h *HealthZ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Infof("Started health check")
	ctx, cancel := context.WithTimeout(context.Background(), h.RPCTimeout)
	defer cancel()

	conn, err := h.dialUnixSocket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer conn.Close()

	// create the health check grpc client
	client := grpc_health_v1.NewHealthClient(conn)
	// check health check response against gRPC endpoint.
	err = h.checkRPC(ctx, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
	klog.V(5).Infof("Completed health check")
}

// checkRPC initiates a grpc request to validate the socket is responding
// sends a gRPC HealthCheckRequest and checks if the HealthCheckResponse is valid.
func (h *HealthZ) checkRPC(ctx context.Context, client grpc_health_v1.HealthClient) error {
	v, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return err
	}
	if v == nil || v.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("expected health check response serving")
	}
	return nil
}

func (h *HealthZ) dialUnixSocket() (*grpc.ClientConn, error) {
	return grpc.Dial(
		h.UnixSocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", target)
		}),
	)
}
