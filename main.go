package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	goflag "flag"

	"github.com/spf13/pflag"
	"k8s.io/klog"
)

var (
	attributes = pflag.String("attributes", "", "volume attributes")
	secrets    = pflag.String("secrets", "", "node publish ref secret")
	targetPath = pflag.String("targetPath", "", "Target path to write data.")
	permission = pflag.String("permission", "", "File permission")
)

func init() {
	os.Setenv("PROVIDER_LOG_FILE", "/var/log/azure-provider.log")
}

func main() {
	klog.InitFlags(nil)

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()

	var attrib, secret map[string]string
	var filePermission os.FileMode
	var f *os.File
	var err error

	// setup log file into which provider logs will be written to
	if f, err = setupLogFile(); err != nil {
		klog.Fatalf("[error]: %v", err)
	}
	defer f.Close()
	os.Stdout = f
	os.Stderr = f

	err = json.Unmarshal([]byte(*attributes), &attrib)
	if err != nil {
		klog.Fatalf("failed to unmarshal attributes, err: %v", err)
	}
	err = json.Unmarshal([]byte(*secrets), &secret)
	if err != nil {
		klog.Fatalf("failed to unmarshal secrets, err: %v", err)
	}
	err = json.Unmarshal([]byte(*permission), &filePermission)
	if err != nil {
		klog.Fatalf("failed to unmarshal file permission, err: %v", err)
	}

	provider, err := NewProvider()
	if err != nil {
		klog.Fatalf("[error] : %v", err)
	}

	ctx := context.Background()
	err = provider.MountSecretsStoreObjectContent(ctx, attrib, secret, *targetPath, filePermission)
	if err != nil {
		klog.Fatalf("[error] : %v", err)
	}

	klog.Flush()
	os.Exit(0)
}

func setupLogFile() (*os.File, error) {
	fileName := os.Getenv("PROVIDER_LOG_FILE")
	if fileName == "" {
		return nil, fmt.Errorf("env var PROVIDER_LOG_FILE not set")
	}
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s failed with error %+v", fileName, err)
	}
	return f, nil
}
