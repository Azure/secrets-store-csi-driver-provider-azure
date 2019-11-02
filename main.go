package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	attributes = pflag.String("attributes", "", "volume attributes")
	secrets    = pflag.String("secrets", "", "node publish ref secret")
	targetPath = pflag.String("targetPath", "", "Target path to write data.")
	permission = pflag.String("permission", "", "File permission")
	debug      = pflag.Bool("debug", false, "sets log to debug level")
)

// LogHook is used to setup custom hooks
type LogHook struct {
	Writer    io.Writer
	Loglevels []log.Level
}

func main() {
	pflag.Parse()

	var attrib, secret map[string]string
	var filePermission os.FileMode
	var err error

	setupLogger()

	err = json.Unmarshal([]byte(*attributes), &attrib)
	if err != nil {
		log.Fatalf("failed to unmarshal attributes, err: %v", err)
	}
	err = json.Unmarshal([]byte(*secrets), &secret)
	if err != nil {
		log.Fatalf("failed to unmarshal secrets, err: %v", err)
	}
	err = json.Unmarshal([]byte(*permission), &filePermission)
	if err != nil {
		log.Fatalf("failed to unmarshal file permission, err: %v", err)
	}

	provider, err := NewProvider()
	if err != nil {
		log.Fatalf("[error] : %v", err)
	}

	ctx := context.Background()
	err = provider.MountSecretsStoreObjectContent(ctx, attrib, secret, *targetPath, filePermission)
	if err != nil {
		log.Fatalf("[error] : %v", err)
	}

	os.Exit(0)
}

// setupLogger sets up hooks to redirect stdout and stderr
func setupLogger() {
	log.SetOutput(ioutil.Discard)

	// set log level
	log.SetLevel(log.InfoLevel)
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	// set log formatter to json
	//log.SetFormatter(&log.JSONFormatter{})

	// add hook to send info, debug, warn level logs to stdout
	log.AddHook(&LogHook{
		Writer: os.Stdout,
		Loglevels: []log.Level{
			log.InfoLevel,
			log.DebugLevel,
			log.WarnLevel,
		},
	})

	// add hook to send panic, fatal, error logs to stderr
	log.AddHook(&LogHook{
		Writer: os.Stderr,
		Loglevels: []log.Level{
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
		},
	})
}

// Fire is called when logging function with current hook is called
// write to appropriate writer based on log level
func (hook *LogHook) Fire(entry *log.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write([]byte(line))
	return err
}

// Levels defines log levels at which hook is triggered
func (hook *LogHook) Levels() []log.Level {
	return hook.Loglevels
}
