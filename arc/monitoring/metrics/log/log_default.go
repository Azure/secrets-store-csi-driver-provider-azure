//go:build !localbuild
// +build !localbuild

package log

import "github.com/sirupsen/logrus"

func init() {
	DefaultLogger = newLogger(&logrus.JSONFormatter{})
}
