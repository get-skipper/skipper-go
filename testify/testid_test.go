package testify

import (
	"strings"
	"testing"
)

func TestSplitSuiteName(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			// Standard suite: runner name + method name.
			input: "TestAuthSuite/TestLogin",
			want:  []string{"AuthSuite", "TestLogin"},
		},
		{
			// No slash: just the suite runner function name.
			input: "TestAuthSuite",
			want:  []string{"AuthSuite"},
		},
		{
			// Suite without "Test" prefix (unusual but handled).
			input: "AuthSuite/TestLogin",
			want:  []string{"AuthSuite", "TestLogin"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := splitSuiteName(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("splitSuiteName(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestStripTestPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"TestAuthSuite", "AuthSuite"},
		{"TestFoo", "Foo"},
		{"AuthSuite", "AuthSuite"}, // no "Test" prefix → unchanged
		{"Test", ""},               // "Test" alone → empty string
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := stripTestPrefix(tc.input)
			if got != tc.want {
				t.Errorf("stripTestPrefix(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCallerTestFile_ReturnsTestFile(t *testing.T) {
	file := callerTestFile()
	if !strings.HasSuffix(file, "_test.go") {
		t.Errorf("callerTestFile() = %q, want a path ending in _test.go", file)
	}
	if file == "unknown" {
		t.Errorf("callerTestFile() returned %q", file)
	}
}
