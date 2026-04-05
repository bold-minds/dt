// Package dt provides ergonomic datetime parsing and formatting for Go.
//
// The canonical API is small:
//
//	dt.New(value, opts...)   // format any supported value to a string
//	dt.Parse(value)          // parse to time.Time, returns zero on failure
//	dt.ParseAny(value)       // parse to (time.Time, error) — preferred
//
// Values accepted by Parse and ParseAny: time.Time, any int/uint/float width,
// and strings in many common formats (RFC3339, RFC3339Nano, date-only,
// US/EU numeric, month-name natural language, and numeric Unix timestamps).
//
// Output format is controlled via Format (interface) values, constructed by
// ISO(), Unix(), and Custom(pattern). Options (Option interface) configure
// DateOnly, TimeOnly, timezone conversion, and output format.
//
// go.mod targets go 1.21 intentionally: this library is published and the
// Goby library family keeps broad compatibility.
package dt

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// Core types
// -----------------------------------------------------------------------------

// Format represents a datetime output format. Use ISO, Unix, or Custom to
// construct one. The zero value of any concrete Format type is not meaningful;
// callers should always use the constructors.
type Format interface {
	format(t time.Time) string
}

// Option configures parsing and formatting behavior for New. Construct options
// via ToTimezone, DateOnly, TimeOnly, and FormatAs.
type Option interface {
	apply(config *datetimeConfig)
}

// ParseResult is the structured result returned by the legacy ParseDatetime
// function. Its zero value represents an unparsed result; it is only
// meaningful when returned with a nil error.
//
// Format is the format label detected for a string input (e.g. "iso", "unix")
// or the format label the caller supplied. It records what ParseDatetime
// thinks the input *looked like*, not the parser that actually succeeded.
type ParseResult struct {
	UnixMillis int64
	Format     string
	Pattern    string
}

// datetimeConfig is the internal state assembled from Option values.
type datetimeConfig struct {
	dateOnly     bool
	timeOnly     bool
	timezone     *time.Location
	outputFormat Format // nil means "use the default based on dateOnly/timeOnly"
}

// Legacy type aliases preserved for source compatibility with dt v0.1.0.
// New code should use the unprefixed names.
//
// Deprecated: use Format instead.
type DatetimeFormat = Format

// Deprecated: use Option instead.
type DatetimeOption = Option

// Deprecated: use ParseResult instead.
type DatetimeParseResult = ParseResult

// -----------------------------------------------------------------------------
// Format label constants (used by the legacy API and DetectDatetimeFormat)
// -----------------------------------------------------------------------------

const (
	FormatUnix   = "unix"
	FormatISO    = "iso"
	FormatISOTZ  = "iso-tz"
	FormatDate   = "date"
	FormatCustom = "custom"
)

// -----------------------------------------------------------------------------
// Format implementations
// -----------------------------------------------------------------------------

type isoFormat struct{}
type unixFormat struct{}

// customFormat caches its pre-tokenized layout so conversion runs once at
// construction rather than on every format call.
type customFormat struct {
	pattern string
	tokens  []patternToken
}

// format emits t as RFC3339 in the time's current location, preserving the
// numeric offset. This is both the "ISO" and "ISO-TZ" behavior: a UTC time
// renders with "Z" and a non-UTC time renders with "±hh:mm".
func (isoFormat) format(t time.Time) string {
	return t.Format(time.RFC3339)
}

