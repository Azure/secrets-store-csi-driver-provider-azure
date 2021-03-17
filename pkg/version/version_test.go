package version

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	BuildDate = "Now"
	BuildVersion = "version"

	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := PrintVersion()

	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- strings.TrimSpace(buf.String())
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := `{"version":"version","buildDate":"Now"}`
	if !strings.EqualFold(out, expected) {
		t.Fatalf("string doesn't match, expected %s, got %s", expected, out)
	}
}

func TestGetUserAgent(t *testing.T) {
	BuildDate = "now"
	Vcs = "commit"
	BuildVersion = "version"

	tests := []struct {
		name              string
		customUserAgent   string
		expectedUserAgent string
	}{
		{
			name:              "default user agent",
			customUserAgent:   "",
			expectedUserAgent: fmt.Sprintf("csi-secrets-store/version (%s/%s) commit/now", runtime.GOOS, runtime.GOARCH),
		},
		{
			name:              "default user agent and custom user agent",
			customUserAgent:   "managedBy:aks",
			expectedUserAgent: fmt.Sprintf("csi-secrets-store/version (%s/%s) commit/now managedBy:aks", runtime.GOOS, runtime.GOARCH),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			customUserAgent = &test.customUserAgent
			actualUserAgent := GetUserAgent()
			if !strings.EqualFold(test.expectedUserAgent, actualUserAgent) {
				t.Fatalf("expected user agent: %s, got: %s.", test.expectedUserAgent, actualUserAgent)
			}
		})
	}
}
