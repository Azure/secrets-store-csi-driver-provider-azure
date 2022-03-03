package log

import (
	"context"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
)

const (
	serviceBuildFieldName  = "serviceBuild"
	sourceFieldName        = "source"
	contextlessTraceSource = "AcsContextlessTraceLog"
	fileNameFieldName      = "fileName"
	lineNumberFieldName    = "lineNumber"
)

// BuildVersion is the build version for this build, local builds it will be empty
var (
	BuildVersion  string
	DefaultLogger *logrus.Entry
)

func newLogger(formatter logrus.Formatter) *logrus.Entry {
	logger := logrus.New()
	// Changing to stdout, by default logger is set to stderr
	// https://github.com/sirupsen/logrus/blob/5bd5a315a50f7f5663063a3f608757fdca721f7f/logger.go
	logger.Out = os.Stdout
	if os.Getenv("DEBUG_LOG") != "" {
		logger.Level = logrus.DebugLevel
	}
	logger.Formatter = formatter
	return logger.WithField(serviceBuildFieldName, BuildVersion)
}

type loggingKey string

const loggerKey loggingKey = "AcsLogger"

// Logger is a type that can log trace logs or
type Logger struct {
	TraceLogger *logrus.Entry
	qosLogger   *logrus.Entry
}

// GetContextlessTraceLogger returns a logger that can be used outside of a context
func GetContextlessTraceLogger() *Logger {
	return &Logger{
		TraceLogger: DefaultLogger.WithField(sourceFieldName, contextlessTraceSource),
	}
}

func withCallerInfo(logger *logrus.Entry) *logrus.Entry {
	_, file, line, _ := runtime.Caller(2)
	fields := make(map[string]interface{})
	fields[fileNameFieldName] = file
	fields[lineNumberFieldName] = line
	return logger.WithFields(fields)
}

// WithLogger takes a context and logger then returns a child context with the logger attached
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger pulls a logger off the context and returns it
func GetLogger(ctx context.Context) *Logger {
	if retVal, ok := ctx.Value(loggerKey).(*Logger); ok {
		return retVal
	}
	panic("Couldn't get logger")
}

// TraceInfo logs a trace info line containing the message with a field name msg
func (logger *Logger) TraceInfo(msg string) {
	withCallerInfo(logger.TraceLogger).Info(msg)
}

// TraceInfof logs a trace info line containing the formatted string with a field name msg
func (logger *Logger) TraceInfof(fmt string, args ...interface{}) {
	withCallerInfo(logger.TraceLogger).Infof(fmt, args...)
}

// TraceDebug logs a trace debug line containing the message with a field name msg
func (logger *Logger) TraceDebug(msg string) {
	withCallerInfo(logger.TraceLogger).Debug(msg)
}

// TraceDebugf logs a trace debug line containing the formatted string with a field name msg
func (logger *Logger) TraceDebugf(format string, args ...interface{}) {
	withCallerInfo(logger.TraceLogger).Debugf(format, args...)
}

// TraceWarning logs a trace warning line containing the message with a field name msg
func (logger *Logger) TraceWarning(msg string) {
	withCallerInfo(logger.TraceLogger).Warn(msg)
}

// TraceWarningf logs a trace warning line containing the formatted string with a field name msg
func (logger *Logger) TraceWarningf(fmt string, args ...interface{}) {
	withCallerInfo(logger.TraceLogger).Warnf(fmt, args...)
}

// TraceError logs a trace error line containing the message with a field name msg
func (logger *Logger) TraceError(msg string) {
	withCallerInfo(logger.TraceLogger).Error(msg)
}

// TraceErrorf logs a trace error line containing the formatted string with a field name msg
func (logger *Logger) TraceErrorf(fmt string, args ...interface{}) {
	withCallerInfo(logger.TraceLogger).Errorf(fmt, args...)
}

// TraceFatal logs a trace fatal line containing the message with a field name msg and kills process
func (logger *Logger) TraceFatal(msg string) {
	withCallerInfo(logger.TraceLogger).Fatal(msg)
}

// TraceFatalf logs a trace fatal line containing the formated string with a field name msg and kills process
func (logger *Logger) TraceFatalf(fmt string, args ...interface{}) {
	withCallerInfo(logger.TraceLogger).Fatalf(fmt, args...)
}
