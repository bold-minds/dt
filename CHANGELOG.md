# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] — 2026-04-05

Canonical API layer over the inherited goby surface, plus two rounds of
code review. Every v0.1.0 entry point continues to work — the new names
are the preferred entry points and the old names are kept as thin
wrappers in `legacy.go`.

### Added
- **`New(value any, opts ...Option) string`** — canonical parse-and-format entry point. `NewDatetime` is now a one-line wrapper that calls `New`.
- **`Parse(value any) time.Time`** — canonical parser returning `time.Time` with zero on failure.
- **`ParseAny(value any) (time.Time, error)`** — parser with explicit error return for callers that need to distinguish empty input, unparseable strings, unsupported types, out-of-range timestamps, and zero-valued `time.Time`.
- **`Format` / `Option` / `ParseResult` types** — shorter canonical aliases for `DatetimeFormat` / `DatetimeOption` / `DatetimeParseResult`. The long-form names remain as aliases in `legacy.go`.
- Fuzz tests for the format detector and parser covering adversarial input.

### Changed
- Two rounds of code review resolved a combined ~34 items across correctness, API naming, test coverage, documentation, and tooling. Highlights:
  - Timezone handling for `DateOnly()` no longer rolls the date across offset boundaries (BUG-2 regression test added for `Pacific/Kiritimati` at +14:00).
  - `ISO()` format now uses `time.RFC3339` and preserves the offset instead of forcing UTC with a literal `Z` (BUG-24).
  - `ToTimezone(nil)` is now a no-op instead of panicking (BUG-13).
  - `DateOnly` + `TimeOnly` in the same call is now explicitly rejected as mutually exclusive (BUG-10).
  - Custom pattern tokenizer rewritten to avoid the cascading `ReplaceAll` bugs that mangled literal words like "Month".
  - `ParseAny` error messages categorized into distinct strings for nil, unparseable, unsupported-type, out-of-range, and zero-time cases so callers can dispatch on them.

### Fixed
- NaN/Inf handling in numeric timestamp paths.
- Unreachable layout dead code in the parser.
- Various package-doc and comment corrections.

## [0.1.0] — Initial release

### Added
- `NewDatetime(value any, opts ...DatetimeOption) string` — parse-and-format pipeline accepting `time.Time`, Unix timestamp (int/int64/float64), or a datetime string with auto-detected format.
- `ToDatetime(value string) time.Time` — parse a string to `time.Time`, returning zero time on failure.
- `ParseDatetime(value any, expectedFormat string, customPattern string) (*DatetimeParseResult, error)` — parse and return a result struct with format metadata.
- `DetectDatetimeFormat(input string) string` — heuristically detect the format of a datetime string (`iso`, `iso-tz`, `unix`, `date`, `custom`).
- `FormatDatetime(unixMillis int64, format string, customPattern string) (string, error)` — format a Unix millisecond timestamp using a named format or custom pattern.
- `IsValidDatetimeFormat(format string) bool` — validate a format name.
- `ParseTimeString(s string) (time.Time, error)` — thin wrapper around `ToDatetime` that returns an explicit error on failure.
- Format constructors: `ISO`, `Unix`, `Custom(pattern)`.
- Option constructors: `DateOnly`, `TimeOnly`, `ToTimezone(*time.Location)`, `FormatAs(DatetimeFormat)`.
- Supported parse layouts: ISO 8601 (UTC, with-timezone, with-milliseconds), US and European slash-dot-dash dates, natural-language month-name formats (`Jan 2 2006`, `January 2, 2006`, `2 Jan 2006`), short-year variants, time-only (`15:04:05`, `15:04`), and Unix timestamps in seconds or milliseconds.
- Custom pattern format using `YYYY`, `MM`, `DD`, `HH`, `mm`, `ss`, plus `DDD` for ordinal day (`1st`, `2nd`, `3rd`, ...).
- Length-gated heuristic pre-filter to avoid attempting layout parses on non-datetime strings.
- Unix timestamp range validation rejecting values below 0 or above 9999999999999 (year ~2286).
- Zero external dependencies; benchmarks for every path.

### Requires
- Go 1.21 or later
