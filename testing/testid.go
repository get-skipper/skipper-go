package testing

import (
	"runtime"
	"strings"

	"github.com/get-skipper/skipper-go/core"
)

// testIDFromT derives a Skipper test ID from a testing.T.
//
// It uses runtime.Caller to find the first _test.go file in the call stack
// and t.Name() to extract the test hierarchy (split on "/").
func testIDFromCaller(name string, _ int) string {
	file := callerFile()
	parts := splitTestName(name)
	return core.BuildTestID(file, parts)
}

// callerFile returns the path of the first _test.go file found in the call
// stack. This is always the user's test file because adapter implementation
// files (skipper.go, testid.go) are never _test.go files.
func callerFile() string {
	for depth := 1; depth < 30; depth++ {
		_, file, _, ok := runtime.Caller(depth)
		if !ok {
			break
		}
		if strings.HasSuffix(file, "_test.go") {
			return file
		}
	}
	return "unknown"
}

// splitTestName splits a t.Name() value on "/" to produce the title parts
// used in the test ID. For example:
//
//	"TestCheckout/with_valid_card" → ["TestCheckout", "with valid card"]
//
// Underscores are only restored for subtest names (index > 0) because Go
// replaces spaces with underscores in t.Run names. The top-level test
// function name is kept verbatim (underscores there are part of the name).
func splitTestName(name string) []string {
	parts := strings.Split(name, "/")
	for i, p := range parts {
		if i > 0 {
			// Go replaces spaces with underscores in subtest names; restore them.
			parts[i] = strings.ReplaceAll(p, "_", " ")
		}
	}
	return parts
}
