package core

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiskCacheFile is the name of the persistent on-disk cache written after a
// successful API fetch. Tests may override this variable to use a temp path.
var DiskCacheFile = ".skipper-cache.json"

// diskCacheEntry is the JSON structure persisted to DiskCacheFile.
type diskCacheEntry struct {
	WrittenAt time.Time       `json:"written_at"`
	Data      json.RawMessage `json:"data"`
}

// WriteDiskCache serialises resolver data together with a timestamp and writes
// it to DiskCacheFile. The file is only readable by the current user (0o600).
// HTML escaping is disabled so that test-ID separators (e.g. " > ") are stored
// verbatim rather than being encoded as "\u003e".
func WriteDiskCache(data []byte) error {
	entry := diskCacheEntry{
		WrittenAt: time.Now().UTC(),
		Data:      json.RawMessage(data),
	}
	f, err := os.OpenFile(DiskCacheFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("skipper: cannot write disk cache: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("skipper: cannot marshal disk cache: %w", err)
	}
	return nil
}

// LoadDiskCache reads DiskCacheFile and returns the resolver data if the cache
// was written within ttl. Returns an error if the file is missing, unreadable,
// malformed, or expired.
func LoadDiskCache(ttl time.Duration) ([]byte, error) {
	b, err := os.ReadFile(DiskCacheFile)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot read disk cache: %w", err)
	}
	var entry diskCacheEntry
	if err := json.Unmarshal(b, &entry); err != nil {
		return nil, fmt.Errorf("skipper: malformed disk cache: %w", err)
	}
	if time.Since(entry.WrittenAt) > ttl {
		return nil, fmt.Errorf("skipper: disk cache expired (age %s > ttl %s)", time.Since(entry.WrittenAt).Round(time.Second), ttl)
	}
	return entry.Data, nil
}

// CacheManager manages the temporary directory used to share resolver state
// between the main test process and any parallel worker processes.
type CacheManager struct{}

const cacheFileName = "cache.json"

// WriteResolverCache writes serialized resolver data to a new temp directory
// and returns the directory path. Set SKIPPER_CACHE_FILE to dir/cache.json
// so worker processes can rehydrate the resolver.
func (CacheManager) WriteResolverCache(data []byte) (string, error) {
	dir, err := os.MkdirTemp("", "skipper-*")
	if err != nil {
		return "", fmt.Errorf("skipper: cannot create temp dir: %w", err)
	}
	path := filepath.Join(dir, cacheFileName)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("skipper: cannot write cache file: %w", err)
	}
	Logf("wrote resolver cache to %s", path)
	return dir, nil
}

// ReadResolverCache reads the serialized resolver data from the given file path.
func (CacheManager) ReadResolverCache(cacheFile string) ([]byte, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot read cache file %q: %w", cacheFile, err)
	}
	return data, nil
}

// WriteDiscoveredIDs writes a list of discovered test IDs to an individual file
// inside dir. File names include the process ID and a random suffix to avoid
// collisions in parallel test environments.
func (CacheManager) WriteDiscoveredIDs(dir string, ids []string) error {
	name := fmt.Sprintf("%d-%d-%s.json",
		os.Getpid(),
		time.Now().UnixNano(),
		randomHex(4),
	)
	path := filepath.Join(dir, name)
	data, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("skipper: cannot marshal discovered IDs: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("skipper: cannot write discovered IDs to %q: %w", path, err)
	}
	return nil
}

// MergeDiscoveredIDs reads all per-process discovered ID files from dir,
// deduplicates them, and returns the combined list.
func (CacheManager) MergeDiscoveredIDs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot read discovered dir %q: %w", dir, err)
	}

	seen := make(map[string]struct{})
	var result []string

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == cacheFileName || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			Warn(fmt.Sprintf("cannot read discovered file %q: %v", entry.Name(), err))
			continue
		}
		var ids []string
		if err := json.Unmarshal(data, &ids); err != nil {
			Warn(fmt.Sprintf("cannot parse discovered file %q: %v", entry.Name(), err))
			continue
		}
		for _, id := range ids {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				result = append(result, id)
			}
		}
	}
	return result, nil
}

// Cleanup removes the temp directory and all its contents.
func (CacheManager) Cleanup(dir string) error {
	return os.RemoveAll(dir)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp bytes if crypto/rand fails (extremely unlikely).
		copy(b, []byte(fmt.Sprintf("%x", time.Now().UnixNano())))
	}
	return fmt.Sprintf("%x", b)
}
