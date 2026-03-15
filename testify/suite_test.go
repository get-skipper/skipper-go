package testify_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/get-skipper/skipper-go/core"
	skippertestify "github.com/get-skipper/skipper-go/testify"
	"github.com/stretchr/testify/suite"
)

// ---- ExampleSuite: a basic suite that embeds SkipperSuite ----

type ExampleSuite struct {
	skippertestify.SkipperSuite
	ranTests []string
}

func (s *ExampleSuite) TestAlwaysEnabled() {
	s.ranTests = append(s.ranTests, "TestAlwaysEnabled")
}

func (s *ExampleSuite) TestAnotherEnabled() {
	s.ranTests = append(s.ranTests, "TestAnotherEnabled")
}

// ---- Tests ----

// TestSkipperSuite_RunsWithoutCredentials verifies that a suite with no Config
// set (nil resolver) runs all tests normally without panicking.
func TestSkipperSuite_RunsWithoutCredentials(t *testing.T) {
	s := &ExampleSuite{}
	suite.Run(t, s)

	if len(s.ranTests) != 2 {
		t.Errorf("expected 2 tests to run, got %d: %v", len(s.ranTests), s.ranTests)
	}
}

// TestSkipperSuite_SetupSuiteReadsFromCacheFile verifies that when
// SKIPPER_CACHE_FILE is set, SetupSuite rehydrates the resolver from it
// instead of calling Initialize (which would require Google Sheets access).
func TestSkipperSuite_SetupSuiteReadsFromCacheFile(t *testing.T) {
	// Build a JSON cache that disables a test that will never run in this suite,
	// so we can verify the resolver was loaded (not nil) without test skips.
	future := time.Now().AddDate(0, 0, 30).UTC().Format(time.RFC3339)
	cacheData, _ := json.Marshal(map[string]*string{
		"tests/placeholder.go > testplaceholder": &future,
	})

	dir, err := core.CacheManager{}.WriteResolverCache(cacheData)
	if err != nil {
		t.Fatal(err)
	}
	defer core.CacheManager{}.Cleanup(dir)

	prev := os.Getenv("SKIPPER_CACHE_FILE")
	os.Setenv("SKIPPER_CACHE_FILE", dir+"/cache.json")
	defer func() {
		if prev == "" {
			os.Unsetenv("SKIPPER_CACHE_FILE")
		} else {
			os.Setenv("SKIPPER_CACHE_FILE", prev)
		}
	}()

	// Suite should initialize from cache and run all tests (placeholder test
	// is disabled but doesn't exist in this suite, so nothing is skipped).
	s := &ExampleSuite{}
	suite.Run(t, s)

	if len(s.ranTests) != 2 {
		t.Errorf("expected 2 tests to run (cache-loaded resolver), got %d: %v",
			len(s.ranTests), s.ranTests)
	}
}

// TestSkipperSuite_WithLiveSpreadsheet runs the ExampleSuite against the real
// spreadsheet when credentials are available.
func TestSkipperSuite_WithLiveSpreadsheet(t *testing.T) {
	creds := resolveCredentials()
	if creds == nil {
		t.Skip("no credentials available: set GOOGLE_CREDS_B64 or provide service-account-skipper-bot.json")
	}

	s := &ExampleSuite{
		SkipperSuite: skippertestify.SkipperSuite{
			Config: core.SkipperConfig{
				SpreadsheetID: spreadsheetID(),
				Credentials:   creds,
				SheetName:     "skipper-go",
			},
		},
	}
	suite.Run(t, s)
}

func resolveCredentials() core.Credentials {
	if b64 := os.Getenv("GOOGLE_CREDS_B64"); b64 != "" {
		return core.Base64Credentials{Encoded: b64}
	}
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