func (unixFormat) format(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

func (f customFormat) format(t time.Time) string {
	var b strings.Builder
	b.Grow(len(f.pattern) + 16)
	for _, tk := range f.tokens {
		switch tk.kind {
		case tokLiteral:
			b.WriteString(tk.value)
		case tokGoLayout:
			b.WriteString(t.Format(tk.value))
		case tokOrdinalDay:
			b.WriteString(getOrdinalDay(t.Day()))
		case tokMillis:
			fmt.Fprintf(&b, "%03d", t.Nanosecond()/int(time.Millisecond))
		}
	}
	return b.String()
}

// ISO returns a format that emits RFC3339 with the time's location offset.
// Unlike the legacy behavior this does NOT force UTC — a time carrying a
// -05:00 offset formats with "-05:00", not "Z".
func ISO() Format { return isoFormat{} }

// Unix returns a format that emits the Unix time in milliseconds as a decimal
// string.
func Unix() Format { return unixFormat{} }

// Custom returns a format that uses a human-friendly token pattern. Supported
// tokens:
//
//	YYYY = 4-digit year        YY   = 2-digit year
//	MMMM = full month name     MMM  = short month name
//	MM   = 2-digit month       M    = 1-or-2 digit month
//	DDD  = ordinal day (1st)   DD   = 2-digit day            D = 1-or-2 digit day
//	HH   = 24-hour (00-23)     H    = 24-hour (Go has no 1-digit variant)
//	hh   = 12-hour (01-12)     h    = 1-or-2 digit 12-hour
//	mm   = 2-digit minute      m    = 1-or-2 digit minute
//	ss   = 2-digit second      s    = 1-or-2 digit second
//	.SSS = fractional seconds (3 digits, e.g. ".123")
//	SSS  = milliseconds as 3 digits without a leading dot
//
// All other characters are emitted literally, including characters Go's own
// time package would interpret as format verbs.
func Custom(pattern string) Format {
	return customFormat{pattern: pattern, tokens: tokenizePattern(pattern)}
}

// -----------------------------------------------------------------------------
// Option implementations
// -----------------------------------------------------------------------------

type dateOnlyOption struct{}
type timeOnlyOption struct{}
type timezoneOption struct{ tz *time.Location }
type formatAsOption struct{ f Format }

func (dateOnlyOption) apply(c *datetimeConfig)   { c.dateOnly = true }
func (timeOnlyOption) apply(c *datetimeConfig)   { c.timeOnly = true }
func (o timezoneOption) apply(c *datetimeConfig) { c.timezone = o.tz }
func (o formatAsOption) apply(c *datetimeConfig) { c.outputFormat = o.f }

// DateOnly returns an option that zeros the time-of-day components and,
// unless FormatAs is also provided, switches the default output format to
// "2006-01-02". The calendar day is taken from the input's own location —
// parsing "2023-01-15T23:00:00-05:00" with DateOnly yields "2023-01-15", not
// the UTC day "2023-01-16". To select the day in a specific zone, combine
// with ToTimezone. DateOnly and TimeOnly are mutually exclusive; passing
// both causes New to return the empty string.
func DateOnly() Option { return dateOnlyOption{} }

// TimeOnly returns an option that zeros the date components (preserving the
// clock) and, unless FormatAs is also provided, switches the default output
// format to "15:04:05". DateOnly and TimeOnly are mutually exclusive; passing
// both causes New to return the empty string.
func TimeOnly() Option { return timeOnlyOption{} }

// ToTimezone returns an option that converts the parsed time to the given
// location before formatting. A nil location is a no-op and preserves the
// parsed time's location.
func ToTimezone(tz *time.Location) Option { return timezoneOption{tz: tz} }

// FormatAs returns an option that selects the output format explicitly.
func FormatAs(format Format) Option { return formatAsOption{f: format} }

// -----------------------------------------------------------------------------
// Canonical API: New, Parse, ParseAny
// -----------------------------------------------------------------------------

// New parses value and formats it according to the given options, returning
// the formatted string. It returns the empty string if value cannot be
// parsed, if the type is unsupported, or if DateOnly and TimeOnly are both
// specified. Callers who need to distinguish these cases should use ParseAny.
func New(value any, opts ...Option) string {
	t := parseToTime(value)
	if t.IsZero() {
		return ""
	}

	cfg := &datetimeConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	// Mutually exclusive: both set is a programming error, return empty.
	if cfg.dateOnly && cfg.timeOnly {
		return ""
	}

	if cfg.timezone != nil {
		t = t.In(cfg.timezone)
	}

	// Normalize the timestamp. Note: we zero components in t.Location() so
	// that subsequent formatting (which we also perform in t.Location()) does
	// not roll the date across a timezone boundary — that was the pre-fix bug
	// where DateOnly + a non-UTC timezone produced the wrong date.
	switch {
	case cfg.dateOnly:
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case cfg.timeOnly:
		t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	}

	// Pick default format based on filters if the caller did not select one.
	if cfg.outputFormat == nil {
		switch {
		case cfg.dateOnly:
			cfg.outputFormat = Custom("YYYY-MM-DD")
		case cfg.timeOnly:
			cfg.outputFormat = Custom("HH:mm:ss")
		default:
			cfg.outputFormat = ISO()
		}
	}

	return cfg.outputFormat.format(t)
}

// Parse parses value to a time.Time. It returns the zero time on any failure.
// Callers who need to distinguish "unparseable" from "unsupported type" should
// use ParseAny.
func Parse(value any) time.Time {
	return parseToTime(value)
}

// ParseAny parses value to a time.Time and returns a descriptive error on
// failure. This is the preferred primitive for callers that need to
// distinguish unparseable input from an unsupported type or a zero time.Time.
func ParseAny(value any) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("dt: cannot parse nil")
	}
	t := parseToTime(value)
	if t.IsZero() {
		switch v := value.(type) {
		case time.Time:
			// Caller passed a zero time deliberately; surface as error.
			return time.Time{}, fmt.Errorf("dt: zero time.Time is not a valid datetime")
		case string:
			return time.Time{}, fmt.Errorf("dt: cannot parse %q as datetime", v)
		case float32:
			f := float64(v)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return time.Time{}, fmt.Errorf("dt: invalid numeric value %v is not a valid timestamp", v)
			}
			return time.Time{}, fmt.Errorf("dt: numeric value %v out of valid timestamp range", v)
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return time.Time{}, fmt.Errorf("dt: invalid numeric value %v is not a valid timestamp", v)
			}
			return time.Time{}, fmt.Errorf("dt: numeric value %v out of valid timestamp range", v)
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			return time.Time{}, fmt.Errorf("dt: numeric value %v out of valid timestamp range", v)
		default:
			return time.Time{}, fmt.Errorf("dt: unsupported type %T for datetime parsing", value)
		}
	}
	return t, nil
}

