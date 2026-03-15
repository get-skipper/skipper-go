package core

import (
	"testing"
	"time"
)

// newResolverFromEntries is a test helper that builds a SkipperResolver
// directly from a list of TestEntry values, without fetching from Google Sheets.
func newResolverFromEntries(entries []TestEntry) *SkipperResolver {
	cache := make(map[string]*time.Time, len(entries))
	for _, e := range entries {
		cache[NormalizeTestID(e.TestID)] = e.DisabledUntil
	}
	return &SkipperResolver{cache: cache}
}

func future(days int) *time.Time {
	t := time.Now().AddDate(0, 0, days)
	return &t
}

func past(days int) *time.Time {
	t := time.Now().AddDate(0, 0, -days)
	return &t
}

func TestIsTestEnabled_UnknownTestIsEnabledByDefault(t *testing.T) {
	r := newResolverFromEntries(nil)
	if !r.IsTestEnabled("tests/auth_test.go > TestLogin") {
		t.Error("expected unknown test to be enabled (opt-out model)")
	}
}

func TestIsTestEnabled_NilDisabledUntilIsEnabled(t *testing.T) {
	r := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: nil},
	})
	if !r.IsTestEnabled("tests/auth_test.go > TestLogin") {
		t.Error("expected test with nil disabledUntil to be enabled")
	}
}

func TestIsTestEnabled_PastDateIsEnabled(t *testing.T) {
	r := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: past(1)},
	})
	if !r.IsTestEnabled("tests/auth_test.go > TestLogin") {
		t.Error("expected test with past disabledUntil to be enabled")
	}
}

func TestIsTestEnabled_FutureDateIsDisabled(t *testing.T) {
	r := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: future(30)},
	})
	if r.IsTestEnabled("tests/auth_test.go > TestLogin") {
		t.Error("expected test with future disabledUntil to be disabled")
	}
}

func TestIsTestEnabled_CaseInsensitiveMatching(t *testing.T) {
	r := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: future(30)},
	})
	// ID stored in cache is normalized to lowercase. The query must also normalize.
	if r.IsTestEnabled("TESTS/AUTH_TEST.GO > TESTLOGIN") {
		t.Error("expected case-insensitive matching to find the disabled test")
	}
}

func TestIsTestEnabled_WhitespaceNormalization(t *testing.T) {
	r := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: future(30)},
	})
	if r.IsTestEnabled("tests/auth_test.go  >  TestLogin") {
		t.Error("expected whitespace-normalized matching to find the disabled test")
	}
}

func TestGetDisabledUntil_ReturnsNilForUnknownTest(t *testing.T) {
	r := newResolverFromEntries(nil)
	if r.GetDisabledUntil("tests/auth_test.go > TestLogin") != nil {
		t.Error("expected nil for unknown test")
	}
}

func TestGetDisabledUntil_ReturnsFutureDate(t *testing.T) {
	expected := future(30)
	r := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: expected},
	})
	got := r.GetDisabledUntil("tests/auth_test.go > TestLogin")
	if got == nil {
		t.Fatal("expected non-nil date")
	}
	if !got.Equal(*expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestMarshalCache_RoundTrip(t *testing.T) {
	futureDate := future(10)
	original := newResolverFromEntries([]TestEntry{
		{TestID: "tests/auth_test.go > TestLogin", DisabledUntil: futureDate},
		{TestID: "tests/auth_test.go > TestLogout", DisabledUntil: nil},
		{TestID: "tests/auth_test.go > TestRegister", DisabledUntil: past(5)},
	})

	data, err := original.MarshalCache()
	if err != nil {
		t.Fatalf("MarshalCache: %v", err)
	}

	restored, err := FromMarshaledCache(data)
	if err != nil {
		t.Fatalf("FromMarshaledCache: %v", err)
	}

	cases := []struct {
		testID  string
		enabled bool
	}{
		{"tests/auth_test.go > TestLogin", false},
		{"tests/auth_test.go > TestLogout", true},
		{"tests/auth_test.go > TestRegister", true},
		{"tests/auth_test.go > TestUnknown", true},
	}
	for _, c := range cases {
		if got := restored.IsTestEnabled(c.testID); got != c.enabled {
			t.Errorf("IsTestEnabled(%q) = %v, want %v", c.testID, got, c.enabled)
		}
	}
}

func TestFromMarshaledCache_InvalidJSON(t *testing.T) {
	_, err := FromMarshaledCache([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFromMarshaledCache_InvalidDate(t *testing.T) {
	_, err := FromMarshaledCache([]byte(`{"some-test-id": "not-a-date"}`))
	if err == nil {
		t.Error("expected error for invalid date in cache")
	}
}
