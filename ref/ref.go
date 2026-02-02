package ref

import (
	"reflect"
	"runtime"
	"strings"
)

func Ptr[T any](v T) *T {
	return &v
}

func Deref[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// FuncName returns the full function name in the form <prefix>.(*<type>).<function>
func FuncName(fn any) string {
	if fullName, ok := fn.(string); ok {
		return fullName
	}
	fullName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	// Compiler adds -fm suffix to a function name which has a receiver.
	return strings.TrimSuffix(fullName, "-fm")
}
