package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/api/sheets/v4"
)

// SheetsWriter reconciles a Google Spreadsheet with a set of discovered test IDs.
type SheetsWriter struct {
	config SkipperConfig
	client *SheetsClient
}

// NewSheetsWriter creates a new SheetsWriter.
func NewSheetsWriter(config SkipperConfig) *SheetsWriter {
	return &SheetsWriter{config: config, client: NewSheetsClient(config)}
}

// Sync reconciles the primary sheet with discoveredIDs:
//   - Rows whose test ID is no longer discovered are deleted.
//   - Newly discovered test IDs are appended with an empty disabledUntil.
func (w *SheetsWriter) Sync(ctx context.Context, discoveredIDs []string) error {
	Logf("syncing %d discovered test IDs", len(discoveredIDs))

	result, err := w.client.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("skipper: sync fetch failed: %w", err)
	}

	primary := result.Primary
	svc := result.Service

	testIDIdx := indexOf(primary.Header, w.config.testIDColumn())
	if testIDIdx < 0 {
		return fmt.Errorf("skipper: column %q not found in sheet %q", w.config.testIDColumn(), primary.SheetName)
	}

	// Build a map of normalized discovered IDs.
	discoveredSet := make(map[string]struct{}, len(discoveredIDs))
	for _, id := range discoveredIDs {
		discoveredSet[NormalizeTestID(id)] = struct{}{}
	}

	// Derive the set of file paths "owned" by this sync.
	// Each valid test ID has the form "{file} > {title parts...}".
	// We only touch rows belonging to these files so that syncs from
	// different packages (running as separate test binaries) do not
	// overwrite each other's rows.
	ownedFiles := make(map[string]struct{}, len(discoveredIDs))
	// ownedBases holds the base filename of each owned file so that stale
	// entries written with bare filenames (e.g. "skipper_test.go" instead of
	// "ginkgo/skipper_test.go") are also cleaned up during a sync.
	ownedBases := make(map[string]struct{}, len(discoveredIDs))
	for _, id := range discoveredIDs {
		normalized := NormalizeTestID(id)
		if idx := strings.Index(normalized, " > "); idx >= 0 {
			file := normalized[:idx]
			ownedFiles[file] = struct{}{}
			if i := strings.LastIndex(file, "/"); i >= 0 {
				ownedBases[file[i+1:]] = struct{}{}
			} else {
				ownedBases[file] = struct{}{}
			}
		}
	}

	// Build a map of normalized existing IDs → original test ID.
	existingMap := make(map[string]string, len(primary.Entries))
	for _, e := range primary.Entries {
		existingMap[NormalizeTestID(e.TestID)] = e.TestID
	}

	// Identify rows to delete. Row indices are 0-based; row 0 is the header.
	// We delete a row when:
	//   (a) it is malformed (no " > " separator) — leftover from a previous bug, or
	//   (b) its file path is owned by this sync and it is no longer discovered.
	var rowsToDelete []int
	for i, e := range primary.Entries {
		normalized := NormalizeTestID(e.TestID)
		idx := strings.Index(normalized, " > ")
		if idx < 0 {
			// Malformed row — clean it up unconditionally.
			rowsToDelete = append(rowsToDelete, i+1)
			continue
		}
		file := normalized[:idx]
		_, owned := ownedFiles[file]
		if !owned && !strings.Contains(file, "/") {
			// Bare filename (no directory) — check if it's a stale entry for
			// one of our files that was written before path anchoring was fixed.
			_, owned = ownedBases[file]
		}
		if !owned {
			continue // belongs to another package's sync — leave it alone
		}
		if _, ok := discoveredSet[normalized]; !ok {
			rowsToDelete = append(rowsToDelete, i+1) // +1 for header row
		}
	}

	// Identify test IDs to add (discovered but not in spreadsheet).
	var toAdd []string
	for _, id := range discoveredIDs {
		if _, ok := existingMap[NormalizeTestID(id)]; !ok {
			toAdd = append(toAdd, id)
		}
	}

	// Delete rows in descending order to avoid index shifting.
	if len(rowsToDelete) > 0 {
		sort.Sort(sort.Reverse(sort.IntSlice(rowsToDelete)))
		var reqs []*sheets.Request
		for _, rowIdx := range rowsToDelete {
			reqs = append(reqs, &sheets.Request{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    primary.SheetID,
						Dimension:  "ROWS",
						StartIndex: int64(rowIdx),
						EndIndex:   int64(rowIdx + 1),
					},
				},
			})
		}
		_, err := svc.Spreadsheets.BatchUpdate(w.config.SpreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: reqs,
		}).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("skipper: batch delete failed: %w", err)
		}
		Logf("deleted %d rows from spreadsheet", len(rowsToDelete))
	}

	// Append new rows.
	if len(toAdd) > 0 {
		headerLen := len(primary.Header)
		var values [][]any
		for _, id := range toAdd {
			row := make([]any, headerLen)
			for i := range row {
				row[i] = ""
			}
			row[testIDIdx] = id
			values = append(values, row)
		}
		_, err := svc.Spreadsheets.Values.Append(
			w.config.SpreadsheetID,
			primary.SheetName,
			&sheets.ValueRange{Values: values},
		).ValueInputOption("RAW").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("skipper: append failed: %w", err)
		}
		Logf("appended %d new test IDs to spreadsheet", len(toAdd))
	}

	return nil
}
