package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheManager_WriteAndReadResolverCache(t *testing.T) {
	cm := CacheManager{}
	data := []byte(`{"some/test.go > testfoo": null}`)

	dir, err := cm.WriteResolverCache(data)
	if err != nil {
		t.Fatalf("WriteResolverCache: %v", err)
	}
	defer cm.Cleanup(dir)

	cacheFile := filepath.Join(dir, "cache.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatalf("cache file not created at %s", cacheFile)
	}

	got, err := cm.ReadResolverCache(cacheFile)
	if err != nil {
		t.Fatalf("ReadResolverCache: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("got %q, want %q", string(got), string(data))
	}
}

func TestCacheManager_ReadResolverCache_MissingFile(t *testing.T) {
	cm := CacheManager{}
	_, err := cm.ReadResolverCache("/nonexistent/cache.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestCacheManager_WriteDiscoveredIDs(t *testing.T) {
	cm := CacheManager{}
	dir, err := os.MkdirTemp("", "skipper-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	ids := []string{"tests/auth_test.go > TestLogin", "tests/auth_test.go > TestLogout"}
	if err := cm.WriteDiscoveredIDs(dir, ids); err != nil {
		t.Fatalf("WriteDiscoveredIDs: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 discovered file, got %d", len(entries))
	}
}

func TestCacheManager_WriteDiscoveredIDs_Empty(t *testing.T) {
	cm := CacheManager{}
	dir, err := os.MkdirTemp("", "skipper-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Writing empty IDs should still create a file (consistent behavior).
	if err := cm.WriteDiscoveredIDs(dir, []string{}); err != nil {
		t.Fatalf("WriteDiscoveredIDs with empty slice: %v", err)
	}
}

func TestCacheManager_MergeDiscoveredIDs(t *testing.T) {
	cm := CacheManager{}
	dir, err := os.MkdirTemp("", "skipper-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write cache.json — should be ignored by merge.
	if err := os.WriteFile(filepath.Join(dir, "cache.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Write discovered IDs from two "processes" with a duplicate.
	writeJSON(t, dir, "1-file.json", []string{"tests/a.go > TestA", "tests/b.go > TestB"})
	writeJSON(t, dir, "2-file.json", []string{"tests/b.go > TestB", "tests/c.go > TestC"}) // TestB is duplicate

	merged, err := cm.MergeDiscoveredIDs(dir)
	if err != nil {
		t.Fatalf("MergeDiscoveredIDs: %v", err)
	}

	if len(merged) != 3 {
		t.Errorf("expected 3 unique IDs, got %d: %v", len(merged), merged)
	}
}

func TestCacheManager_MergeDiscoveredIDs_EmptyDir(t *testing.T) {
	cm := CacheManager{}
	dir, err := os.MkdirTemp("", "skipper-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	merged, err := cm.MergeDiscoveredIDs(dir)
	if err != nil {
		t.Fatalf("MergeDiscoveredIDs on empty dir: %v", err)
	}
	if len(merged) != 0 {
		t.Errorf("expected 0 IDs, got %d", len(merged))
	}
}

func TestCacheManager_Cleanup(t *testing.T) {
	cm := CacheManager{}
	dir, err := cm.WriteResolverCache([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := cm.Cleanup(dir); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected dir %s to be removed", dir)
	}
}

// ---- Disk cache (WriteDiskCache / LoadDiskCache) ----------------------------

func TestWriteAndLoadDiskCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = filepath.Join(dir, ".skipper-cache.json")
	t.Cleanup(func() { DiskCacheFile = origFile })

	payload := []byte(`{"some/test.go > testfoo":null}`)
	if err := WriteDiskCache(payload); err != nil {
		t.Fatalf("WriteDiskCache: %v", err)
	}

	got, err := LoadDiskCache(300 * time.Second)
	if err != nil {
		t.Fatalf("LoadDiskCache: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("got %q, want %q", string(got), string(payload))
	}
}

func TestLoadDiskCache_MissingFile(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = filepath.Join(dir, "nonexistent.json")
	t.Cleanup(func() { DiskCacheFile = origFile })

	if _, err := LoadDiskCache(300 * time.Second); err == nil {
		t.Error("expected error for missing disk cache file")
	}
}

func TestLoadDiskCache_ExpiredCache(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = filepath.Join(dir, ".skipper-cache.json")
	t.Cleanup(func() { DiskCacheFile = origFile })

	// Write a cache entry with a timestamp far in the past.
	stale := `{"written_at":"2000-01-01T00:00:00Z","data":{}}`
	if err := os.WriteFile(DiskCacheFile, []byte(stale), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadDiskCache(300 * time.Second); err == nil {
		t.Error("expected error for expired disk cache")
	}
}

func TestLoadDiskCache_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	origFile := DiskCacheFile
	DiskCacheFile = filepath.Join(dir, ".skipper-cache.json")
	t.Cleanup(func() { DiskCacheFile = origFile })

	if err := os.WriteFile(DiskCacheFile, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadDiskCache(300 * time.Second); err == nil {
		t.Error("expected error for malformed disk cache JSON")
	}
}

// writeJSON is a helper that writes a JSON-encoded value to a file in dir.
func writeJSON(t *testing.T, dir, name string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
}
