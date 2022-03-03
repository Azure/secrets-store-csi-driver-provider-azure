//go:build localbuild
// +build localbuild

package log

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func init() {
	DefaultLogger = newLogger(&filterFormatter{&logrus.TextFormatter{
		ForceColors: true,
	}})
}

type filterFormatter struct {
	wrapped logrus.Formatter
}

func (f *filterFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	line := entry.Data[lineNumberFieldName].(int)
	entry.Data["src"] = strings.Join([]string{filepath.Base(entry.Data[fileNameFieldName].(string)), strconv.Itoa(line)}, ":")
	entry.Message = strings.TrimSpace(entry.Message)

	// Remove everything but the condensed log location
	for k := range entry.Data {
		if k == "src" {
			continue
		}
		delete(entry.Data, k)
	}

	return f.wrapped.Format(entry)
}
