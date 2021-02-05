// +build e2e

package openssl

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// ParsePKCS12 parses PKCS#12 pfx data and returns pem with private key
// and certificate
func ParsePKCS12(pfxData, password string) (string, error) {
	tmpFile, err := ioutil.TempFile("", "")
	Expect(err).To(BeNil())

	_, err = tmpFile.Write([]byte(pfxData))
	Expect(err).To(BeNil())
	defer os.Remove(tmpFile.Name())

	args := append([]string{
		"pkcs12",
		"-nodes",
		"-passin",
		fmt.Sprintf("pass:%s", password),
		"-in",
		tmpFile.Name(),
	})

	out, err := openssl(args)
	return out, err
}

func openssl(args []string) (string, error) {
	By(fmt.Sprintf("openssl %s", strings.Join(args, " ")))

	cmd := exec.Command("openssl", args...)
	stdoutStderr, err := cmd.CombinedOutput()

	return string(stdoutStderr), err
}
