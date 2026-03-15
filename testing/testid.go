package testing

import (
	"runtime"
	"strings"

	"github.com/get-skipper/skipper-go/core"
)

// testIDFromT derives a Skipper test ID from a testing.T.
//
// It uses runtime.Caller to find the source file of the test function and
// t.Name() to extract the test hierarchy (split on "/").
//
// depth is the number of stack frames to skip relative to this function.
// Callers should pass 1 so that the frame of SkipIfDisabled is skipped and
// the test function's frame is used.
func testIDFromCaller(name string, depth int) string {
	// Walk up the call stack to find the test file (skip internal frames).
	file := callerFile(depth + 1)
	parts := splitTestName(name)
	return core.BuildTestID(file, parts)
}

// callerFile returns the source file path at the given depth above this function.
func callerFile(depth int) string {
	for d := depth; d < depth+20; d++ {
		_, file, _, ok := runtime.Caller(d)
		if !ok {
			break
		}
		// Skip frames from this package and the standard testing package.
		if isInternalFrame(file) {
			continue
		}
		return file
	}
	return "unknown"
}

// splitTestName splits a t.Name() value on "/" to produce the title parts
// used in the test ID. For example:
//
//	"TestCheckout/with_valid_card" → ["TestCheckout", "with valid card"]
func splitTestName(name string) []string {
	parts := strings.Split(name, "/")
	for i, p := range parts {
		// Go replaces spaces with underscores in subtest names; restore them.
		parts[i] = strings.ReplaceAll(p, "_", " ")
	}
	return parts
}

func isInternalFrame(file string) bool {
	return strings.Contains(file, "skipper-go/testing") ||
		strings.Contains(file, "testing/testing.go") ||
		strings.Contains(file, "runtime/")
}
