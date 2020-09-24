package utils

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/klog"

	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
)

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func LogGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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
