# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] – 2026-04-09

### Fixed

- **Strict `disabledUntil` date parsing**: replaced the lenient multi-format
  parser with a strict `YYYY-MM-DD`-only implementation. Partial dates, locale
  formats, and datetime strings are now rejected immediately with a row-numbered
  error instead of being silently ignored.
- **Timezone-consistent expiry**: dates are now parsed in UTC via
  `time.ParseInLocation` and stored as midnight UTC of the day *after* the given
  date (e.g. `2026-04-01` expires at `2026-04-02T00:00:00Z`). All CI runners
  reach the same expiry instant regardless of their local timezone.
- Empty or whitespace `disabledUntil` values continue to be treated as "not
  disabled" (no error).

## [1.1.0] – 2026-03-28

### Added

- **`SKIPPER_FAIL_OPEN`** (default `true`): when the Google Sheets API is
  unreachable and no usable disk cache exists, `Initialize` returns `nil`
  instead of an error, allowing all tests to run unblocked.
- **`SKIPPER_CACHE_TTL`** (default `300` seconds): after every successful API
  fetch, the resolved cache is persisted to `.skipper-cache.json`. On the next
  `Initialize` call, if the API is unavailable the file is used as a fallback
  as long as it is younger than the configured TTL. Set to `0` to disable disk
  caching entirely.
- **`SKIPPER_SYNC_ALLOW_DELETE`** (default `false`): orphaned rows in the
  Google Sheet are no longer deleted automatically during a sync. Set to `true`
  to opt-in to the previous pruning behaviour and prevent accidental data loss.

## [1.0.0] – 2025-01-01

### Added

- Initial release: `core` module with `SkipperResolver`, `SheetsClient`,
  `SheetsWriter`, and `CacheManager`.
- Adapter modules for the standard `testing` package, `testify/suite`, and
  Ginkgo v2.
- `SKIPPER_MODE` env var (`read-only` / `sync`).
- `SKIPPER_DEBUG` env var for verbose logging.
- Reference-sheet support for shared disabled-test lists.
- Test ID anchoring to workspace/module root via `ScanPackageTests`.

[Unreleased]: https://github.com/get-skipper/skipper-go/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/get-skipper/skipper-go/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/get-skipper/skipper-go/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/get-skipper/skipper-go/releases/tag/v1.0.0