// -----------------------------------------------------------------------------
// Parsing core
// -----------------------------------------------------------------------------

// parseToTime normalizes any supported input type to a time.Time. It returns
// a zero time.Time on any failure. All integer and unsigned integer widths
// are supported; float widths preserve sub-second precision where possible.
func parseToTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		return parseDatetimeString(v)

	// Signed integer widths — all funnel through parseUnixTimestamp.
	case int:
		return parseUnixTimestamp(int64(v))
	case int8:
		return parseUnixTimestamp(int64(v))
	case int16:
		return parseUnixTimestamp(int64(v))
	case int32:
		return parseUnixTimestamp(int64(v))
	case int64:
		return parseUnixTimestamp(v)

	// Unsigned integer widths. uint64 can exceed int64 range, so we check.
	case uint:
		if uint64(v) > math.MaxInt64 {
			return time.Time{}
		}
		return parseUnixTimestamp(int64(v))
	case uint8:
		return parseUnixTimestamp(int64(v))
	case uint16:
		return parseUnixTimestamp(int64(v))
	case uint32:
		return parseUnixTimestamp(int64(v))
	case uint64:
		if v > math.MaxInt64 {
			return time.Time{}
		}
		return parseUnixTimestamp(int64(v))

	// Float widths preserve sub-second precision.
	case float32:
		return parseUnixFloat(float64(v))
	case float64:
		return parseUnixFloat(v)

	default:
		return time.Time{}
	}
}

// Maximum accepted Unix timestamp: 9999999999999 milliseconds ≈ year 2286.
// The same bound applies to every numeric entry point (string, int, float) so
// callers get identical validity envelopes regardless of input type.
const maxUnixMillis int64 = 9999999999999

// unixSecondsMillisCutoff is the boundary between "interpret as seconds" and
// "interpret as milliseconds" for numeric Unix timestamps. Values strictly
// below 10^10 are seconds; values at or above it are milliseconds. 10^10
// seconds ≈ year 2286, so any legitimate seconds-since-epoch value for the
// foreseeable future is safely under this cutoff.
const unixSecondsMillisCutoff int64 = 10_000_000_000

// parseUnixTimestamp converts an integer Unix timestamp to time.Time. Values
// with fewer than 10 digits (< unixSecondsMillisCutoff) are interpreted as
// seconds; larger values up to maxUnixMillis are interpreted as milliseconds.
// Microsecond and nanosecond timestamps are rejected — callers that need them
// should convert to time.Time directly.
//
// Numeric inputs carry no zone information; the returned time is always UTC.
// To render in another zone, combine with ToTimezone.
func parseUnixTimestamp(timestamp int64) time.Time {
	if timestamp < 0 || timestamp > maxUnixMillis {
		return time.Time{}
	}
	if timestamp < unixSecondsMillisCutoff {
		return time.Unix(timestamp, 0).UTC()
	}
	return time.UnixMilli(timestamp).UTC()
}

