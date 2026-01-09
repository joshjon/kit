package testutil

import (
	"context"
	"testing"
	"time"
)

const defaultTimeout = 5 * time.Second

type ContextOption func(opts *contextOptions)

func WithTimeout(timeout time.Duration) ContextOption {
	return func(opts *contextOptions) {
		opts.timeout = timeout
	}
}

type contextOptions struct {
	timeout time.Duration
}

func Context(t *testing.T, opts ...ContextOption) context.Context {
	t.Helper()

	options := contextOptions{
		timeout: defaultTimeout,
	}

	for _, opt := range opts {
		opt(&options)
	}

	if options.timeout <= 0 {
		t.Fatalf("testutil.Context: timeout must be > 0")
	}

	ctx, cancel := context.WithTimeout(t.Context(), options.timeout)
	t.Cleanup(cancel)
	return ctx
}
