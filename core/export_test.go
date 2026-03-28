// export_test.go exposes internal helpers to external test packages.
// This file is only compiled when running tests.

package core

import (
	"context"
	"time"
)

// NewResolverWithCache creates a SkipperResolver pre-populated with the given
// cache map. Used by integration-package tests that need a resolver without
// making real Google Sheets API calls.
func NewResolverWithCache(cache map[string]*time.Time) *SkipperResolver {
	return &SkipperResolver{cache: cache}
}

// NewResolverWithFetch creates a SkipperResolver that calls fetchFn instead of
// the real Google Sheets API. Used to test Initialize() behaviour (fail-open,
// disk-cache fallback) without network access.
func NewResolverWithFetch(fetchFn func(ctx context.Context) (*FetchAllResult, error)) *SkipperResolver {
	return &SkipperResolver{
		cache:   make(map[string]*time.Time),
		fetchFn: fetchFn,
	}
}
