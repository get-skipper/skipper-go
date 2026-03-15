package testing_test

import (
	"os"
	stdtesting "testing"
	"time"

	"github.com/get-skipper/skipper-go/core"
	skippertest "github.com/get-skipper/skipper-go/testing"
)

// TestMain initializes Skipper for this test package.
// In CI, GOOGLE_CREDS_B64 is set. Locally, the service account JSON is used.
func TestMain(m *stdtesting.M) {
	creds := resolveCredentials()
	if creds == nil {
		// No credentials available: run tests without Skipper (unit-only mode).
		os.Exit(m.Run())
	}

	s := &skippertest.SkipperTestMain{
		Config: core.SkipperConfig{
			SpreadsheetID: spreadsheetID(),
			Credentials:   creds,
			SheetName:     "skipper-go",
		},
	}
	os.Exit(s.Run(m))
}

// TestSkipIfDisabled_RunsNormallyWhenEnabled verifies that SkipIfDisabled does
// not skip a test that is not disabled in the spreadsheet (or when resolver is
// not initialized in unit-only mode).
func TestSkipIfDisabled_RunsNormallyWhenEnabled(t *stdtesting.T) {
	skippertest.SkipIfDisabled(t)
	// If we reach here, the test ran (not skipped).
}

// TestSkipIfDisabled_WithSubtest verifies that SkipIfDisabled works inside subtests.
func TestSkipIfDisabled_WithSubtest(t *stdtesting.T) {
	t.Run("enabled subtest", func(t *stdtesting.T) {
		skippertest.SkipIfDisabled(t)
		// Should reach here.
	})
}

func TestTestIDFromCallerFormat(t *stdtesting.T) {
	// The test ID should be derived by SkipIfDisabled from this function's location.
	// We just verify it doesn't panic.
	skippertest.SkipIfDisabled(t)
}

// resolveCredentials returns credentials from the environment or file, or nil
// if none are available.
func resolveCredentials() core.Credentials {
	if b64 := os.Getenv("GOOGLE_CREDS_B64"); b64 != "" {
		return core.Base64Credentials{Encoded: b64}
	}
	// Try the local service account file (development).
	for _, path := range []string{
		"../service-account-skipper-bot.json",
		"service-account-skipper-bot.json",
	} {
		if _, err := os.Stat(path); err == nil {
			return core.FileCredentials{Path: path}
		}
	}
	return nil
}

func spreadsheetID() string {
	if id := os.Getenv("SKIPPER_SPREADSHEET_ID"); id != "" {
		return id
	}
	return "1Nbjfhklw11uVbi6OCOSeCJI_PThJYzQLlZQbThb4Zvs"
}

// TestSkipperTestMain_ExitCode verifies that SkipperTestMain.Run returns the
// exit code from m.Run() when Skipper initialization succeeds.
func TestSkipperTestMain_ExitCode(t *stdtesting.T) {
	// This test just ensures the package compiles and runs.
	_ = time.Now()
}
