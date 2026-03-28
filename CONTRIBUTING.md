# Contributing to skipper-go

Thank you for your interest in contributing!

---

## Requirements

- Go 1.21+
- A Google Cloud service account (for integration testing against a real spreadsheet)

## Setup

```bash
git clone https://github.com/get-skipper/skipper-go.git
cd skipper-go
go work sync
```

## Running tests

```bash
make test
```

## Linting

```bash
make lint
```

Linting must pass before a pull request can be merged.

---

## Commit messages

All commits **must** follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): short description

[optional body]

[optional footer]
```

### Types

| Type | When to use |
|------|-------------|
| `feat` | A new feature or integration |
| `fix` | A bug fix |
| `docs` | Documentation changes only |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or updating tests |
| `chore` | Build process, dependency updates, tooling |

### Scopes

| Scope | Applies to |
|-------|------------|
| `core` | `core/` module |
| `testing` | `testing/` module |
| `testify` | `testify/` module |
| `ginkgo` | `ginkgo/` module |

### Examples

```
feat(testing): add SkipIfDisabled helper for subtest support
fix(core): correct date parsing for ISO-8601 with timezone offset
docs(ginkgo): document SynchronizedBeforeSuite parallel mode
refactor(core): extract SheetsClient authentication into separate method
test(core): add edge cases for NormalizeTestID
chore: update google.golang.org/api to v0.190.0
```

---

## Changelog

All notable changes are recorded in [CHANGELOG.md](CHANGELOG.md) following the
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format.

- Entries live under `## [Unreleased]` until a release is cut.
- Every change belongs to one of: `Added`, `Changed`, `Deprecated`, `Removed`,
  `Fixed`, `Security`.
- When a new version is released, rename `## [Unreleased]` to
  `## [x.y.z] – YYYY-MM-DD`, add a fresh empty `## [Unreleased]` section at
  the top, and append a comparison link at the bottom of the file.

---

## Pull requests

1. Fork the repository and create a branch:
   ```bash
   git checkout -b feat/my-feature
   ```

2. Make your changes. Ensure:
   - `make test` passes
   - `make lint` passes
   - New functionality has corresponding unit tests

3. Commit using Conventional Commits format (see above).

4. Open a pull request with a clear title and description.

---

## Project structure

```
core/       # Shared logic: SheetsClient, SkipperResolver, SheetsWriter, CacheManager, etc.
testing/    # Standard library testing integration
testify/    # testify/suite integration
ginkgo/     # Ginkgo v2 integration
```

Each framework integration should:
- Initialize the resolver (or rehydrate from cache) before tests run
- Skip disabled tests using the framework's native skip mechanism (`t.Skip()`, `ginkgo.Skip()`)
- Collect discovered test IDs for sync mode
- Call `SheetsWriter.Sync()` after all tests finish (sync mode only)

See existing integrations for reference patterns.
