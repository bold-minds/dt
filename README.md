# dt

[![Go Reference](https://pkg.go.dev/badge/github.com/bold-minds/dt.svg)](https://pkg.go.dev/github.com/bold-minds/dt)
[![Build](https://img.shields.io/github/actions/workflow/status/bold-minds/dt/test.yaml?branch=main&label=tests)](https://github.com/bold-minds/dt/actions/workflows/test.yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bold-minds/dt)](go.mod)

**Datetime parsing and formatting for real-world inputs — auto-detect the format, accept `any`, output a string.**

Go's `time` package requires you to know the layout before parsing. That's fine for structured code, but it's friction when you're processing ingested data: user input, API payloads, log lines, config values. `dt` auto-detects the format and accepts `time.Time`, Unix timestamps, or strings interchangeably.

```go
// Parse whatever the caller passes, format as ISO 8601
s := dt.NewDatetime("Jan 2, 2026")                    // "2026-01-02T00:00:00Z"
s  = dt.NewDatetime(int64(1735776000))                // "2025-01-02T00:00:00Z"
s  = dt.NewDatetime(time.Now(), dt.FormatAs(dt.Unix())) // Unix millis string

// Convert to time.Time when you need a real value
t := dt.ToDatetime("2026-01-15T10:30:00Z")
```

## ✨ Why dt?

- 🎯 **Accepts `any`** — `time.Time`, `int64`, `int`, `float64`, or `string`. No "first convert to..." step.
- 🔎 **Format auto-detection** — drop in user input, get a parsed result. Handles ISO 8601, Unix timestamps, US/European dates, natural-language month names, ordinal days.
- 🛡️ **Never panics** — malformed input returns a zero `time.Time`, not a crash.
- 🔒 **Parser resilience** — length-gated heuristic filter avoids quadratic behavior on non-datetime input. Unix timestamp range validation rejects implausible values.
- 🪶 **Zero dependencies** — pure Go stdlib (`fmt`, `regexp`, `strconv`, `strings`, `time`).
- 🌍 **Timezone and format options** — convert to a target timezone, strip date or time components, choose between ISO / Unix / custom output patterns.

## 📦 Installation

```bash
go get github.com/bold-minds/dt
```

Requires Go 1.21 or later.

## 🎯 Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/bold-minds/dt"
)

func main() {
    // ISO in, ISO out (default)
    fmt.Println(dt.NewDatetime("2026-01-15T10:30:00Z"))

    // Natural-language in, ISO out
    fmt.Println(dt.NewDatetime("Jan 15, 2026"))

    // Unix seconds in, ISO out
    fmt.Println(dt.NewDatetime(int64(1736938200)))

    // ISO in, Unix millis out
    fmt.Println(dt.NewDatetime("2026-01-15T10:30:00Z", dt.FormatAs(dt.Unix())))

    // ISO in, custom format out
    fmt.Println(dt.NewDatetime("2026-01-15T10:30:00Z", dt.FormatAs(dt.Custom("YYYY-MM-DD"))))

    // With timezone conversion
    la, _ := time.LoadLocation("America/Los_Angeles")
    fmt.Println(dt.NewDatetime("2026-01-15T18:30:00Z", dt.ToTimezone(la)))

    // Strip the time component
    fmt.Println(dt.NewDatetime("2026-01-15T10:30:00Z", dt.DateOnly()))

    // Ordinal day in custom output: "Jan 15th, 2026"
    fmt.Println(dt.NewDatetime("2026-01-15T00:00:00Z",
        dt.FormatAs(dt.Custom("MMM DDD, YYYY"))))
}
```

## 📚 API

### Core

| Function | Purpose |
|---|---|
| `NewDatetime(value any, opts ...DatetimeOption) string` | Parse-and-format pipeline. Returns `""` on failure. |
| `ToDatetime(value string) time.Time` | Parse to `time.Time`. Returns zero time on failure. |
| `ParseDatetime(value any, expectedFormat, customPattern string) (*DatetimeParseResult, error)` | Parse with format metadata. |
| `DetectDatetimeFormat(input string) string` | Heuristic format detection. |
| `FormatDatetime(unixMillis int64, format, customPattern string) (string, error)` | Format a Unix milliseconds value. |
| `IsValidDatetimeFormat(format string) bool` | Format name validation. |
| `ParseTimeString(s string) (time.Time, error)` | Thin wrapper around `ToDatetime` returning an explicit error. |

### Output formats

| Constructor | Produces |
|---|---|
| `ISO()` | `2006-01-02T15:04:05Z` |
| `Unix()` | Unix milliseconds as a string |
| `Custom(pattern)` | Custom pattern — `YYYY`, `MM`, `DD`, `DDD` (ordinal day), `HH`, `hh`, `mm`, `ss` |

### Options

| Option | Effect |
|---|---|
| `DateOnly()` | Zero out time components |
| `TimeOnly()` | Zero out date components |
| `ToTimezone(tz *time.Location)` | Convert to target timezone before formatting |
| `FormatAs(format DatetimeFormat)` | Override the default ISO output format |

## 🧭 Supported input layouts

`dt` attempts these layouts in order when parsing a string (first match wins):

- **ISO 8601:** `2006-01-02T15:04:05Z`, `...00.000Z`, `...-07:00`, `...+07:00`
- **Date-only:** `2006-01-02`
- **Time-only:** `15:04:05`, `15:04`
- **Slash/dot/dash dates:** `01/02/2006`, `1/2/2006`, `02/01/2006`, `2/1/2006`, `02.01.2006`, `2.1.2006`, `01-02-2006`, `1-2-2006`
- **Date-with-time:** `01/02/2006 15:04:05`, `02.01.2006 15:04:05`, `2006-01-02 15:04:05`
- **Month names:** `Jan 2 2006`, `Jan 2, 2006`, `January 2 2006`, `January 2, 2006`, `2 Jan 2006`, `2 January 2006`
- **Short year:** `Jan 2 06`, `Jan 2, 06`, `1/2/06`, `01/02/06`, `2/1/06`, `02/01/06`
- **Unix timestamps** (numeric strings): 10-digit seconds, 13-digit milliseconds
- **Ordinal days** (`1st`, `2nd`, `3rd`, ...) are stripped before parsing, allowing inputs like `Jan 4th 25`.

## 🔗 Related bold-minds libraries

- [`bold-minds/to`](https://github.com/bold-minds/to) — safe type conversion. Useful when feeding API input through `dt`.
- [`bold-minds/txt`](https://github.com/bold-minds/txt) — string formatting and manipulation. Pair with `dt.NewDatetime` when composing log or message strings.
- [`bold-minds/dig`](https://github.com/bold-minds/dig) — nested data navigation. Extract a timestamp field from a deep JSON blob, then pass it to `dt.NewDatetime`.

## 🚫 Non-goals

- **No locale support.** Month names are English-only. Non-English month names will not parse.
- **No relative-time parsing.** "yesterday", "2 hours ago", "next Friday" are not supported. Use a dedicated library.
- **No calendar arithmetic.** "Add 30 days", "start of week", etc. belong in Go's `time` package or a calendar library.
- **No ambiguity resolution.** If a date is ambiguous between US and European (`01/02/2006`), `dt` picks the first match. If your application is locale-sensitive, parse with an explicit layout via `time.Parse` and skip `dt`.
- **No timezone database bundling.** `ToTimezone` relies on the caller providing a `*time.Location` from `time.LoadLocation`.

## 📄 License

MIT — see [LICENSE](LICENSE).
