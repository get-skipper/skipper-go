package core

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	stdtesting "testing"
)

var (
	integrationConfig     SkipperConfig
	integrationResolver   *SkipperResolver
	integrationDiscovered []string
	integrationDiscoveredMu sync.Mutex
)

func TestMain(m *stdtesting.M) {
	ctx := context.Background()

	creds := resolveIntegrationCreds()
	if creds == nil {
		// No credentials: run tests anyway (each will skip via testCredentials(t)).
		os.Exit(m.Run())
	}

	spreadsheetID := os.Getenv("SKIPPER_SPREADSHEET_ID")
	if spreadsheetID == "" {
		spreadsheetID = testSpreadsheetID
	}
	integrationConfig = SkipperConfig{
		SpreadsheetID: spreadsheetID,
		Credentials:   creds,
		SheetName:     testSheetName,
	}

	r := NewSkipperResolver(integrationConfig)
	if err := r.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[skipper] initialization failed: %v\n", err)
		os.Exit(1)
	}
	integrationResolver = r

	code := m.Run()

	if SkipperModeFromEnv() == SkipperModeSync {
		integrationDiscoveredMu.Lock()
		ids := make([]string, len(integrationDiscovered))
		copy(ids, integrationDiscovered)
		integrationDiscoveredMu.Unlock()

		scanned := ScanPackageTests()
		seen := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			seen[NormalizeTestID(id)] = struct{}{}
		}
		for _, id := range scanned {
			if _, ok := seen[NormalizeTestID(id)]; !ok {
				ids = append(ids, id)
			}
		}

		writer := NewSheetsWriter(integrationConfig)
		if err := writer.Sync(ctx, ids); err != nil {
			fmt.Fprintf(os.Stderr, "[skipper] sync failed: %v\n", err)
		}
	}

	os.Exit(code)
}

// integrationSkipIfDisabled records the calling test's ID for sync and skips
// the test if it is currently disabled in the spreadsheet.
func integrationSkipIfDisabled(t *stdtesting.T) {
	t.Helper()
	if integrationResolver == nil {
		return
	}

	file := integrationCallerFile()
	parts := integrationSplitTestName(t.Name())
	testID := BuildTestID(file, parts)

	integrationDiscoveredMu.Lock()
	integrationDiscovered = append(integrationDiscovered, testID)
	integrationDiscoveredMu.Unlock()

	if !integrationResolver.IsTestEnabled(testID) {
		until := integrationResolver.GetDisabledUntil(testID)
		msg := "[skipper] Test disabled"
		if until != nil {
			msg += " until " + until.Format("2006-01-02")
		}
		t.Skip(msg)
	}
}

func resolveIntegrationCreds() Credentials {
	if b64 := os.Getenv("GOOGLE_CREDS_B64"); b64 != "" {
		return Base64Credentials{Encoded: b64}
	}
	if _, err := os.Stat("service-account-skipper-bot.json"); err == nil {
		return FileCredentials{Path: "service-account-skipper-bot.json"}
	}
	if _, err := os.Stat("../service-account-skipper-bot.json"); err == nil {
		return FileCredentials{Path: "../service-account-skipper-bot.json"}
	}
	return nil
}

func integrationCallerFile() string {
	// Skip frames from this file (integration_main_test.go) and find the
	// first _test.go frame that belongs to a different file.
	self := "integration_main_test.go"
	for depth := 1; depth < 30; depth++ {
		_, file, _, ok := runtime.Caller(depth)
		if !ok {
			break
		}
		if strings.HasSuffix(file, "_test.go") && !strings.HasSuffix(file, self) {
			return file
		}
	}
	return "unknown"
}

func integrationSplitTestName(name string) []string {
	parts := strings.Split(name, "/")
	for i, p := range parts {
		if i > 0 {
			// Restore spaces that Go replaced with underscores in subtest names.
			parts[i] = strings.ReplaceAll(p, "_", " ")
		}
	}
	return parts
}
