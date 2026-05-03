package core

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

// Report contains metrics about disabled tests and quarantine debt.
type Report struct {
	DisabledCount       int       `json:"disabled_count"`
	DueThisWeek         []string  `json:"due_this_week"`
	ReenabledCount      int       `json:"reenabled_count"`
	QuarantineDaysDebt  int       `json:"quarantine_days_debt"`
	GeneratedAt         time.Time `json:"generated_at"`
}

// GenerateReport generates a quarantine report from the current resolver state.
// It calculates:
//   - disabled_count: number of tests currently disabled (disabledUntil is in future)
//   - due_this_week: test IDs with disabledUntil dates within next 7 days
//   - reenabled_count: tests that were disabled but are now enabled (disabledUntil in past)
//   - quarantine_days_debt: sum of days between disabledUntil and now for disabled tests
func GenerateReport(resolver *SkipperResolver) *Report {
	if resolver == nil {
		return &Report{GeneratedAt: time.Now().UTC()}
	}

	now := time.Now().UTC()
	weekFromNow := now.AddDate(0, 0, 7)

	disabledCount := 0
	reenabledCount := 0
	dueThisWeek := []string{}
	quarantineDaysDebt := 0

	for testID, disabledUntil := range resolver.cache {
		if disabledUntil == nil {
			continue
		}

		if now.Before(*disabledUntil) {
			// Test is currently disabled
			disabledCount++

			// Check if due this week
			if disabledUntil.Before(weekFromNow) || disabledUntil.Equal(weekFromNow) {
				dueThisWeek = append(dueThisWeek, testID)
			}

			// Add to debt: days from now until disabledUntil
			days := int(math.Ceil(disabledUntil.Sub(now).Hours() / 24))
			quarantineDaysDebt += days
		} else {
			// Test is reenabled (was disabled but date passed)
			reenabledCount++
		}
	}

	return &Report{
		DisabledCount:      disabledCount,
		DueThisWeek:        dueThisWeek,
		ReenabledCount:     reenabledCount,
		QuarantineDaysDebt: quarantineDaysDebt,
		GeneratedAt:        now,
	}
}

// WriteReport writes the report to both GitHub Actions summary and skipper-report.json.
func WriteReport(report *Report) error {
	if report == nil {
		return fmt.Errorf("report is nil")
	}

	// Write to skipper-report.json
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal report: %w", err)
	}

	if err := os.WriteFile("skipper-report.json", data, 0644); err != nil {
		Warn(fmt.Sprintf("could not write skipper-report.json: %v", err))
	} else {
		Logf("wrote skipper-report.json")
	}

	// Write to GitHub Actions job summary if env var is set
	if summaryFile := os.Getenv("GITHUB_STEP_SUMMARY"); summaryFile != "" {
		markdown := formatReportMarkdown(report)
		if err := os.WriteFile(summaryFile, []byte(markdown), 0644); err != nil {
			Warn(fmt.Sprintf("could not write to GITHUB_STEP_SUMMARY: %v", err))
		} else {
			Logf("wrote quarantine report to GITHUB_STEP_SUMMARY")
		}
	}

	return nil
}

// formatReportMarkdown formats the report as GitHub Actions markdown.
func formatReportMarkdown(report *Report) string {
	markdown := "# Quarantine Report\n\n"

	markdown += fmt.Sprintf("| Metric | Count |\n|--------|-------|\n")
	markdown += fmt.Sprintf("| Disabled Tests | %d |\n", report.DisabledCount)
	markdown += fmt.Sprintf("| Reenabled Tests | %d |\n", report.ReenabledCount)
	markdown += fmt.Sprintf("| Quarantine Days Debt | %d |\n", report.QuarantineDaysDebt)

	if len(report.DueThisWeek) > 0 {
		markdown += "\n## Tests Due This Week\n\n"
		for _, testID := range report.DueThisWeek {
			markdown += fmt.Sprintf("- %s\n", testID)
		}
	}

	markdown += fmt.Sprintf("\n*Generated at: %s*\n", report.GeneratedAt.Format(time.RFC3339))

	return markdown
}
