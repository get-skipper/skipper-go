// Package testing integrates Skipper with the standard library testing package.
//
// Usage:
//
//	func TestMain(m *testing.M) {
//	    s := &skippertest.SkipperTestMain{
//	        Config: core.SkipperConfig{
//	            SpreadsheetID: "your-spreadsheet-id",
//	            Credentials:   core.FileCredentials{Path: "./service-account.json"},
//	        },
//	    }
//	    os.Exit(s.Run(m))
//	}
//
//	func TestLogin(t *testing.T) {
//	    skippertest.SkipIfDisabled(t)
//	    // ... test body
//	}
package testing

import (
	"context"
	"fmt"
	"os"
	"sync"
	stdtesting "testing"

	"github.com/get-skipper/skipper-go/core"
)

var (
	globalResolver *core.SkipperResolver
	globalCacheDir string
	discoveredMu   sync.Mutex
	discoveredIDs  []string
	preScannedIDs  []string
)

// SkipperTestMain wraps testing.M to initialize and finalize Skipper.
type SkipperTestMain struct {
	Config core.SkipperConfig
}

// Run initializes Skipper, runs all tests via m.Run(), performs an optional
// sync, cleans up, and returns the exit code. Pass the result to os.Exit.
func (s *SkipperTestMain) Run(m *stdtesting.M) int {
	ctx := context.Background()

	if err := s.initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[skipper] initialization failed: %v\n", err)
		return 1
	}

	code := m.Run()

	if err := s.finalize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[skipper] finalize failed: %v\n", err)
	}

	return code
}

func (s *SkipperTestMain) initialize(ctx context.Context) error {
	// If a cache file is already set (e.g., from an external orchestrator),
	// rehydrate the resolver from it instead of fetching from Google Sheets.
	if cacheFile := os.Getenv("SKIPPER_CACHE_FILE"); cacheFile != "" {
		core.Logf("rehydrating resolver from cache file %s", cacheFile)
		data, err := core.CacheManager{}.ReadResolverCache(cacheFile)
		if err != nil {
			return err
		}
		r, err := core.FromMarshaledCache(data)
		if err != nil {
			return err
		}
		globalResolver = r
		preScannedIDs = core.ScanPackageTests()
		return nil
	}

	r := core.NewSkipperResolver(s.Config)
	if err := r.Initialize(ctx); err != nil {
		return err
	}
	globalResolver = r

	// Write cache to temp dir for any sub-processes.
	data, err := r.MarshalCache()
	if err != nil {
		return err
	}
	dir, err := core.CacheManager{}.WriteResolverCache(data)
	if err != nil {
		return err
	}
	globalCacheDir = dir
	os.Setenv("SKIPPER_CACHE_FILE", dir+"/cache.json")
	os.Setenv("SKIPPER_DISCOVERED_DIR", dir)

	preScannedIDs = core.ScanPackageTests()
	return nil
}

func (s *SkipperTestMain) finalize(ctx context.Context) error {
	// Flush in-memory discovered IDs to the shared directory.
	if globalCacheDir != "" {
		discoveredMu.Lock()
		ids := make([]string, len(discoveredIDs))
		copy(ids, discoveredIDs)
		discoveredMu.Unlock()

		if len(ids) > 0 {
			if err := (core.CacheManager{}).WriteDiscoveredIDs(globalCacheDir, ids); err != nil {
				core.Warn(fmt.Sprintf("could not write discovered IDs: %v", err))
			}
		}
	}

	if core.SkipperModeFromEnv() == core.SkipperModeSync {
		var ids []string
		if globalCacheDir != "" {
			var err error
			ids, err = core.CacheManager{}.MergeDiscoveredIDs(globalCacheDir)
			if err != nil {
				core.Warn(fmt.Sprintf("could not merge discovered IDs: %v", err))
				discoveredMu.Lock()
				ids = make([]string, len(discoveredIDs))
				copy(ids, discoveredIDs)
				discoveredMu.Unlock()
			}
		} else {
			discoveredMu.Lock()
			ids = make([]string, len(discoveredIDs))
			copy(ids, discoveredIDs)
			discoveredMu.Unlock()
		}

		// Merge pre-scanned IDs (tests without SkipIfDisabled) into the sync list.
		existing := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			existing[core.NormalizeTestID(id)] = struct{}{}
		}
		for _, id := range preScannedIDs {
			if _, ok := existing[core.NormalizeTestID(id)]; !ok {
				ids = append(ids, id)
			}
		}

		writer := core.NewSheetsWriter(s.Config)
		if err := writer.Sync(ctx, ids); err != nil {
			return err
		}
	}

	if globalCacheDir != "" {
		core.CacheManager{}.Cleanup(globalCacheDir)
	}
	return nil
}

// SkipIfDisabled calls t.Skip if the test is currently disabled in the
// spreadsheet. It also records the test ID for sync mode discovery.
//
// Call this at the top of every test function and subtest.
func SkipIfDisabled(t *stdtesting.T) {
	t.Helper()

	if globalResolver == nil {
		// Skipper not initialized — skip silently (no TestMain configured).
		return
	}

	testID := testIDFromCaller(t.Name(), 2)

	discoveredMu.Lock()
	discoveredIDs = append(discoveredIDs, testID)
	discoveredMu.Unlock()

	if !globalResolver.IsTestEnabled(testID) {
		until := globalResolver.GetDisabledUntil(testID)
		msg := "[skipper] Test disabled"
		if until != nil {
			msg += " until " + until.Format("2006-01-02")
		}
		t.Skip(msg)
	}
}
