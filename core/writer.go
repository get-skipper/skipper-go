package core

import (
	"context"
	"fmt"
	"sort"

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

	// Build a map of normalized existing IDs → original test ID.
	existingMap := make(map[string]string, len(primary.Entries))
	for _, e := range primary.Entries {
		existingMap[NormalizeTestID(e.TestID)] = e.TestID
	}

	// Identify rows to delete (in spreadsheet but not in discovered set).
	// Row indices in the Sheets API are 0-based; row 0 is the header.
	var rowsToDelete []int
	for i, e := range primary.Entries {
		if _, ok := discoveredSet[NormalizeTestID(e.TestID)]; !ok {
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
