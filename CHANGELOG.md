# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
