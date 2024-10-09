package utils

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		desc          string
		endpoint      string
		expectedProto string
		expectedAddr  string
		expectedErr   bool
	}{
		{
			desc:        "invalid endpoint",
			endpoint:    "udp:///provider/azure.sock",
			expectedErr: true,
		},
		{
			desc:          "invalid unix endpoint",
			endpoint:      "unix://",
			expectedProto: "",
			expectedAddr:  "",
			expectedErr:   true,
		},
		{
			desc:          "valid tcp endpoint",
			endpoint:      "tcp://:7777",
			expectedProto: "tcp",
			expectedAddr:  ":7777",
			expectedErr:   false,
		},
		{
			desc:          "valid unix endpoint",
			endpoint:      "unix:///provider/azure.sock",
			expectedProto: "unix",
			expectedAddr:  "/provider/azure.sock",
			expectedErr:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			proto, addr, err := ParseEndpoint(tc.endpoint)
			if tc.expectedErr && err == nil || !tc.expectedErr && err != nil {
				t.Fatalf("expected error: %v, got error: %v", tc.expectedErr, err)
			}
			if proto != tc.expectedProto {
				t.Fatalf("expected proto: %v, got: %v", tc.expectedProto, proto)
			}
			if addr != tc.expectedAddr {
				t.Fatalf("expected addr: %v, got: %v", tc.expectedAddr, addr)
			}
		})
	}
}

func TestLogInterceptor(t *testing.T) {
	fs := &flag.FlagSet{}
	klog.InitFlags(fs)
	fs.Parse([]string{"-v", "5"})

	// required to make SetOutput work
	klog.LogToStderr(false)
	b := new(bytes.Buffer)
	klog.SetOutput(b)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{
		Server:     nil,
		FullMethod: "FakeMethod",
	}

	_, got := LogInterceptor()(context.Background(), nil, info, handler)

	if want := codes.OK; status.Code(got) != want {
		t.Errorf("LogInterceptor() error =\n\t%v,\n\twant = %v", got, want)
	}

	klog.Flush()

	if !strings.Contains(b.String(), "request") {
		t.Errorf("LogInterceptor() did not log request\n\tgot:%v", b.String())
	}
	if !strings.Contains(b.String(), "response") {
		t.Errorf("LogInterceptor() did not log response\n\tgot:%v", b.String())
	}
	if !strings.Contains(b.String(), "code=\"OK\"") {
		t.Errorf("LogInterceptor() did not log response code OK, got:\n%v", b.String())
	}
}

func TestLogInterceptor_Error(t *testing.T) {
	fs := &flag.FlagSet{}
	klog.InitFlags(fs)
	fs.Parse([]string{"-v", "5"})

	// required to make SetOutput work
	klog.LogToStderr(false)
	b := new(bytes.Buffer)
	klog.SetOutput(b)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, status.Error(codes.Internal, "bad request")
	}
	info := &grpc.UnaryServerInfo{
		Server:     nil,
		FullMethod: "FakeMethod",
	}

	_, got := LogInterceptor()(context.Background(), nil, info, handler)

	if want := codes.Internal; status.Code(got) != want {
		t.Errorf("LogInterceptor() error =\n\t%v,\n\twant = %v", got, want)
	}

	klog.Flush()

	if !strings.Contains(b.String(), "request") {
		t.Errorf("LogInterceptor() did not log request\n\tgot:%v", b.String())
	}
	if !strings.Contains(b.String(), "response") {
		t.Errorf("LogInterceptor() did not log response\n\tgot:%v", b.String())
	}
	if !strings.Contains(b.String(), "code=\"Internal\"") {
		t.Errorf("LogInterceptor() did not log response code Internal, got:\n%v", b.String())
	}
}
