//go:build integration

// Integration tests for the SkipperResolver against a real Google Spreadsheet.
//
// Run with:
//
//	go test -tags integration ./...
//
// Required environment variables or files:
//   - GOOGLE_CREDS_B64: base64-encoded service account JSON, OR
//   - service-account-skipper-bot.json in the working directory
//   - SKIPPER_SPREADSHEET_ID: spreadsheet ID (default: hardcoded test spreadsheet)

package core

import (
	"context"
	"os"
	"testing"
)

const (
	testSpreadsheetID = "1Nbjfhklw11uVbi6OCOSeCJI_PThJYzQLlZQbThb4Zvs"
	testSheetName     = "skipper-go"
)

// testCredentials returns the credentials to use for integration tests.
// Priority:
//  1. GOOGLE_CREDS_B64 env var
//  2. service-account-skipper-bot.json in the working directory
func testCredentials(t *testing.T) Credentials {
	t.Helper()
	if b64 := os.Getenv("GOOGLE_CREDS_B64"); b64 != "" {
		return Base64Credentials{Encoded: b64}
	}
	if _, err := os.Stat("service-account-skipper-bot.json"); err == nil {
		return FileCredentials{Path: "service-account-skipper-bot.json"}
	}
	// Try the parent directory (when running from a subdirectory).
	if _, err := os.Stat("../service-account-skipper-bot.json"); err == nil {
		return FileCredentials{Path: "../service-account-skipper-bot.json"}
	}
	t.Skip("no credentials available: set GOOGLE_CREDS_B64 or provide service-account-skipper-bot.json")
	return nil
}

func testSpreadsheetConfig(t *testing.T) SkipperConfig {
	t.Helper()
	spreadsheetID := os.Getenv("SKIPPER_SPREADSHEET_ID")
	if spreadsheetID == "" {
		spreadsheetID = testSpreadsheetID
	}
	return SkipperConfig{
		SpreadsheetID: spreadsheetID,
		Credentials:   testCredentials(t),
		SheetName:     testSheetName,
	}
}

func TestIntegration_ResolverInitialize(t *testing.T) {
	config := testSpreadsheetConfig(t)
	r := NewSkipperResolver(config)

	if err := r.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Logf("resolver initialized successfully")
}

func TestIntegration_UnknownTestIsEnabled(t *testing.T) {
	config := testSpreadsheetConfig(t)
	r := NewSkipperResolver(config)
	if err := r.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	testID := "tests/nonexistent_test.go > TestThatDoesNotExist"
	if !r.IsTestEnabled(testID) {
		t.Errorf("expected unknown test %q to be enabled", testID)
	}
}

func TestIntegration_MarshalCacheRoundTrip(t *testing.T) {
	config := testSpreadsheetConfig(t)
	r := NewSkipperResolver(config)
	if err := r.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	data, err := r.MarshalCache()
	if err != nil {
		t.Fatalf("MarshalCache: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty cache data")
	}

	restored, err := FromMarshaledCache(data)
	if err != nil {
		t.Fatalf("FromMarshaledCache: %v", err)
	}

	// The restored resolver should behave identically to the original.
	testID := "tests/nonexistent_test.go > TestThatDoesNotExist"
	if original, restored := r.IsTestEnabled(testID), restored.IsTestEnabled(testID); original != restored {
		t.Errorf("IsTestEnabled(%q): original=%v, restored=%v", testID, original, restored)
	}
	t.Logf("cache round-trip successful, %d bytes", len(data))
}

// TestIntegration_Example is the "example" test that will appear in the
// spreadsheet when sync mode is enabled. Its test ID is registered so that
// SKIPPER_MODE=sync adds it (or keeps it) in the "skipper-go" sheet.
func TestIntegration_Example(t *testing.T) {
	config := testSpreadsheetConfig(t)
	r := NewSkipperResolver(config)
	if err := r.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	testID := BuildTestID("core/integration_test.go", []string{"TestIntegration_Example"})
	t.Logf("test ID: %s", testID)

	if !r.IsTestEnabled(testID) {
		until := r.GetDisabledUntil(testID)
		msg := "[skipper] Test disabled"
		if until != nil {
			msg += " until " + until.Format("2006-01-02")
		}
		t.Skip(msg)
	}
}
