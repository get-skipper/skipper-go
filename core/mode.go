package core

import "os"

// SkipperMode controls whether Skipper only reads from the spreadsheet
// or also syncs discovered test IDs back to it.
type SkipperMode string

const (
	// SkipperModeReadOnly fetches the spreadsheet and skips disabled tests.
	// This is the default mode.
	SkipperModeReadOnly SkipperMode = "read-only"

	// SkipperModeSync fetches the spreadsheet, skips disabled tests, and
	// then reconciles the spreadsheet with the set of discovered test IDs.
	SkipperModeSync SkipperMode = "sync"
)

// SkipperModeFromEnv returns the mode configured via the SKIPPER_MODE env var.
// Defaults to SkipperModeReadOnly.
func SkipperModeFromEnv() SkipperMode {
	if os.Getenv("SKIPPER_MODE") == "sync" {
		return SkipperModeSync
	}
	return SkipperModeReadOnly
}
