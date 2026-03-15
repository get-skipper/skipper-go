// export_test.go exposes internal helpers to external test packages.
// This file is only compiled when running tests.

package core

import "time"

// NewResolverWithCache creates a SkipperResolver pre-populated with the given
// cache map. Used by integration-package tests that need a resolver without
// making real Google Sheets API calls.
func NewResolverWithCache(cache map[string]*time.Time) *SkipperResolver {
	return &SkipperResolver{cache: cache}
}
