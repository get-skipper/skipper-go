package core

import (
	"context"
	"errors"
	"os"
	"testing"
)

var errFakeAPI = errors.New("fake API error")

func fetchError(_ context.Context) (*FetchAllResult, error) {
	return nil, errFakeAPI
}

func fetchEmpty(_ context.Context) (*FetchAllResult, error) {
	return &FetchAllResult{}, nil
}

// ---- SKIPPER_FAIL_OPEN -------------------------------------------------------

func TestFailOpen_DefaultIsTrue(t *testing.T) {
	t.Setenv("SKIPPER_FAIL_OPEN", "")
	if !FailOpen() {
		t.Error("expected FailOpen() == true when env var is unset")
	}
}

func TestFailOpen_ExplicitTrue(t *testing.T) {
	t.Setenv("SKIPPER_FAIL_OPEN", "true")
	if !FailOpen() {
		t.Error("expected FailOpen() == true")
	}
}

func TestFailOpen_ExplicitFalse(t *testing.T) {
	t.Setenv("SKIPPER_FAIL_OPEN", "false")
	if FailOpen() {
		t.Error("expected FailOpen() == false")
	}
}

func TestFailOpen_InvalidValueDefaultsToTrue(t *testing.T) {
	t.Setenv("SKIPPER_FAIL_OPEN", "maybe")
	if !FailOpen() {
		t.Error("expected FailOpen() == true for invalid value")
	}
}

func TestInitialize_FailOpen_ReturnsNilOnAPIError(t *testing.T) {
	t.Setenv("SKIPPER_FAIL_OPEN", "true")
	t.Setenv("SKIPPER_CACHE_TTL", "0") // disable disk cache
	r := NewResolverWithFetch(fetchError)
	if err := r.Initialize(context.Background()); err != nil {
		t.Errorf("expected nil error in fail-open mode, got %v", err)
	}
	// All tests should be enabled (empty cache = opt-out model).
	if !r.IsTestEnabled("any/test.go > AnyTest") {
		t.Error("expected all tests enabled in fail-open mode")
	}
}

func TestInitialize_FailClosed_ReturnsErrorOnAPIError(t *testing.T) {
	t.Setenv("SKIPPER_FAIL_OPEN", "false")
	t.Setenv("SKIPPER_CACHE_TTL", "0") // disable disk cache
	r := NewResolverWithFetch(fetchError)
	if err := r.Initialize(context.Background()); err == nil {
		t.Error("expected error in fail-closed mode")
	}
}

// ---- SKIPPER_CACHE_TTL -------------------------------------------------------

func TestCacheTTL_DefaultIs300s(t *testing.T) {
	t.Setenv("SKIPPER_CACHE_TTL", "")
	if got := CacheTTL().Seconds(); got != 300 {
		t.Errorf("expected 300s, got %v", got)
	}
}

func TestCacheTTL_Zero_DisablesCache(t *testing.T) {
	t.Setenv("SKIPPER_CACHE_TTL", "0")
	if got := CacheTTL(); got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

func TestCacheTTL_CustomValue(t *testing.T) {
	t.Setenv("SKIPPER_CACHE_TTL", "60")
	if got := CacheTTL().Seconds(); got != 60 {
		t.Errorf("expected 60s, got %v", got)
	}
}

func TestCacheTTL_InvalidValueDefaultsTo300s(t *testing.T) {
	t.Setenv("SKIPPER_CACHE_TTL", "not-a-number")
	if got := CacheTTL().Seconds(); got != 300 {
		t.Errorf("expected 300s for invalid value, got %v", got)
	}
}

func TestInitialize_WritesDiskCacheOnSuccess(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = dir + "/.skipper-cache.json"
	t.Cleanup(func() { DiskCacheFile = origFile })

	t.Setenv("SKIPPER_CACHE_TTL", "300")

	r := NewResolverWithFetch(fetchEmpty)
	if err := r.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if _, err := os.Stat(DiskCacheFile); os.IsNotExist(err) {
		t.Error("expected disk cache file to be created after successful Initialize")
	}
}

func TestInitialize_UsesDiskCacheWhenAPIFails(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = dir + "/.skipper-cache.json"
	t.Cleanup(func() { DiskCacheFile = origFile })

	t.Setenv("SKIPPER_CACHE_TTL", "300")
	t.Setenv("SKIPPER_FAIL_OPEN", "false") // would fail hard without cache

	// First: populate the disk cache via a successful fetch.
	r1 := NewResolverWithFetch(fetchEmpty)
	if err := r1.Initialize(context.Background()); err != nil {
		t.Fatalf("first Initialize: %v", err)
	}

	// Second: API fails; resolver should load from disk cache.
	r2 := NewResolverWithFetch(fetchError)
	if err := r2.Initialize(context.Background()); err != nil {
		t.Errorf("expected nil when disk cache is available, got %v", err)
	}
}

func TestInitialize_IgnoresExpiredDiskCache(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = dir + "/.skipper-cache.json"
	t.Cleanup(func() { DiskCacheFile = origFile })

	// Write a cache that is older than the TTL by writing directly.
	if err := os.WriteFile(DiskCacheFile, []byte(`{"written_at":"2000-01-01T00:00:00Z","data":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SKIPPER_CACHE_TTL", "300")
	t.Setenv("SKIPPER_FAIL_OPEN", "true") // fall back to fail-open after expired cache

	r := NewResolverWithFetch(fetchError)
	if err := r.Initialize(context.Background()); err != nil {
		t.Errorf("expected nil (fail-open after expired cache), got %v", err)
	}
}

func TestInitialize_NoDiskCacheWhenTTLIsZero(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = dir + "/.skipper-cache.json"
	t.Cleanup(func() { DiskCacheFile = origFile })

	t.Setenv("SKIPPER_CACHE_TTL", "0") // disabled

	r := NewResolverWithFetch(fetchEmpty)
	if err := r.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if _, err := os.Stat(DiskCacheFile); !os.IsNotExist(err) {
		t.Error("expected no disk cache file when TTL=0")
	}
}