// parseUnixFloat converts a float Unix timestamp to time.Time, preserving
// fractional seconds. Values < unixSecondsMillisCutoff are treated as seconds
// (fractional part is nanoseconds); values >= unixSecondsMillisCutoff are
// treated as milliseconds (fractional part is discarded — sub-millisecond
// precision is not representable there). Returned times are always UTC.
//
// NaN and ±Inf are rejected explicitly. Every comparison with NaN is false,
// so without this guard a NaN input would fall through the range checks into
// int64(NaN) — which is implementation-defined, typically MinInt64 on amd64,
// producing a wildly wrong "valid-looking" time.Time. Memory note from the
// Goby review: NaN handling is a standing review checkpoint for this family.
func parseUnixFloat(v float64) time.Time {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return time.Time{}
	}
	if v < 0 || v > float64(maxUnixMillis) {
		return time.Time{}
	}
	if v < float64(unixSecondsMillisCutoff) {
		sec := int64(v)
		// nsec can round to exactly 1e9 for values like 1673784000.9999999999;
		// time.Unix normalizes that by rolling into the next second, so the
		// output is correct without an explicit clamp here. Intentional
		// reliance on time.Unix's normalization contract.
		nsec := int64((v - float64(sec)) * 1e9)
		return time.Unix(sec, nsec).UTC()
	}
	return time.UnixMilli(int64(v)).UTC()
}

// isDatetime performs a fast heuristic reject for strings that obviously are
// not datetime-shaped. A true result does not guarantee the string parses; a
// false result guarantees we skip the (expensive) layout-tryout loop.
func isDatetime(s string) bool {
	sLen := len(s)
	// Upper bound covers RFC3339Nano with offset (~35 chars) plus slack.
	if sLen == 0 || sLen > 64 {
		return false
	}

	// Fast path: pure-digit strings get treated as potential Unix timestamps
	// with the same range as parseUnixTimestamp accepts (≤ 13 digits).
	allDigits := true
	for i := 0; i < sLen; i++ {
		if s[i] < '0' || s[i] > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return sLen <= 13
	}

	// Non-numeric strings need a minimum shape.
	if sLen < 6 {
		return false
	}

	// Fast path: strings containing "T" and at least one digit are overwhelmingly
	// ISO-like and worth trying.
	if strings.IndexByte(s, 'T') >= 0 {
		for i := 0; i < sLen; i++ {
			if s[i] >= '0' && s[i] <= '9' {
				return true
			}
		}
	}

	var (
		hasDigit     bool
		hasSeparator bool
		digitCount   int
		letterCount  int
		spaceCount   int
	)
	for i := 0; i < sLen; i++ {
		b := s[i]
		switch {
		case b >= '0' && b <= '9':
			hasDigit = true
			digitCount++
		case b == '-' || b == '/' || b == ':' || b == 'T' || b == 'Z' || b == '.' || b == ',':
			hasSeparator = true
		case (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z'):
			letterCount++
			// Cap at 12: the longest legitimate prefix is "September" (9) plus
			// a short suffix such as " 2," (letters-only count stays ≤ 12).
			// Anything longer is non-datetime text (URLs, sentences, etc.).
			if letterCount > 12 {
				return false
			}
		case b == ' ':
			spaceCount++
		}
	}
	if !hasDigit {
		return false
	}
	// No separators and no spaces at this point means non-numeric junk — the
	// all-digit fast path above already handled the valid shape.
	if !hasSeparator && spaceCount == 0 {
		return false
	}

	if spaceCount > 0 {
		if spaceCount > 1 {
			return true
		}
		// Exactly one space: inspect trailing segment for common junk
		// (alphanumeric codes, long numeric IDs, short letter codes).
		lastSpace := strings.LastIndexByte(s, ' ')
		if lastSpace >= 0 && lastSpace < sLen-1 {
			trailing := s[lastSpace+1:]
			tLen := len(trailing)
			if tLen == 2 && (trailing == "AM" || trailing == "PM" ||
				trailing == "am" || trailing == "pm") {
				return true
			}
			if tLen <= 6 {
				tDigits, tLetters := 0, 0
				for i := 0; i < tLen; i++ {
					b := trailing[i]
					switch {
					case b >= '0' && b <= '9':
						tDigits++
					case (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z'):
						tLetters++
					}
				}
				if tDigits == tLen && tLen >= 5 {
					return false
				}
				if tDigits > 0 && tLetters > 0 {
					return false
				}
				if tLetters > 0 && tLen <= 2 {
					return false
				}
			}
		}
	}
	return digitCount >= sLen/4
}

// parseLayouts is the ordered list of layouts attempted when an input string
// doesn't parse as a Unix timestamp. time.RFC3339 and time.RFC3339Nano cover
// the entire ISO 8601 family (with or without fractional seconds, Z or
// numeric offset), replacing six hand-written variants in the v0.1.0 code.
//
// Note: both US ("01/02/2006") and European ("02/01/2006") orderings appear.
// The US form is first, so ambiguous input like "03/04/2023" is parsed as
// 2023-03-04 (March 4). This is documented behavior; callers with locale
// knowledge should use ParseAny with a known unambiguous format upstream.
var parseLayouts = [...]string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02",
	"15:04:05",
	// Note: "15:04" (HH:MM only) is intentionally NOT in this list. The
	// isDatetime prefilter rejects non-numeric strings shorter than 6 chars,
	// so a 5-char layout could never be reached. Callers needing HH:MM should
	// supply seconds ("15:04:00") or use time.Parse directly.
	"01/02/2006", // US
	"1/2/2006",
	"02/01/2006", // European
	"2/1/2006",
	"02.01.2006",
	"2.1.2006",
	"01-02-2006",
	"1-2-2006",
	"01/02/2006 15:04:05",
	"02.01.2006 15:04:05",
	"2006-01-02 15:04:05",
	"Jan 2 2006",
	"Jan 2, 2006",
	"January 2 2006",
	"January 2, 2006",
	"2 Jan 2006",
	"2 January 2006",
	"Jan 2 06",
	"Jan 2, 06",
	"1/2/06",
	"01/02/06",
	"2/1/06",
	"02/01/06",
}

// parseDatetimeString tries to parse a string datetime value using the shared
// layout list. It preserves the parsed location — the value's original offset
// is NOT forced to UTC.
func parseDatetimeString(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if !isDatetime(value) {
		return time.Time{}
	}
	value = stripOrdinalSuffixes(value)

	if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parseUnixTimestamp(ts)
	}

	for _, layout := range parseLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// -----------------------------------------------------------------------------
// Custom pattern tokenizer (token-walker, not cascading ReplaceAll)
// -----------------------------------------------------------------------------

type tokenKind int

const (
	tokLiteral tokenKind = iota
	tokGoLayout
	tokOrdinalDay
	tokMillis
)

type patternToken struct {
	kind  tokenKind
	value string // Go layout fragment for tokGoLayout, literal text for tokLiteral
}

// patternRule maps a human-friendly token to either a Go time-layout fragment
// or a special token kind. Rules are sorted longest-first before use so that
// "MMMM" matches before "MMM" and ".SSS" matches before "SSS".
//
// allAlpha is set for rules whose token is composed entirely of ASCII
// letters. These rules only match at word boundaries — if the character
// immediately before or after the match is also a letter, the rule does not
// match. This prevents single-letter tokens like "M" or "h" from corrupting
// literal words such as "Month:" or "Hour:" (BUG-1).
type patternRule struct {
	token    string
	kind     tokenKind
	goLayout string
	allAlpha bool
}

// isASCIIAlpha reports whether b is an ASCII letter.
func isASCIIAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isAllASCIIAlpha reports whether every byte of s is an ASCII letter.
func isAllASCIIAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isASCIIAlpha(s[i]) {
			return false
		}
	}
	return true
}

