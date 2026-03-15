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
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, filePath); err == nil {
				filePath = rel
			}
		}
	}
	// Normalize to forward slashes for cross-platform consistency.
	return filepath.ToSlash(filePath)
}
