package testing

import (
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
			input: "TestMultiple_Underscores__Here",
			want:  []string{"TestMultiple Underscores  Here"},
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

func TestIsInternalFrame(t *testing.T) {
	internalFrames := []string{
		"/home/user/go/src/skipper-go/testing/skipper.go",
		"/usr/local/go/src/testing/testing.go",
		"/usr/local/go/src/runtime/proc.go",
	}
	for _, f := range internalFrames {
		if !isInternalFrame(f) {
			t.Errorf("isInternalFrame(%q) = false, want true", f)
		}
	}

	externalFrames := []string{
		"/home/user/myproject/auth_test.go",
		"/home/user/myproject/internal/service_test.go",
	}
	for _, f := range externalFrames {
		if isInternalFrame(f) {
			t.Errorf("isInternalFrame(%q) = true, want false", f)
		}
	}
}