var patternRules = func() []patternRule {
	rules := []patternRule{
		// Year
		{token: "YYYY", kind: tokGoLayout, goLayout: "2006"},
		{token: "YY", kind: tokGoLayout, goLayout: "06"},
		// Month
		{token: "MMMM", kind: tokGoLayout, goLayout: "January"},
		{token: "MMM", kind: tokGoLayout, goLayout: "Jan"},
		{token: "MM", kind: tokGoLayout, goLayout: "01"},
		{token: "M", kind: tokGoLayout, goLayout: "1"},
		// Day — DDD (ordinal) must come before DD/D
		{token: "DDD", kind: tokOrdinalDay},
		{token: "DD", kind: tokGoLayout, goLayout: "02"},
		{token: "D", kind: tokGoLayout, goLayout: "2"},
		// Hour (24-hour). Go has no single-digit 24-hour verb — "H" and "HH"
		// both emit "15" (zero-padded 00..23). Callers needing unpadded
		// 24-hour output must post-process.
		{token: "HH", kind: tokGoLayout, goLayout: "15"},
		{token: "H", kind: tokGoLayout, goLayout: "15"},
		// Hour (12-hour)
		{token: "hh", kind: tokGoLayout, goLayout: "03"},
		{token: "h", kind: tokGoLayout, goLayout: "3"},
		// Minute
		{token: "mm", kind: tokGoLayout, goLayout: "04"},
		{token: "m", kind: tokGoLayout, goLayout: "4"},
		// Second
		{token: "ss", kind: tokGoLayout, goLayout: "05"},
		{token: "s", kind: tokGoLayout, goLayout: "5"},
		// Fractional seconds — ".SSS" must come before "SSS" so "HH:mm:ss.SSS"
		// tokenizes the dot+fractional as a single unit.
		{token: ".SSS", kind: tokGoLayout, goLayout: ".000"},
		{token: "SSS", kind: tokMillis},
	}
	for i := range rules {
		rules[i].allAlpha = isAllASCIIAlpha(rules[i].token)
	}
	sort.SliceStable(rules, func(i, j int) bool {
		return len(rules[i].token) > len(rules[j].token)
	})
	return rules
}()

