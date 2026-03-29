# skipper-go

[![CI](https://github.com/get-skipper/skipper-go/actions/workflows/ci.yml/badge.svg)](https://github.com/get-skipper/skipper-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/get-skipper/skipper-go/core.svg)](https://pkg.go.dev/github.com/get-skipper/skipper-go/core)
[![License](https://img.shields.io/github/license/get-skipper/skipper-go)](LICENSE)

Test-gating for Go via Google Spreadsheet. Enable or disable tests without changing code — just update a date in a Google Sheet.

A Go port of [get-skipper/skipper](https://github.com/get-skipper/skipper), supporting the standard `testing` package, testify, and Ginkgo.

---

## How it works

A Google Spreadsheet stores test IDs with optional `disabledUntil` dates:

| testId | disabledUntil | notes |
|--------|---------------|-------|
| `tests/auth_test.go > TestLogin` | | |
| `tests/payment_test.go > TestCheckout` | `2099-12-31` | Flaky on CI |
| `tests/auth_test.go > Auth > Login > can log in` | `2026-06-01` | Under investigation |

- **Empty `disabledUntil`** → test runs normally
- **Past date** → test runs normally
- **Future date** → test is skipped automatically

Tests not listed in the spreadsheet **always run** (opt-out model).

---

## Packages

| Package | Import path | Framework |
|---------|------------|-----------|
| `core` | `github.com/get-skipper/skipper-go/core` | Shared core (resolver, client, cache) |
| `testing` | `github.com/get-skipper/skipper-go/testing` | Standard library `testing` |
| `testify` | `github.com/get-skipper/skipper-go/testify` | [testify/suite](https://github.com/stretchr/testify) |
| `ginkgo` | `github.com/get-skipper/skipper-go/ginkgo` | [Ginkgo v2](https://github.com/onsi/ginkgo) |

---

## Installation

```bash
# Standard testing package
go get github.com/get-skipper/skipper-go/testing

# testify
go get github.com/get-skipper/skipper-go/testify

# Ginkgo
go get github.com/get-skipper/skipper-go/ginkgo
```

---

## Google Sheets setup

1. Create a Google Spreadsheet with the following columns in row 1:
   - `testId`
   - `disabledUntil`
   - `notes` (optional)

2. Create a Google Cloud service account and download the JSON key file.

3. Share the spreadsheet with the service account's email (`client_email` in the JSON).

4. Note the spreadsheet ID from the URL:
   `https://docs.google.com/spreadsheets/d/YOUR_SPREADSHEET_ID/edit`

---

## Credentials

Three formats are accepted:

| Format | Type | Use case |
|--------|------|----------|
| File path | `core.FileCredentials{Path: "./service-account.json"}` | Local development |
| Base64 string | `core.Base64Credentials{Encoded: os.Getenv("GOOGLE_CREDS")}` | CI/CD env vars |
| Inline struct | `core.ServiceAccountCredentials{...}` | Programmatic use |

---

## Usage

### Standard `testing` package

```go
package mypackage_test

import (
    "os"
    "testing"

    "github.com/get-skipper/skipper-go/core"
    skippertest "github.com/get-skipper/skipper-go/testing"
)

func TestMain(m *testing.M) {
    s := &skippertest.SkipperTestMain{
        Config: core.SkipperConfig{
            SpreadsheetID: "your-spreadsheet-id",
            Credentials:   core.FileCredentials{Path: "./service-account.json"},
        },
    }
    os.Exit(s.Run(m))
}

func TestLogin(t *testing.T) {
    skippertest.SkipIfDisabled(t)
    // ... test body
}

func TestCheckout(t *testing.T) {
    skippertest.SkipIfDisabled(t)
    t.Run("with valid card", func(t *testing.T) {
        skippertest.SkipIfDisabled(t)
        // ... subtest body
    })
}
```

**Test ID format:** `tests/auth_test.go > TestLogin`
**Subtests:** `tests/auth_test.go > TestCheckout > with valid card`

---

### testify/suite

```go
package mypackage_test

import (
    "testing"

    "github.com/get-skipper/skipper-go/core"
    skippertestify "github.com/get-skipper/skipper-go/testify"
    "github.com/stretchr/testify/suite"
)

type AuthSuite struct {
    skippertestify.SkipperSuite
}

func (s *AuthSuite) TestLogin() {
    // SetupTest already called t.Skip() if disabled — just write the test
    s.Equal(200, 200)
}

func (s *AuthSuite) TestLogout() {
    s.True(true)
}

func TestAuthSuite(t *testing.T) {
    s := &AuthSuite{
        SkipperSuite: skippertestify.SkipperSuite{
            Config: core.SkipperConfig{
                SpreadsheetID: "your-spreadsheet-id",
                Credentials:   core.FileCredentials{Path: "./service-account.json"},
            },
        },
    }
    suite.Run(t, s)
}
```

**Test ID format:** `tests/auth_suite_test.go > AuthSuite > TestLogin`

---

### Ginkgo v2

```go
package mypackage_test

import (
    "testing"

    "github.com/get-skipper/skipper-go/core"
    skipperginkgo "github.com/get-skipper/skipper-go/ginkgo"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
    RegisterFailHandler(Fail)
    skipperginkgo.RegisterSkipperHooks(core.SkipperConfig{
        SpreadsheetID: "your-spreadsheet-id",
        Credentials:   core.FileCredentials{Path: "./service-account.json"},
    })
    RunSpecs(t, "Auth Suite")
}

var _ = Describe("Auth", func() {
    Context("Login", func() {
        It("can log in with valid credentials", func() {
            Expect(true).To(BeTrue())
        })
    })
})
```

**Test ID format:** `tests/auth_test.go > Auth > Login > can log in with valid credentials`

---

## Modes

| Mode | Env var | Behavior |
|------|---------|----------|
| `read-only` (default) | — | Fetch spreadsheet, skip disabled tests |
| `sync` | `SKIPPER_MODE=sync` | Same as above, then reconcile spreadsheet with discovered tests |

In sync mode, new tests are added to the spreadsheet. Deletion of orphaned rows requires `SKIPPER_SYNC_ALLOW_DELETE=true`.

---

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SKIPPER_MODE` | `read-only` | Set to `sync` to enable sync mode |
| `SKIPPER_DEBUG` | — | Any non-empty value enables verbose debug logging |
| `SKIPPER_FAIL_OPEN` | `true` | When `true`, `Initialize` returns `nil` instead of an error if the API is unreachable and no usable disk cache exists, so all tests are allowed to run rather than blocking CI/CD |
| `SKIPPER_CACHE_TTL` | `300` | Seconds to keep the on-disk cache (`.skipper-cache.json`) as a fallback when the API is unavailable. Set to `0` to disable caching |
| `SKIPPER_SYNC_ALLOW_DELETE` | `false` | In sync mode, orphaned rows are **not** deleted by default. Set to `true` to restore automatic pruning of tests that no longer exist |

---

## Test ID format

Test IDs follow the pattern: `{relative/path/to/file_test.go} > {title parts...}`

- Paths are relative to the working directory
- Title parts are separated by ` > `
- IDs are case-insensitive and whitespace-collapsed for matching

---

## Reference sheets

You can include additional read-only sheets to inherit disabled tests from:

```go
core.SkipperConfig{
    SpreadsheetID: "your-spreadsheet-id",
    Credentials:   core.FileCredentials{Path: "./service-account.json"},
    ReferenceSheets: []string{"GlobalDisabled", "FlakyTests"},
}
```

When the same test ID appears in multiple sheets, the most restrictive (latest) `disabledUntil` date wins.

---

## CI example

```yaml
- name: Run tests
  run: go test ./...
  env:
    GOOGLE_CREDS_B64: ${{ secrets.GOOGLE_CREDS_B64 }}
```

Use `Base64Credentials` to pass credentials via environment variable:

```go
core.Base64Credentials{Encoded: os.Getenv("GOOGLE_CREDS_B64")}
```
