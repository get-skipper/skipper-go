package core

import (
	"testing"
)

func TestNormalizeTestID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "tests/auth_test.go > TestLogin",
			want:  "tests/auth_test.go > testlogin",
		},
		{
			name:  "uppercase to lowercase",
			input: "Tests/Auth_Test.go > TestLogin",
			want:  "tests/auth_test.go > testlogin",
		},
		{
			name:  "collapses multiple spaces",
			input: "tests/auth_test.go  >  TestLogin",
			want:  "tests/auth_test.go > testlogin",
		},
		{
			name:  "collapses tabs",
			input: "tests/auth_test.go\t>\tTestLogin",
			want:  "tests/auth_test.go > testlogin",
		},
		{
			name:  "trims leading and trailing whitespace",
			input: "  tests/auth_test.go > TestLogin  ",
			want:  "tests/auth_test.go > testlogin",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   ",
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeTestID(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeTestID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildTestID(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		titleParts []string
		want       string
	}{
		{
			name:       "single title part",
			filePath:   "tests/auth_test.go",
			titleParts: []string{"TestLogin"},
			want:       "tests/auth_test.go > TestLogin",
		},
		{
			name:       "multiple title parts",
			filePath:   "tests/auth_test.go",
			titleParts: []string{"AuthSuite", "TestLogin"},
			want:       "tests/auth_test.go > AuthSuite > TestLogin",
		},
		{
			name:       "deep hierarchy",
			filePath:   "tests/auth_test.go",
			titleParts: []string{"Auth", "Login", "with valid credentials"},
			want:       "tests/auth_test.go > Auth > Login > with valid credentials",
		},
		{
			name:       "no title parts",
			filePath:   "tests/auth_test.go",
			titleParts: []string{},
			want:       "tests/auth_test.go",
		},
		{
			name:       "backslash path normalized to forward slash",
			filePath:   `tests\auth_test.go`,
			titleParts: []string{"TestLogin"},
			want:       "tests/auth_test.go > TestLogin",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildTestID(tc.filePath, tc.titleParts)
			if got != tc.want {
				t.Errorf("BuildTestID(%q, %v) = %q, want %q", tc.filePath, tc.titleParts, got, tc.want)
			}
		})
	}
}
