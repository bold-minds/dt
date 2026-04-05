package dt

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

// This file contains the deprecated v0.1.0 API. Every function is a thin
// wrapper over the canonical API in dt.go. All of these are marked
// // Deprecated: so golangci-lint and gopls surface them. They remain for
// source compatibility with v0.1.0 callers.
//
// All silent-corruption bugs from v0.1.0 are also fixed in this layer:
//
//   - FormatDatetime no longer silently returns Unix millis for unknown
//     format codes (BUG-5).
//   - ParseDatetime accepts the same numeric range as parseUnixTimestamp
//     (BUG-6).
//   - IsValidDatetimeFormat actually inspects the pattern instead of
//     hardcoding a single rejected value (BUG-8).
//   - DetectDatetimeFormat uses package-scope regexes (BUG-14).
//   - FormatDatetime no longer shadows the imported "fmt" package (BUG-28).

// NewDatetime is the v0.1.0 entry point for formatting any supported value.
//
// Deprecated: use New instead.
func NewDatetime(value any, opts ...Option) string { return New(value, opts...) }

// ToDatetime parses a string value to a time.Time.
//
// Deprecated: use Parse (for the lossy variant) or ParseAny (for an
// error-returning variant) instead.
func ToDatetime(value string) time.Time { return parseDatetimeString(value) }

// ParseDatetime parses value and returns a structured ParseResult. This is
// the v0.1.0 legacy entry point; new code should use ParseAny for an
// error-returning primitive that returns time.Time directly.
//
// Deprecated: use ParseAny instead.
func ParseDatetime(value any, expectedFormat string, customPattern string) (*ParseResult, error) {
	var t time.Time

	switch v := value.(type) {
	case string:
		t = parseDatetimeString(v)
		if t.IsZero() {
			return nil, fmt.Errorf("dt: unable to parse datetime string: %q", v)
		}
	case time.Time:
		if v.IsZero() {
			return nil, fmt.Errorf("dt: zero time.Time is not a valid datetime")
		}
		t = v
	case int64:
		if v < 0 || v > maxUnixMillis {
			return nil, fmt.Errorf("dt: timestamp out of valid range: %d", v)
		}
		t = parseUnixTimestamp(v)
	case int:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case int8:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case int16:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case int32:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case uint8:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case uint16:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case uint32:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case uint:
		// uint is 32 or 64 bits depending on platform; on 64-bit it can
		// exceed int64 range, so guard explicitly (gosec G115).
		if uint64(v) > math.MaxInt64 {
			return nil, fmt.Errorf("dt: timestamp out of valid range: %d", v)
		}
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case uint64:
		if v > math.MaxInt64 {
			return nil, fmt.Errorf("dt: timestamp out of valid range: %d", v)
		}
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case float64:
		parsed := parseUnixFloat(v)
		if parsed.IsZero() {
			return nil, fmt.Errorf("dt: timestamp out of valid range: %v", v)
		}
		t = parsed
	case float32:
		return ParseDatetime(float64(v), expectedFormat, customPattern)
	default:
		return nil, fmt.Errorf("dt: unsupported type for datetime parsing: %T", value)
	}

	// Determine detected format label if not explicitly provided.
	format := expectedFormat
	if format == "" {
		if s, ok := value.(string); ok {
			format = DetectDatetimeFormat(s)
		} else {
			format = FormatUnix
		}
	}

	return &ParseResult{
		UnixMillis: t.UnixMilli(),
		Format:     format,
		Pattern:    customPattern,
	}, nil
}

// Package-scope detection regexes. Compiled once at import time instead of
// once per DetectDatetimeFormat call.
var (
	detectUnixRE  = regexp.MustCompile(`^\d{10,13}$`)
	detectISORE   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$`)
	detectISOTZRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?[+-]\d{2}:\d{2}$`)
	detectDateRE  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// DetectDatetimeFormat classifies a datetime string into one of the legacy
