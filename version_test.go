package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	BuildDate = "Now"
	BuildVersion = "version"

	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printVersion()

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
	expected := fmt.Sprintf(`{"version":"version","buildDate":"Now","minDriverVersion":"%s"}`, minDriverVersion)
	if !strings.EqualFold(out, expected) {
		t.Fatalf("string doesn't match, expected %s, got %s", expected, out)
	}
}
