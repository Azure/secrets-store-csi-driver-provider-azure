package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var (
	attributes = pflag.String("attributes", "", "volume attributes")
	secrets    = pflag.String("secrets", "", "node publish ref secret")
	targetPath = pflag.String("targetPath", "", "Target path to write data.")
	permission = pflag.String("permission", "", "File permission")
)

func main() {
	pflag.Parse()

	ctx := context.Background()

	var attrib map[string]string
	var secret map[string]string
	var filePermission os.FileMode

	err := json.Unmarshal([]byte(*attributes), &attrib)
	if err != nil {
		log(fmt.Sprintf("failed to unmarshal attributes, err: %v\n", err))
		os.Exit(1)
	}
	err = json.Unmarshal([]byte(*secrets), &secret)
	if err != nil {
		log(fmt.Sprintf("failed to unmarshal secrets, err: %v\n", err))
		os.Exit(1)
	}
	err = json.Unmarshal([]byte(*permission), &filePermission)
	if err != nil {
		log(fmt.Sprintf("failed to unmarshal file permission, err: %v\n", err))
		os.Exit(1)
	}

	provider, err := NewProvider()
	if err != nil {
		log(fmt.Sprintf("error creating new provider : %v\n", err))
		os.Exit(1)
	}
	err = provider.MountSecretsStoreObjectContent(ctx, attrib, secret, *targetPath, filePermission)
	if err != nil {
		log(fmt.Sprintf("error mounting secret store object content : %v\n", err))
		os.Exit(1)
	}
	os.Exit(0)
}

func log(msg string) {
	fmt.Fprintf(os.Stdout, msg)
}
