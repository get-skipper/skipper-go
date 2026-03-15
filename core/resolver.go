package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SkipperResolver fetches a Google Spreadsheet and determines whether
// individual tests should run based on their disabledUntil dates.
type SkipperResolver struct {
	config SkipperConfig
	client *SheetsClient
	// cache maps normalized test IDs to their disabledUntil date.
	// A nil value means the test has no date and is enabled.
	cache map[string]*time.Time
}

// NewSkipperResolver creates a new resolver. Call Initialize before use.
func NewSkipperResolver(config SkipperConfig) *SkipperResolver {
	return &SkipperResolver{
		config: config,
		client: NewSheetsClient(config),
		cache:  make(map[string]*time.Time),
	}
}

// Initialize fetches the spreadsheet and populates the internal cache.
// Must be called once before IsTestEnabled.
func (r *SkipperResolver) Initialize(ctx context.Context) error {
	Log("initializing resolver")
	result, err := r.client.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("skipper: initialize failed: %w", err)
	}

	for _, entry := range result.Entries {
		nid := NormalizeTestID(entry.TestID)
		r.cache[nid] = entry.DisabledUntil
	}

	Logf("loaded %d test entries from spreadsheet", len(r.cache))
	return nil
}

// IsTestEnabled reports whether a test should run.
//
//   - Tests not in the spreadsheet always run (opt-out model).
//   - Tests with a nil or past disabledUntil date run normally.
//   - Tests with a future disabledUntil date are skipped.
func (r *SkipperResolver) IsTestEnabled(testID string) bool {
	nid := NormalizeTestID(testID)
	disabledUntil, found := r.cache[nid]
	if !found {
		return true
	}
	if disabledUntil == nil {
		return true
	}
	return !disabledUntil.After(time.Now())
}

// GetDisabledUntil returns the disabledUntil date for a test ID, or nil if
// the test is not in the spreadsheet or has no date set.
func (r *SkipperResolver) GetDisabledUntil(testID string) *time.Time {
	nid := NormalizeTestID(testID)
	return r.cache[nid]
}

// cacheJSON is the serialized representation used for cross-process sharing.
// A nil time is serialized as JSON null; a non-nil time as an RFC3339 string.
type cacheJSON map[string]*string

// MarshalCache serializes the resolver cache to JSON bytes for sharing
// with worker processes via a temp file.
func (r *SkipperResolver) MarshalCache() ([]byte, error) {
	out := make(cacheJSON, len(r.cache))
	for k, v := range r.cache {
		if v == nil {
			out[k] = nil
		} else {
			s := v.Format(time.RFC3339)
			out[k] = &s
		}
	}
	return json.Marshal(out)
}

// FromMarshaledCache rehydrates a SkipperResolver from JSON bytes
// produced by MarshalCache. The resulting resolver does not need
// Initialize to be called again.
func FromMarshaledCache(data []byte) (*SkipperResolver, error) {
	var raw cacheJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("skipper: cannot unmarshal cache: %w", err)
	}

	cache := make(map[string]*time.Time, len(raw))
	for k, v := range raw {
		if v == nil {
			cache[k] = nil
		} else {
			t, err := time.Parse(time.RFC3339, *v)
			if err != nil {
				return nil, fmt.Errorf("skipper: invalid date in cache for %q: %w", k, err)
			}
			cache[k] = &t
		}
	}

	return &SkipperResolver{cache: cache}, nil
}
