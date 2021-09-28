//go:build e2e
// +build e2e

package framework

import (
	"time"
)

const (
	// Timeout represents the duration it waits until a long-running operation times out.
	// 10m is for windows nodes as pods can take time to come up
	Timeout = 10 * time.Minute

	// Polling represents the polling interval for a long-running operation.
	Polling = 5 * time.Second
)
