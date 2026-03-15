// Package testify integrates Skipper with testify/suite.
//
// Embed SkipperSuite in your test suite struct instead of suite.Suite
// and set the Config field before calling suite.Run:
//
//	type AuthSuite struct {
//	    skippertestify.SkipperSuite
//	}
//
//	func (s *AuthSuite) TestLogin() {
//	    // SetupTest already called t.Skip() if this test is disabled.
//	    s.Equal(200, resp.StatusCode)
//	}
//
//	func TestAuthSuite(t *testing.T) {
//	    s := &AuthSuite{
//	        SkipperSuite: skippertestify.SkipperSuite{
//	            Config: core.SkipperConfig{
//	                SpreadsheetID: "your-spreadsheet-id",
//	                Credentials:   core.FileCredentials{Path: "./service-account.json"},
//	            },
//	        },
//	    }
//	    suite.Run(t, s)
//	}
package testify

import (
	"context"
	"fmt"
	"os"

	"github.com/get-skipper/skipper-go/core"
	"github.com/stretchr/testify/suite"
)

// SkipperSuite is an embeddable struct that hooks into testify's suite lifecycle
// to skip tests that are disabled in the configured Google Spreadsheet.
// Set the Config field before suite.Run is called.
type SkipperSuite struct {
	suite.Suite
	Config        core.SkipperConfig
	resolver      *core.SkipperResolver
	discoveredIDs []string
}

// SetupSuite initializes the Skipper resolver once before the suite's tests run.
// If SKIPPER_CACHE_FILE is set, the resolver is rehydrated from the cache file
// instead of fetching from Google Sheets.
func (s *SkipperSuite) SetupSuite() {
	if cacheFile := os.Getenv("SKIPPER_CACHE_FILE"); cacheFile != "" {
		core.Logf("rehydrating resolver from cache file %s", cacheFile)
		data, err := core.CacheManager{}.ReadResolverCache(cacheFile)
		if err != nil {
			s.FailNow(fmt.Sprintf("[skipper] cannot read cache file: %v", err))
			return
		}
		r, err := core.FromMarshaledCache(data)
		if err != nil {
			s.FailNow(fmt.Sprintf("[skipper] cannot rehydrate resolver: %v", err))
			return
		}
		s.resolver = r
		return
	}

	r := core.NewSkipperResolver(s.Config)
	if err := r.Initialize(context.Background()); err != nil {
		s.FailNow(fmt.Sprintf("[skipper] initialization failed: %v", err))
		return
	}
	s.resolver = r
}

// SetupTest is called before each test method in the suite. It records the
// test ID for sync mode discovery and calls t.Skip if the test is disabled.
func (s *SkipperSuite) SetupTest() {
	if s.resolver == nil {
		return
	}

	testID := testIDFromSuite(s.T().Name())
	s.discoveredIDs = append(s.discoveredIDs, testID)

	if !s.resolver.IsTestEnabled(testID) {
		until := s.resolver.GetDisabledUntil(testID)
		msg := "[skipper] Test disabled"
		if until != nil {
			msg += " until " + until.Format("2006-01-02")
		}
		s.T().Skip(msg)
	}
}

// TearDownSuite flushes discovered IDs and, in sync mode, reconciles the
// spreadsheet after all suite tests have run.
func (s *SkipperSuite) TearDownSuite() {
	if s.resolver == nil {
		return
	}

	// Flush discovered IDs to the shared directory if one is configured.
	if dir := os.Getenv("SKIPPER_DISCOVERED_DIR"); dir != "" {
		if err := (core.CacheManager{}).WriteDiscoveredIDs(dir, s.discoveredIDs); err != nil {
			core.Warn(fmt.Sprintf("could not write discovered IDs: %v", err))
		}
	}

	if core.SkipperModeFromEnv() != core.SkipperModeSync {
		return
	}

	writer := core.NewSheetsWriter(s.Config)
	if err := writer.Sync(context.Background(), s.discoveredIDs); err != nil {
		fmt.Fprintf(os.Stderr, "[skipper] sync failed: %v\n", err)
	}
}
