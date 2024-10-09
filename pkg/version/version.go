package version

import (
	"encoding/json"
	"flag"
	"fmt"
	"runtime"

	"k8s.io/klog/v2"
)

var (
	// BuildDate is date when binary was built
	BuildDate string
	// BuildVersion is the version of binary
	BuildVersion string
	// Vcs is is the commit hash for the binary build
	Vcs string

	// custom user agent to append for adal and keyvault calls
	customUserAgent = flag.String("custom-user-agent", "", "user agent to append in addition to akv provider versions")
)

// providerVersion holds current provider version
type providerVersion struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
}

func PrintVersion() (err error) {
	pv := providerVersion{
		Version:   BuildVersion,
		BuildDate: BuildDate,
	}

	var res []byte
	if res, err = json.Marshal(pv); err != nil {
		return
	}

	fmt.Printf("%s\n", res)
	return
}

// GetUserAgent returns UserAgent string to append to the agent identifier.
func GetUserAgent() string {
	if *customUserAgent != "" {
		klog.V(5).InfoS("Appending custom user agent", "userAgent", *customUserAgent)
		return fmt.Sprintf("csi-secrets-store/%s (%s/%s) %s/%s %s", BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildDate, *customUserAgent)
	}
	return fmt.Sprintf("csi-secrets-store/%s (%s/%s) %s/%s", BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildDate)
}
