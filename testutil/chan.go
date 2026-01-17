package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func AssertReceiveChan[T any](t *testing.T, ch <-chan T, timeout time.Duration) T {
	select {
	case got := <-ch:
		return got
	case <-time.After(timeout):
		require.Fail(t, "channel didnt receive within timeout")
	}
	return *new(T)
}

func AssertReceiveChanContext[T any](t *testing.T, ctx context.Context, ch <-chan T) T {
	select {
	case got := <-ch:
		return got
	case <-ctx.Done():
		require.Fail(t, "channel didnt receive within context timeout")
	}
	return *new(T)
}