// tokenizePattern walks pattern once, emitting tokens for recognized tokens
// and accumulating everything else as literal runs. This replaces the
// cascading strings.ReplaceAll approach, which corrupted literal text by
// letting each replacement's output feed into the next rule.
func tokenizePattern(pattern string) []patternToken {
	var out []patternToken
	var lit strings.Builder
	flushLit := func() {
		if lit.Len() > 0 {
			out = append(out, patternToken{kind: tokLiteral, value: lit.String()})
			lit.Reset()
		}
	}
	// prevWasToken records whether the previous character was consumed by a
	// token rule. The left-boundary check allows alpha-only tokens to follow
	// other tokens (so "HHmm" correctly tokenizes as HH + mm) but rejects
	// them when the previous character was a literal letter (so "Month:" is
	// not mangled by the "M" rule).
	prevWasToken := false
	i := 0
	for i < len(pattern) {
		matched := false
		for _, r := range patternRules {
			if !strings.HasPrefix(pattern[i:], r.token) {
				continue
			}
			// Word-boundary check for all-alphabetic tokens. A token match
			// is valid if:
			//   - left side: start-of-pattern, previous char non-alpha, or
			//     previous char was part of another token
			//   - right side: end-of-pattern, next char non-alpha, or the
			//     remaining pattern starts with another alpha token
			if r.allAlpha {
				if i > 0 && !prevWasToken && isASCIIAlpha(pattern[i-1]) {
					continue
				}
				end := i + len(r.token)
				if end < len(pattern) && isASCIIAlpha(pattern[end]) &&
					!startsWithAlphaToken(pattern[end:]) {
					continue
				}
			}
			flushLit()
			out = append(out, patternToken{kind: r.kind, value: r.goLayout})
			i += len(r.token)
			prevWasToken = true
			matched = true
			break
		}
		if !matched {
			lit.WriteByte(pattern[i])
			i++
			prevWasToken = false
		}
	}
	flushLit()
	return out
}

// startsWithAlphaToken reports whether s begins with any of our alphabetic
// pattern tokens (YYYY, MMMM, HH, ss, etc.). Used by the tokenizer's
// right-boundary check to allow tokens to sit immediately adjacent to each
// other ("HHmmss" → HH + mm + ss) while still rejecting tokens embedded in
// literal words.
func startsWithAlphaToken(s string) bool {
	for _, r := range patternRules {
		if r.allAlpha && strings.HasPrefix(s, r.token) {
			return true
		}
	}
	return false
}

// getOrdinalDay converts a day-of-month number to ordinal format.
func getOrdinalDay(day int) string {
	switch {
	case day >= 11 && day <= 13:
		return fmt.Sprintf("%dth", day)
	case day%10 == 1:
		return fmt.Sprintf("%dst", day)
	case day%10 == 2:
		return fmt.Sprintf("%dnd", day)
	case day%10 == 3:
		return fmt.Sprintf("%drd", day)
	default:
		return fmt.Sprintf("%dth", day)
	}
}

// ordinalSuffixRE matches any ordinal suffix (1st, 2nd, 3rd, 4th, ...) in any
// case combination, anchored to a digit run. Using a regex avoids the
// arithmetic-coincidence fragility of the prior substring-replace approach
// and handles mixed-case input ("2Nd", "31St") uniformly.
var ordinalSuffixRE = regexp.MustCompile(`(?i)(\d+)(st|nd|rd|th)\b`)

// stripOrdinalSuffixes removes ordinal suffixes from a datetime string so the
// remaining digits can be parsed by time.Parse. "jan 4th 25" → "jan 4 25".
func stripOrdinalSuffixes(s string) string {
	return ordinalSuffixRE.ReplaceAllString(s, "$1")
}
