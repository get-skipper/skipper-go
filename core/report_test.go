package core

import (
	"os"
	"testing"
	"time"
)

func TestGenerateReport(t *testing.T) {
	now := time.Now().UTC()

	resolver := &SkipperResolver{
		cache: map[string]*time.Time{
			"test1": ptrTime(now.AddDate(0, 0, 3)), // disabled, due in 3 days
			"test2": ptrTime(now.AddDate(0, 0, 10)), // disabled, due in 10 days
			"test3": ptrTime(now.AddDate(0, 0, -2)), // reenabled (past date)
			"test4": nil,                             // no date, enabled
		},
	}

	report := GenerateReport(resolver)

	if report == nil {
		t.Fatalf("GenerateReport returned nil")
	}

	if report.DisabledCount != 2 {
		t.Errorf("expected DisabledCount=2, got %d", report.DisabledCount)
	}

	if report.ReenabledCount != 1 {
		t.Errorf("expected ReenabledCount=1, got %d", report.ReenabledCount)
	}

	if len(report.DueThisWeek) != 1 {
		t.Errorf("expected 1 test due this week, got %d", len(report.DueThisWeek))
	}

	if report.DueThisWeek[0] != "test1" {
		t.Errorf("expected 'test1' in DueThisWeek, got %v", report.DueThisWeek)
	}

	// Check quarantine days debt: test1 (3 days) + test2 (10 days) = 13
	expectedDebt := 3 + 10
	if report.QuarantineDaysDebt != expectedDebt {
		t.Errorf("expected QuarantineDaysDebt=%d, got %d", expectedDebt, report.QuarantineDaysDebt)
	}
}

func TestGenerateReportNilResolver(t *testing.T) {
	report := GenerateReport(nil)

	if report == nil {
		t.Fatalf("GenerateReport(nil) returned nil, expected empty report")
	}

	if report.DisabledCount != 0 {
		t.Errorf("expected DisabledCount=0 for nil resolver, got %d", report.DisabledCount)
	}
}

func TestWriteReport(t *testing.T) {
	tmpDir := t.TempDir()
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get cwd: %v", err)
	}
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("could not change to temp dir: %v", err)
	}

	report := &Report{
		DisabledCount:      2,
		DueThisWeek:        []string{"test1", "test2"},
		ReenabledCount:     1,
		QuarantineDaysDebt: 10,
		GeneratedAt:        time.Now().UTC(),
	}

	if err := WriteReport(report); err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	// Check skipper-report.json was created
	data, err := os.ReadFile("skipper-report.json")
	if err != nil {
		t.Fatalf("skipper-report.json not found: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("skipper-report.json is empty")
	}
}

func TestWriteReportGitHubActionsIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	summaryFile := tmpDir + "/summary.txt"

	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get cwd: %v", err)
	}
	defer os.Chdir(originalCwd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("could not change to temp dir: %v", err)
	}

	os.Setenv("GITHUB_STEP_SUMMARY", summaryFile)
	defer os.Unsetenv("GITHUB_STEP_SUMMARY")

	report := &Report{
		DisabledCount:      2,
		DueThisWeek:        []string{"test1"},
		ReenabledCount:     1,
		QuarantineDaysDebt: 15,
		GeneratedAt:        time.Now().UTC(),
	}

	if err := WriteReport(report); err != nil {
		t.Fatalf("WriteReport failed: %v", err)
	}

	// Check summary file was created
	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("summary file not found: %v", err)
	}

	markdown := string(data)
	if len(markdown) == 0 {
		t.Errorf("summary file is empty")
	}

	if !contains(markdown, "Quarantine Report") {
		t.Errorf("summary missing title")
	}

	if !contains(markdown, "2") {
		t.Errorf("summary missing disabled count")
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
