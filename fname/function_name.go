package fname

import (
	"reflect"
	"runtime"
	"strings"
)

// FuncName returns the full function name in the form <prefix>.(*<type>).<function>
func FuncName(fn any) string {
	if fullName, ok := fn.(string); ok {
		return fullName
	}
	fullName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	// Compiler adds -fm suffix to a function name which has a receiver.
	return strings.TrimSuffix(fullName, "-fm")
}

// CurrentFuncName returns the name of the function that calls CurrentFuncName.
func CurrentFuncName() string {
	return CallerFuncName(1)
}

// CallerFuncName returns the function name of the caller at the given stack depth.
//
// skip meanings:
//
//	0 = CallerFuncName
//	1 = function calling CallerFuncName
//	2 = parent of that function
//	etc...
func CallerFuncName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}

	if fn := runtime.FuncForPC(pc); fn != nil {
		return strings.TrimSuffix(fn.Name(), "-fm")
	}

	return "unknown"
}

// ShortFuncName returns just the function/method name without package or type path.
func ShortFuncName(full string) string {
	// Strip path
	if i := strings.LastIndex(full, "/"); i >= 0 {
		full = full[i+1:]
	}
	// Strip package/type
	if i := strings.LastIndex(full, "."); i >= 0 {
		return full[i+1:]
	}
	return full
}

// CurrentFuncShortName returns the short name of the calling function.
func CurrentFuncShortName() string {
	return ShortFuncName(CurrentFuncName())
}

// CallerFuncShortName returns the short name of the caller at the given stack depth.
func CallerFuncShortName(skip int) string {
	return ShortFuncName(CallerFuncName(skip))
}
