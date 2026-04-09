package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// SkipperResolver fetches a Google Spreadsheet and determines whether
// individual tests should run based on their disabledUntil dates.
type SkipperResolver struct {
	config  SkipperConfig
	client  *SheetsClient
	fetchFn func(ctx context.Context) (*FetchAllResult, error) // nil in production; injected in tests
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
//
// Safety behaviours (all configurable via environment variables):
//   - SKIPPER_FAIL_OPEN (default "true"): when the API is unreachable and no
//     usable disk cache exists, return nil instead of an error so that all
//     tests are allowed to run.
//   - SKIPPER_CACHE_TTL (default "300"): seconds to keep the on-disk cache
//     (.skipper-cache.json) valid. Set to "0" to disable disk caching.
func (r *SkipperResolver) Initialize(ctx context.Context) error {
	Log("initializing resolver")

	fetch := r.fetchFn
	if fetch == nil {
		fetch = r.client.FetchAll
	}

	result, err := fetch(ctx)
	if err != nil {
		// API unavailable — try the on-disk TTL cache first.
		if ttl := CacheTTL(); ttl > 0 {
			if data, cerr := LoadDiskCache(ttl); cerr == nil {
				restored, rerr := FromMarshaledCache(data)
				if rerr == nil {
					r.cache = restored.cache
					Logf("API unavailable; loaded resolver cache from disk: %v", err)
					return nil
				}
				Warn(fmt.Sprintf("disk cache unreadable: %v", rerr))
			}
		}

		// No usable cache — honour SKIPPER_FAIL_OPEN (default: true).
		if FailOpen() {
			Warn(fmt.Sprintf("skipper: initialize failed, running in fail-open mode (all tests enabled): %v", err))
			return nil
		}
		return fmt.Errorf("skipper: initialize failed: %w", err)
	}

	for _, entry := range result.Entries {
		nid := NormalizeTestID(entry.TestID)
		r.cache[nid] = entry.DisabledUntil
	}

	Logf("loaded %d test entries from spreadsheet", len(r.cache))

	// Persist a fresh disk cache for future fail-over use.
	if ttl := CacheTTL(); ttl > 0 {
		data, merr := r.MarshalCache()
		if merr == nil {
			if werr := WriteDiskCache(data); werr != nil {
				Warn(fmt.Sprintf("could not write disk cache: %v", werr))
			} else {
				Logf("wrote disk cache to %s (TTL %s)", DiskCacheFile, ttl)
			}
		}
	}

	return nil
}

// FailOpen returns the value of SKIPPER_FAIL_OPEN (default: true).
// When true, Initialize returns nil instead of an error if the API is
// unreachable and no usable disk cache exists.
func FailOpen() bool {
	v := os.Getenv("SKIPPER_FAIL_OPEN")
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}

// CacheTTL returns the disk-cache TTL derived from SKIPPER_CACHE_TTL
// (default: 300 s). A value of 0 disables the disk cache entirely.
func CacheTTL() time.Duration {
	v := os.Getenv("SKIPPER_CACHE_TTL")
	if v == "" {
		return 300 * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 300 * time.Second
	}
	return time.Duration(n) * time.Second
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
	return !time.Now().UTC().Before(*disabledUntil)
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
