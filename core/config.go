package core

// SkipperConfig holds the complete configuration for a Skipper resolver.
type SkipperConfig struct {
	SpreadsheetID       string
	Credentials         Credentials
	SheetName           string   // empty → use first sheet
	ReferenceSheets     []string // read-only sheets merged into the resolver cache
	TestIDColumn        string   // default: "testId"
	DisabledUntilColumn string   // default: "disabledUntil"
}

func (c *SkipperConfig) testIDColumn() string {
	if c.TestIDColumn == "" {
		return "testId"
	}
	return c.TestIDColumn
}

func (c *SkipperConfig) disabledUntilColumn() string {
	if c.DisabledUntilColumn == "" {
		return "disabledUntil"
	}
	return c.DisabledUntilColumn
}
