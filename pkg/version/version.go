package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	// BuildDate is date when binary was built
	BuildDate string
	// BuildVersion is the version of binary
	BuildVersion string
	// Vcs is is the commit hash for the binary build
	Vcs string

	minDriverVersion = "v0.0.8"
)

// providerVersion holds current provider version
type providerVersion struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	// MinDriverVersion is minimum driver version the provider works with
	MinDriverVersion string `json:"minDriverVersion"`
}

func PrintVersion() (err error) {
	pv := providerVersion{
		Version:          BuildVersion,
		BuildDate:        BuildDate,
		MinDriverVersion: minDriverVersion,
	}

	var res []byte
	if res, err = json.Marshal(pv); err != nil {
		return
	}

	fmt.Printf(string(res) + "\n")
	return
}

// GetUserAgent returns UserAgent string to append to the agent identifier.
func GetUserAgent() string {
	return fmt.Sprintf("csi-secrets-store/%s (%s/%s) %s/%s", BuildVersion, runtime.GOOS, runtime.GOARCH, Vcs, BuildDate)
}
