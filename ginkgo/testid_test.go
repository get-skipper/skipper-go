package ginkgo

import (
	"testing"

	"github.com/onsi/ginkgo/v2/types"
)

func TestTestIDFromReport(t *testing.T) {
	tests := []struct {
		name   string
		report types.SpecReport
		want   string
	}{
		{
			name: "single describe + it",
			report: types.SpecReport{
				ContainerHierarchyTexts: []string{"Auth"},
				LeafNodeText:            "can log in",
				LeafNodeLocation:        types.CodeLocation{FileName: "tests/auth_test.go"},
			},
			want: "tests/auth_test.go > Auth > can log in",
		},
		{
			name: "nested describe/context + it",
			report: types.SpecReport{
				ContainerHierarchyTexts: []string{"Auth", "Login"},
				LeafNodeText:            "with valid credentials",
				LeafNodeLocation:        types.CodeLocation{FileName: "tests/auth_test.go"},
			},
			want: "tests/auth_test.go > Auth > Login > with valid credentials",
		},
		{
			name: "no container hierarchy",
			report: types.SpecReport{
				ContainerHierarchyTexts: []string{},
				LeafNodeText:            "does something",
				LeafNodeLocation:        types.CodeLocation{FileName: "tests/simple_test.go"},
			},
			want: "tests/simple_test.go > does something",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := testIDFromReport(tc.report)
			if got != tc.want {
				t.Errorf("testIDFromReport() = %q, want %q", got, tc.want)
			}
		})
	}
}
