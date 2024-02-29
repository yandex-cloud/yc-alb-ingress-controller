//go:build e2e
// +build e2e

package testutil

import (
	"context"
	"testing"
	"time"
)

func Eventually(t *testing.T, check CheckFunc, opts ...EventuallyOption) bool {
	options := &eventuallyOptions{
		timeout:      5 * time.Second,
		pollInterval: 10 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(options)
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
	defer cancel()

	success := make(chan struct{})
	ticker := time.NewTicker(options.pollInterval)
	defer ticker.Stop()
	// check could block, so it should be done in a separate goroutine.
	go func() {
		for {
			if check() {
				close(success)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	select {
	case <-ctx.Done():
		t.Fatalf("Eventually condition not met: "+options.messageFormat, options.formatArgs...)
		return false
	case <-success:
		return true
	}
}

type eventuallyOptions struct {
	timeout      time.Duration
	pollInterval time.Duration

	messageFormat string
	formatArgs    []interface{}
}

func PollTimeout(timeout time.Duration) EventuallyOption {
	return func(opts *eventuallyOptions) {
		opts.timeout = timeout
	}
}

func PollInterval(interval time.Duration) EventuallyOption {
	return func(opts *eventuallyOptions) {
		opts.pollInterval = interval
	}
}

type CheckFunc func() bool

type EventuallyOption func(opts *eventuallyOptions)

func Message(format string, args ...interface{}) EventuallyOption {
	return func(opts *eventuallyOptions) {
		opts.messageFormat = format
		opts.formatArgs = args
	}
}
