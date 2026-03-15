package testify

import (
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
		{"AuthSuite", "AuthSuite"},  // no "Test" prefix → unchanged
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

func TestIsInternalFrame(t *testing.T) {
	internalFrames := []string{
		"/home/user/go/pkg/mod/github.com/stretchr/testify@v1.9.0/suite/suite.go",
		"/home/user/go/src/skipper-go/testify/suite.go",
		"/usr/local/go/src/testing/testing.go",
		"/usr/local/go/src/runtime/proc.go",
	}
	for _, f := range internalFrames {
		if !isInternalFrame(f) {
			t.Errorf("isInternalFrame(%q) = false, want true", f)
		}
	}

	externalFrames := []string{
		"/home/user/myproject/auth_suite_test.go",
		"/home/user/myproject/internal/payment_suite_test.go",
	}
	for _, f := range externalFrames {
		if isInternalFrame(f) {
			t.Errorf("isInternalFrame(%q) = true, want false", f)
		}
	}
}
