package ginkgo

import (
	"github.com/get-skipper/skipper-go/core"
	"github.com/onsi/ginkgo/v2/types"
)

// testIDFromReport builds a Skipper test ID from a Ginkgo SpecReport.
//
// The test ID format mirrors the Ginkgo spec hierarchy:
//
//	"path/to/spec_test.go > Describe block > Context block > It text"
func testIDFromReport(report types.SpecReport) string {
	titleParts := append(
		append([]string{}, report.ContainerHierarchyTexts...),
		report.LeafNodeText,
	)
	return core.BuildTestID(report.FileName(), titleParts)
}
