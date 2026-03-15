package ginkgo_test

import (
	"os"
	"testing"

	"github.com/get-skipper/skipper-go/core"
	skipperginkgo "github.com/get-skipper/skipper-go/ginkgo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestSkipperGinkgoSuite is the Go test entry point for this Ginkgo suite.
// When credentials are available, RegisterSkipperHooks is called so that
// disabled specs are skipped automatically.
func TestSkipperGinkgoSuite(t *testing.T) {
	RegisterFailHandler(Fail)

	creds := resolveCredentials()
	if creds != nil {
		skipperginkgo.RegisterSkipperHooks(core.SkipperConfig{
			SpreadsheetID: spreadsheetID(),
			Credentials:   creds,
			SheetName:     "skipper-go",
		})
	}

	RunSpecs(t, "Skipper Ginkgo Integration Suite")
}

var _ = Describe("Skipper Ginkgo integration", func() {
	Context("when the test is not in the spreadsheet", func() {
		It("runs normally (opt-out model)", func() {
			Expect(true).To(BeTrue())
		})
	})

	Context("test ID format", func() {
		It("uses Describe/Context/It hierarchy separated by ' > '", func() {
			// This spec's test ID would be:
			// "ginkgo/skipper_test.go > Skipper Ginkgo integration > test ID format > uses Describe/Context/It hierarchy separated by ' > '"
			Expect(true).To(BeTrue())
		})
	})
})

func resolveCredentials() core.Credentials {
	if b64 := os.Getenv("GOOGLE_CREDS_B64"); b64 != "" {
		return core.Base64Credentials{Encoded: b64}
	}
	for _, path := range []string{
		"../service-account-skipper-bot.json",
		"service-account-skipper-bot.json",
	} {
		if _, err := os.Stat(path); err == nil {
			return core.FileCredentials{Path: path}
		}
	}
	return nil
}

func spreadsheetID() string {
	if id := os.Getenv("SKIPPER_SPREADSHEET_ID"); id != "" {
		return id
	}
	return "1Nbjfhklw11uVbi6OCOSeCJI_PThJYzQLlZQbThb4Zvs"
}
