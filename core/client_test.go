package core

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		wantUTC string // expected UTC date portion, empty means error expected
	}{
		{input: "2099-12-31", wantUTC: "2099-12-31"},
		{input: "2099-12-31T23:59:59+00:00", wantUTC: "2099-12-31"},
		{input: "2099-12-31T23:59:59", wantUTC: "2099-12-31"},
		{input: "not-a-date", wantErr: true},
		{input: "", wantErr: true},
		{input: "31/12/2099", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseDate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("parseDate(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDate(%q) unexpected error: %v", tc.input, err)
			}
			gotDate := got.UTC().Format("2006-01-02")
			if gotDate != tc.wantUTC {
				t.Errorf("parseDate(%q) date = %q, want %q", tc.input, gotDate, tc.wantUTC)
			}
		})
	}
}

func TestMergeEntries_MostRestrictiveDateWins(t *testing.T) {
	later := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	primary := []TestEntry{
		{TestID: "tests/a.go > TestA", DisabledUntil: &earlier},
	}
	reference := []TestEntry{
		{TestID: "tests/a.go > TestA", DisabledUntil: &later},
	}

	merged := mergeEntries(primary, reference)
	if len(merged) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(merged))
	}
	if !merged[0].DisabledUntil.Equal(later) {
		t.Errorf("expected later date to win, got %v", merged[0].DisabledUntil)
	}
}

func TestMergeEntries_NilDoesNotOverrideExistingDate(t *testing.T) {
	future := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

	primary := []TestEntry{
		{TestID: "tests/a.go > TestA", DisabledUntil: &future},
	}
	reference := []TestEntry{
		{TestID: "tests/a.go > TestA", DisabledUntil: nil},
	}

	merged := mergeEntries(primary, reference)
	if len(merged) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(merged))
	}
	if merged[0].DisabledUntil == nil {
		t.Error("nil from reference should not override existing future date")
	}
}

func TestMergeEntries_NewEntryFromReference(t *testing.T) {
	future := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

	primary := []TestEntry{
		{TestID: "tests/a.go > TestA", DisabledUntil: nil},
	}
	reference := []TestEntry{
		{TestID: "tests/b.go > TestB", DisabledUntil: &future},
	}

	merged := mergeEntries(primary, reference)
	if len(merged) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(merged))
	}
}

func TestMergeEntries_CaseInsensitiveDedup(t *testing.T) {
	future := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

	primary := []TestEntry{
		{TestID: "tests/a.go > TestA", DisabledUntil: nil},
	}
	// Same test ID but with different case in reference sheet.
	reference := []TestEntry{
		{TestID: "TESTS/A.GO > TESTA", DisabledUntil: &future},
	}

	merged := mergeEntries(primary, reference)
	if len(merged) != 1 {
		t.Fatalf("expected 1 entry after case-insensitive dedup, got %d", len(merged))
	}
}

func TestMoreRestrictive(t *testing.T) {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	if moreRestrictive(nil, nil) {
		t.Error("nil candidate should not be more restrictive than nil current")
	}
	if moreRestrictive(nil, &future) {
		t.Error("nil candidate should not be more restrictive than non-nil current")
	}
	if !moreRestrictive(&future, nil) {
		t.Error("non-nil candidate should be more restrictive than nil current")
	}
	if moreRestrictive(&past, &future) {
		t.Error("earlier candidate should not be more restrictive than later current")
	}
	if !moreRestrictive(&future, &past) {
		t.Error("later candidate should be more restrictive than earlier current")
	}
}
