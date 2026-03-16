package testing_test

import (
	stdtesting "testing"

	skippertest "github.com/get-skipper/skipper-go/testing"
)

func TestUsageExample(t *stdtesting.T) {
	skippertest.SkipIfDisabled(t)
}
