package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"
	k8spb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"

	"google.golang.org/grpc"
)

func TestServe(t *testing.T) {
	tests := []struct {
		desc                   string
		setupServer            func(socketPath string)
		expectedHTTPStatusCode int
	}{
		{
			desc:                   "failed health check",
			setupServer:            func(socketPath string) {},
			expectedHTTPStatusCode: http.StatusServiceUnavailable,
		},
		{
			desc: "successful health check",
			setupServer: func(socketPath string) {
				listener, err := net.Listen("unix", socketPath)
				if err != nil {
					t.Fatalf("expected error to be nil, got: %v", err)
				}
				s := grpc.NewServer()
				k8spb.RegisterCSIDriverProviderServer(s, &CSIDriverProviderServer{})
				grpc_health_v1.RegisterHealthServer(s, &CSIDriverProviderServer{})
				go s.Serve(listener)
			},
			expectedHTTPStatusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			socketPath := fmt.Sprintf("%s/azure.sock", getTempTestDir(t))
			defer os.Remove(socketPath)

			test.setupServer(socketPath)

			healthz := &HealthZ{
				UnixSocketPath: socketPath,
				RPCTimeout:     20 * time.Second,
				HealthCheckURL: &url.URL{
					Scheme: "http",
					Host:   net.JoinHostPort("localhost", "8080"),
					Path:   "/healthz",
				},
			}

			server := httptest.NewServer(healthz)
			defer server.Close()

			respCode, body := doHealthCheck(t, server.URL)
			if respCode != test.expectedHTTPStatusCode {
				t.Fatalf("expected status code: %v, got: %v", test.expectedHTTPStatusCode, respCode)
			}
			if test.expectedHTTPStatusCode == http.StatusOK && string(body) != "ok" {
				t.Fatalf("expected response body to be 'ok', got: %s", string(body))
			}
		})
	}
}

func TestCheckRPC(t *testing.T) {
	socketPath := fmt.Sprintf("%s/azure.sock", getTempTestDir(t))
	defer os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("expected error to be nil, got: %v", err)
	}

	s := grpc.NewServer()
	k8spb.RegisterCSIDriverProviderServer(s, &CSIDriverProviderServer{})
	grpc_health_v1.RegisterHealthServer(s, &CSIDriverProviderServer{})
	go s.Serve(listener)

	healthz := &HealthZ{
		UnixSocketPath: socketPath,
	}

	conn, err := healthz.dialUnixSocket()
	if err != nil {
		t.Fatalf("failed to create connection, err: %+v", err)
	}
	err = healthz.checkRPC(context.TODO(), grpc_health_v1.NewHealthClient(conn))
	if err != nil {
		t.Fatalf("expected err to be nil, got: %+v", err)
	}
}

func getTempTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "ut")
	if err != nil {
		t.Fatalf("expected err to be nil, got: %+v", err)
	}
	return tmpDir
}

func doHealthCheck(t *testing.T, url string) (int, []byte) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("failed to create new http request, err: %+v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to invoke http request, err: %+v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body, err: %+v", err)
	}
	return resp.StatusCode, body
}