// format labels. Returns FormatCustom for anything it can't classify.
//
// Deprecated: format detection is a weak signal and the returned label has
// no operational meaning for the canonical API.
func DetectDatetimeFormat(input string) string {
	input = trimSpace(input)
	if input == "" {
		return FormatCustom
	}
	switch {
	case detectUnixRE.MatchString(input):
		return FormatUnix
	case detectISORE.MatchString(input):
		return FormatISO
	case detectISOTZRE.MatchString(input):
		return FormatISOTZ
	case detectDateRE.MatchString(input):
		return FormatDate
	default:
		return FormatCustom
	}
}

// trimSpace is a package-local alias to avoid pulling strings into legacy.go
// just for one call (and to keep the deprecated file's imports minimal).
func trimSpace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	j := len(s)
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

// IsValidDatetimeFormat reports whether format is either a known format
// label ("unix", "iso", "iso-tz", "date", "custom") or a custom pattern
// containing at least one recognized pattern token.
//
// This replaces the v0.1.0 implementation which returned true for anything
// except the literal string "invalid" (BUG-8).
//
// Deprecated: this function exists for v0.1.0 source compatibility and has
// no role in the canonical API.
func IsValidDatetimeFormat(format string) bool {
	if format == "" {
		return false
	}
	switch format {
	case FormatUnix, FormatISO, FormatISOTZ, FormatDate, FormatCustom:
		return true
	}
	// Treat as a custom pattern — valid iff at least one recognized token.
	for _, tk := range tokenizePattern(format) {
		if tk.kind != tokLiteral {
			return true
		}
	}
	return false
}

// FormatDatetime formats a Unix-millisecond timestamp according to a legacy
// format label. If format is not a known label, it is treated as either a
// custom pattern (if it contains any recognized token) or a raw Go time
// layout (if it does not). This replaces the v0.1.0 behavior which silently
// fell back to Unix for unknown formats (BUG-5).
//
// Deprecated: use New with FormatAs instead.
func FormatDatetime(unixMillis int64, format string, customPattern string) (string, error) {
	t := time.UnixMilli(unixMillis).UTC()
	// Note: variable is named "f", not "fmt", to avoid shadowing the imported
	// "fmt" package (BUG-28).
	var f Format

	switch format {
	case FormatUnix:
		f = Unix()
	case FormatISO, FormatISOTZ:
		// Both label to RFC3339; the label distinction was a v0.1.0 artifact
		// (BUG-20). Since the input here is a UTC millisecond timestamp there
		// is no original offset to preserve anyway.
		f = ISO()
	case FormatDate:
		f = Custom("YYYY-MM-DD")
	case FormatCustom:
		if customPattern == "" {
			return "", fmt.Errorf("dt: FormatDatetime: format=%q requires non-empty customPattern", FormatCustom)
		}
		// If the pattern contains any of our recognized tokens treat it as a
		// dt-style pattern; otherwise fall back to a raw Go time layout. This
		// preserves v0.1.0 behavior where callers passed Go layouts like
		// "2006-01-02" directly as customPattern.
		if hasAnyPatternToken(customPattern) {
			f = Custom(customPattern)
		} else {
			return t.Format(customPattern), nil
		}
	default:
		// Unknown label. If it looks like one of our custom patterns, use it
		// as one; otherwise assume the caller passed a Go time layout.
		if hasAnyPatternToken(format) {
			f = Custom(format)
		} else {
			return t.Format(format), nil
		}
	}

	return f.format(t), nil
}

// hasAnyPatternToken reports whether a string contains at least one custom
// pattern token (YYYY, MM, DD, HH, etc.).
func hasAnyPatternToken(s string) bool {
	for _, tk := range tokenizePattern(s) {
		if tk.kind != tokLiteral {
			return true
		}
	}
	return false
}

// ParseTimeString parses a string to a time.Time, returning an error on
// failure. Provided for v0.1.0 source compatibility.
//
// Deprecated: use ParseAny instead.
func ParseTimeString(s string) (time.Time, error) {
	t := parseDatetimeString(s)
	if t.IsZero() {
		return time.Time{}, fmt.Errorf("dt: unable to parse datetime string: %q", s)
	}
	return t, nil
}
