package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

// NormalizeTestID lowercases a test ID and collapses multiple whitespace
// characters into a single space. Used for case-insensitive matching.
func NormalizeTestID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.ToLower(id)
	id = whitespaceRe.ReplaceAllString(id, " ")
	return id
}

// BuildTestID constructs a test ID in the format:
//
//	"{relative/path/to/file_test.go} > {titleParts[0]} > ... > {titleParts[n]}"
//
// If filePath is absolute, it is made relative to the current working directory.
// Path separators are normalized to forward slashes.
func BuildTestID(filePath string, titleParts []string) string {
	relPath := toRelativePath(filePath)
	parts := append([]string{relPath}, titleParts...)
	return strings.Join(parts, " > ")
}

func toRelativePath(filePath string) string {
	if filepath.IsAbs(filePath) {
		// Prefer the project root (go.work or go.mod) over the working
		// directory so that paths are stable across packages in the same
		// workspace. E.g. "testing/skipper_test.go" rather than just
		// "skipper_test.go" when each package binary runs in its own subdir.
		root := findProjectRoot(filePath)
		if root == "" {
			root, _ = os.Getwd()
		}
		if rel, err := filepath.Rel(root, filePath); err == nil {
			filePath = rel
		}
	}
	// Normalize to forward slashes for cross-platform consistency.
	return strings.ReplaceAll(filePath, "\\", "/")
}

// findProjectRoot walks up the directory tree from filePath looking for a
// go.work file (workspace root) or, if absent, a go.mod file (module root).
// go.work takes priority so that paths are relative to the workspace root in
// multi-module setups.
func findProjectRoot(filePath string) string {
	dir := filepath.Dir(filePath)
	var modRoot string
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir // workspace root wins
		}
		if modRoot == "" {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				modRoot = dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}
	return modRoot
}

var testFuncRe = regexp.MustCompile(`(?m)^func (Test[A-Z][A-Za-z0-9_]*)\(`)

// ScanPackageTests scans the *_test.go files in the current working directory
// and returns test IDs for all top-level TestXxx functions found.
// Intended for use in TestMain / suite setup to pre-discover tests that do not
// explicitly call SkipIfDisabled, so they appear in the spreadsheet in sync mode.
func ScanPackageTests() []string {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*_test.go"))
	var ids []string
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, m := range testFuncRe.FindAllSubmatch(content, -1) {
			name := string(m[1])
			if name == "TestMain" {
				continue
			}
			ids = append(ids, BuildTestID(f, []string{name}))
		}
	}
	return ids
}
