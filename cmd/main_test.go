package main

import (
	"testing"
)

func TestNewJSONLogger(t *testing.T) {
	// newJSONLogger must not panic. Before the fix, passing empty
	// LoggingOptions (nil ErrorStream/InfoStream) caused a nil pointer
	// dereference inside component-base's json writer on the first
	// Write call.
	logger := newJSONLogger()
	if logger.GetSink() == nil {
		t.Fatal("expected logger to have a non-nil sink")
	}

	// Verify the logger can actually write without panicking.
	logger.Info("test message", "key", "value")
}
