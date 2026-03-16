package testing

import (
	"strings"
	"testing"
)

func TestSplitTestName(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "TestLogin",
			want:  []string{"TestLogin"},
		},
		{
			input: "TestCheckout/with_valid_card",
			want:  []string{"TestCheckout", "with valid card"},
		},
		{
			input: "TestAuth/Login/as_admin",
			want:  []string{"TestAuth", "Login", "as admin"},
		},
		{
			// Top-level name: underscores are part of the function name, not space replacements.
			input: "TestMultiple_Underscores__Here",
			want:  []string{"TestMultiple_Underscores__Here"},
		},
		{
			// Top-level name with underscore, plus a subtest where underscores ARE space replacements.
			input: "TestFoo_Bar/sub_test",
			want:  []string{"TestFoo_Bar", "sub test"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := splitTestName(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("splitTestName(%q) = %v (len %d), want %v (len %d)",
					tc.input, got, len(got), tc.want, len(tc.want))
			}
			for i, part := range got {
				if part != tc.want[i] {
					t.Errorf("part[%d] = %q, want %q", i, part, tc.want[i])
				}
			}
		})
	}
}

func TestCallerFile_ReturnsTestFile(t *testing.T) {
	file := callerFile()
	if !strings.HasSuffix(file, "_test.go") {
		t.Errorf("callerFile() = %q, want a path ending in _test.go", file)
	}
	if file == "unknown" {
		t.Errorf("callerFile() returned %q", file)
	}
}
