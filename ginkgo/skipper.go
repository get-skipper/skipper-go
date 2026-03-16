// Package ginkgo integrates Skipper with Ginkgo v2.
//
// Call RegisterSkipperHooks once in your suite bootstrap function,
// before RunSpecs:
//
//	func TestAuth(t *testing.T) {
//	    RegisterFailHandler(gomega.Fail)
//	    skipperginkgo.RegisterSkipperHooks(core.SkipperConfig{
//	        SpreadsheetID: "your-spreadsheet-id",
//	        Credentials:   core.FileCredentials{Path: "./service-account.json"},
//	    })
//	    RunSpecs(t, "Auth Suite")
//	}
//
// In parallel mode (ginkgo -p), only the primary Ginkgo node fetches from
// Google Sheets; all other nodes rehydrate from the serialized cache.
package ginkgo

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/get-skipper/skipper-go/core"
	ginkgo "github.com/onsi/ginkgo/v2"
)

// RegisterSkipperHooks installs Ginkgo lifecycle hooks that initialize the
// Skipper resolver, skip disabled specs, and (in sync mode) reconcile the
// spreadsheet after the suite finishes.
func RegisterSkipperHooks(config core.SkipperConfig) {
	var (
		resolver      *core.SkipperResolver
		discoveredMu  sync.Mutex
		discoveredIDs []string
	)

	// SynchronizedBeforeSuite ensures only the primary Ginkgo node (node 1)
	// fetches from Google Sheets. All other nodes rehydrate from the
	// serialized cache bytes returned by node 1.
	ginkgo.SynchronizedBeforeSuite(
		func() []byte {
			// Primary node: initialize and serialize the resolver.
			r := core.NewSkipperResolver(config)
			if err := r.Initialize(context.Background()); err != nil {
				ginkgo.Fail(fmt.Sprintf("[skipper] initialization failed: %v", err))
				return nil
			}
			data, err := r.MarshalCache()
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("[skipper] cannot serialize cache: %v", err))
				return nil
			}
			resolver = r
			return data
		},
		func(data []byte) {
			// All nodes: rehydrate from primary's data.
			r, err := core.FromMarshaledCache(data)
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("[skipper] cannot rehydrate cache: %v", err))
				return
			}
			resolver = r
		},
	)

	// BeforeEach runs before every It block. It records the spec's test ID
	// for sync mode and skips the spec if it is currently disabled.
	ginkgo.BeforeEach(func() {
		if resolver == nil {
			return
		}
		report := ginkgo.CurrentSpecReport()
		testID := testIDFromReport(report)

		discoveredMu.Lock()
		discoveredIDs = append(discoveredIDs, testID)
		discoveredMu.Unlock()

		if !resolver.IsTestEnabled(testID) {
			until := resolver.GetDisabledUntil(testID)
			msg := "[skipper] Test disabled"
			if until != nil {
				msg += " until " + until.Format("2006-01-02")
			}
			ginkgo.Skip(msg)
		}
	})

	// AfterSuite reconciles the spreadsheet in sync mode.
	// Only the primary node performs the sync to avoid race conditions.
	ginkgo.SynchronizedAfterSuite(
		func() {
			// All nodes: nothing to do.
		},
		func() {
			// Primary node only.
			if core.SkipperModeFromEnv() != core.SkipperModeSync {
				return
			}

			discoveredMu.Lock()
			ids := make([]string, len(discoveredIDs))
			copy(ids, discoveredIDs)
			discoveredMu.Unlock()

			scanned := core.ScanPackageTests()
			seen := make(map[string]struct{}, len(ids))
			for _, id := range ids {
				seen[core.NormalizeTestID(id)] = struct{}{}
			}
			for _, id := range scanned {
				if _, ok := seen[core.NormalizeTestID(id)]; !ok {
					ids = append(ids, id)
				}
			}

			writer := core.NewSheetsWriter(config)
			if err := writer.Sync(context.Background(), ids); err != nil {
				fmt.Fprintf(os.Stderr, "[skipper] sync failed: %v\n", err)
			}
		},
	)
}
