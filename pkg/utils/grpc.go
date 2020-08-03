package utils

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"

	"k8s.io/klog"

	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
	k8spb "sigs.k8s.io/secrets-store-csi-driver/provider/v1alpha1"
)

// Defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start(endpoint string, srv k8spb.CSIDriverProviderServer)
	// Waits for the service to stop
	Wait()
	// Stops the service gracefully
	Stop()
	// Stops the service forcefully
	ForceStop()
}

func NewNonBlockingGRPCServer() NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

// NonBlocking server
type nonBlockingGRPCServer struct {
	k8spb.CSIDriverProviderServer
	wg     sync.WaitGroup
	server *grpc.Server
}

func (s *nonBlockingGRPCServer) Start(endpoint string, srv k8spb.CSIDriverProviderServer) {
	s.wg.Add(1)
	go s.serve(endpoint, srv)
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

func (s *nonBlockingGRPCServer) serve(endpoint string, srv k8spb.CSIDriverProviderServer) {
	proto, addr, err := ParseEndpoint(endpoint)
	if err != nil {
		klog.Fatal(err.Error())
	}

	if proto == "unix" {
		if runtime.GOOS != "windows" {
			addr = "/" + addr
		}
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			klog.Fatalf("Failed to remove %s, error: %s", addr, err.Error())
		}
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	k8spb.RegisterCSIDriverProviderServer(server, srv)
	s.server = server

	klog.Infof("Listening for connections on address: %v", listener.Addr())

	err = server.Serve(listener)
	if err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
}

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(2).Infof("GRPC call: %s", info.FullMethod)
	klog.V(2).Infof("GRPC request: %s", protosanitizer.StripSecrets(req).String())
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("GRPC error: %v", err)
	} else {
		klog.V(2).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp).String())
	}
	return resp, err
}
