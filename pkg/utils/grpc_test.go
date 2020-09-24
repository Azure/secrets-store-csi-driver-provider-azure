package utils

import (
	"testing"
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
