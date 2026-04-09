package core

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

const sheetsScope = "https://www.googleapis.com/auth/spreadsheets"

// TestEntry represents a single row from the spreadsheet.
type TestEntry struct {
	TestID        string
	DisabledUntil *time.Time // nil means no date → test is enabled
	Notes         string
}

// SheetFetchResult holds the parsed data from a single sheet.
type SheetFetchResult struct {
	SheetName string
	SheetID   int64
	Header    []string
	Entries   []TestEntry
}

// FetchAllResult holds data from all sheets and the authenticated Sheets service
// so it can be reused by SheetsWriter without re-authenticating.
type FetchAllResult struct {
	Primary  SheetFetchResult
	Entries  []TestEntry // merged from primary + reference sheets (most restrictive wins)
	Service  *sheets.Service
}

// SheetsClient authenticates and fetches data from a Google Spreadsheet.
type SheetsClient struct {
	config SkipperConfig
}

// NewSheetsClient creates a new SheetsClient.
func NewSheetsClient(config SkipperConfig) *SheetsClient {
	return &SheetsClient{config: config}
}

// FetchAll fetches all relevant sheets and returns merged entries.
// The returned FetchAllResult.Service can be reused by SheetsWriter.
func (c *SheetsClient) FetchAll(ctx context.Context) (*FetchAllResult, error) {
	credJSON, err := c.config.Credentials.Resolve()
	if err != nil {
		return nil, err
	}

	creds, err := google.CredentialsFromJSON(ctx, credJSON, sheetsScope)
	if err != nil {
		return nil, fmt.Errorf("skipper: invalid credentials: %w", err)
	}

	svc, err := sheets.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot create Sheets service: %w", err)
	}

	spreadsheet, err := svc.Spreadsheets.Get(c.config.SpreadsheetID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot fetch spreadsheet: %w", err)
	}

	primaryName := c.config.SheetName
	if primaryName == "" && len(spreadsheet.Sheets) > 0 {
		primaryName = spreadsheet.Sheets[0].Properties.Title
	}

	primary, err := c.fetchSheet(ctx, svc, primaryName, spreadsheet)
	if err != nil {
		return nil, fmt.Errorf("skipper: cannot fetch primary sheet %q: %w", primaryName, err)
	}

	// Start with primary entries, then merge reference sheets.
	merged := mergeEntries(nil, primary.Entries)

	for _, refName := range c.config.ReferenceSheets {
		ref, err := c.fetchSheet(ctx, svc, refName, spreadsheet)
		if err != nil {
			Warn(fmt.Sprintf("cannot fetch reference sheet %q: %v", refName, err))
			continue
		}
		merged = mergeEntries(merged, ref.Entries)
	}

	return &FetchAllResult{
		Primary: primary,
		Entries: merged,
		Service: svc,
	}, nil
}

func (c *SheetsClient) fetchSheet(
	ctx context.Context,
	svc *sheets.Service,
	sheetName string,
	spreadsheet *sheets.Spreadsheet,
) (SheetFetchResult, error) {
	resp, err := svc.Spreadsheets.Values.
		Get(c.config.SpreadsheetID, sheetName).
		Context(ctx).
		Do()
	if err != nil {
		return SheetFetchResult{}, err
	}

	if len(resp.Values) == 0 {
		return SheetFetchResult{SheetName: sheetName}, nil
	}

	header := toStringSlice(resp.Values[0])
	testIDIdx := indexOf(header, c.config.testIDColumn())
	disabledUntilIdx := indexOf(header, c.config.disabledUntilColumn())
	notesIdx := indexOf(header, "notes")

	if testIDIdx < 0 {
		return SheetFetchResult{}, fmt.Errorf("column %q not found in sheet %q", c.config.testIDColumn(), sheetName)
	}

	sheetID := sheetIDByName(spreadsheet, sheetName)

	var entries []TestEntry
	for i, row := range resp.Values[1:] {
		rowNum := i + 2 // row 1 is the header; data rows start at 2
		if testIDIdx >= len(row) {
			continue
		}
		testID := cellString(row, testIDIdx)
		if testID == "" {
			continue
		}

		var disabledUntil *time.Time
		if disabledUntilIdx >= 0 {
			if raw := cellString(row, disabledUntilIdx); raw != "" {
				t, err := parseDisabledUntil(raw, rowNum)
				if err != nil {
					return SheetFetchResult{}, err
				}
				disabledUntil = &t
			}
		}

		var notes string
		if notesIdx >= 0 {
			notes = cellString(row, notesIdx)
		}

		entries = append(entries, TestEntry{
			TestID:        testID,
			DisabledUntil: disabledUntil,
			Notes:         notes,
		})
	}

	return SheetFetchResult{
		SheetName: sheetName,
		SheetID:   sheetID,
		Header:    header,
		Entries:   entries,
	}, nil
}

// mergeEntries merges new entries into the existing map using the most-restrictive
// (latest future) DisabledUntil date when the same test ID appears in multiple sheets.
func mergeEntries(existing []TestEntry, incoming []TestEntry) []TestEntry {
	type key = string
	idx := make(map[key]int, len(existing))
	result := make([]TestEntry, len(existing))
	copy(result, existing)
	for i, e := range result {
		idx[NormalizeTestID(e.TestID)] = i
	}

	for _, e := range incoming {
		nid := NormalizeTestID(e.TestID)
		if i, ok := idx[nid]; ok {
			// Keep the most restrictive (latest) DisabledUntil.
			existing := result[i].DisabledUntil
			incoming := e.DisabledUntil
			if moreRestrictive(incoming, existing) {
				result[i].DisabledUntil = incoming
			}
		} else {
			idx[nid] = len(result)
			result = append(result, e)
		}
	}
	return result
}

// moreRestrictive returns true if candidate is a later (more restrictive) date than current.
func moreRestrictive(candidate, current *time.Time) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	return candidate.After(*current)
}

// parseDisabledUntil parses a strict YYYY-MM-DD date string pinned to UTC.
// It returns the midnight UTC of the day AFTER the given date so that
// "disabled until 2026-04-01" means through end of that calendar day UTC.
// An empty or whitespace-only string returns a zero Time (not disabled).
// Any string not matching YYYY-MM-DD returns an error with the row number.
func parseDisabledUntil(raw string, rowNum int) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if !dateRe.MatchString(raw) {
		return time.Time{}, fmt.Errorf("[skipper] row %d: invalid disabledUntil %q — use YYYY-MM-DD", rowNum, raw)
	}
	t, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("[skipper] row %d: cannot parse date %q: %w", rowNum, raw, err)
	}
	return t.AddDate(0, 0, 1), nil
}

func indexOf(header []string, col string) int {
	for i, h := range header {
		if h == col {
			return i
		}
	}
	return -1
}

func cellString(row []any, idx int) string {
	if idx >= len(row) {
		return ""
	}
	if s, ok := row[idx].(string); ok {
		return s
	}
	return fmt.Sprintf("%v", row[idx])
}

func toStringSlice(row []any) []string {
	s := make([]string, len(row))
	for i, v := range row {
		if str, ok := v.(string); ok {
			s[i] = str
		} else {
			s[i] = fmt.Sprintf("%v", v)
		}
	}
	return s
}

func sheetIDByName(spreadsheet *sheets.Spreadsheet, name string) int64 {
	for _, s := range spreadsheet.Sheets {
		if s.Properties.Title == name {
			return s.Properties.SheetId
		}
	}
	return 0
}
