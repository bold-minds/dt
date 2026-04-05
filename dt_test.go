package dt_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/bold-minds/dt"
)

// ---------------------------------------------------------------------------
// Canonical API: New + Parse + ParseAny
// ---------------------------------------------------------------------------

// Well-known reference instant used across most tests: 2023-01-15T12:00:00Z.
const refISO = "2023-01-15T12:00:00Z"

func TestNew_NumericTimestamps(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"seconds int64", int64(1673784000), "2023-01-15T12:00:00Z"},
		{"millis int64", int64(1673784000000), "2023-01-15T12:00:00Z"},
		{"seconds int", int(1673784000), "2023-01-15T12:00:00Z"},
		{"seconds int32", int32(1673784000), "2023-01-15T12:00:00Z"},
		{"seconds uint32", uint32(1673784000), "2023-01-15T12:00:00Z"},
		{"seconds float64", float64(1673784000), "2023-01-15T12:00:00Z"},
		{"zero epoch", int64(0), "1970-01-01T00:00:00Z"},
		{"max bound", int64(9999999999999), "2286-11-20T17:46:39Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dt.New(tc.value)
			if got != tc.want {
				t.Errorf("New(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestNew_StringInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"RFC3339 UTC", "2023-01-15T12:00:00Z", "2023-01-15T12:00:00Z"},
		{"RFC3339Nano", "2023-01-15T12:00:00.123456789Z", "2023-01-15T12:00:00Z"},
		{"RFC3339 offset", "2023-01-15T12:00:00-05:00", "2023-01-15T12:00:00-05:00"},
		{"date only", "2023-01-15", "2023-01-15T00:00:00Z"},
		{"US slash", "01/15/2023", "2023-01-15T00:00:00Z"},
		{"unix-seconds string", "1673784000", "2023-01-15T12:00:00Z"},
		{"unix-millis string", "1673784000000", "2023-01-15T12:00:00Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dt.New(tc.input)
			if got != tc.want {
				t.Errorf("New(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNew_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"empty string", ""},
		{"garbage string", "not-a-date"},
		{"unsupported type", []int{1, 2, 3}},
		{"nil", nil},
		{"negative timestamp", int64(-1)},
		{"overflow timestamp", int64(99999999999999)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := dt.New(tc.value); got != "" {
				t.Errorf("New(%v) = %q, want empty", tc.value, got)
			}
		})
	}
}

func TestParse_Basic(t *testing.T) {
	got := dt.Parse("2023-01-15T12:00:00Z")
	want := time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Parse() = %v, want %v", got, want)
	}
}

func TestParse_ZeroOnFailure(t *testing.T) {
	for _, input := range []string{"", "not-a-date", "   "} {
		if got := dt.Parse(input); !got.IsZero() {
			t.Errorf("Parse(%q) = %v, want zero time", input, got)
		}
	}
}

func TestParseAny_ErrorCategorization(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr string
	}{
		{"nil", nil, "cannot parse nil"},
		{"unparseable string", "nonsense", "cannot parse"},
		{"unsupported type", []int{1}, "unsupported type"},
		{"out-of-range int", int64(99999999999999), "out of valid timestamp range"},
		{"zero time.Time", time.Time{}, "zero time.Time"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dt.ParseAny(tc.value)
			if err == nil {
				t.Fatalf("ParseAny(%v) err = nil, want non-nil", tc.value)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("ParseAny(%v) err = %q, want to contain %q", tc.value, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestParseAny_Success(t *testing.T) {
	got, err := dt.ParseAny("2023-01-15T12:00:00Z")
	if err != nil {
		t.Fatalf("ParseAny err = %v", err)
	}
	want := time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ParseAny() = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// Options: DateOnly, TimeOnly, ToTimezone, FormatAs
// ---------------------------------------------------------------------------

func TestOption_DateOnly(t *testing.T) {
	got := dt.New(refISO, dt.DateOnly())
	if got != "2023-01-15" {
		t.Errorf("DateOnly() = %q, want %q", got, "2023-01-15")
	}
}

func TestOption_TimeOnly(t *testing.T) {
	got := dt.New(refISO, dt.TimeOnly())
	if got != "12:00:00" {
		t.Errorf("TimeOnly() = %q, want %q", got, "12:00:00")
	}
}

func TestOption_DateOnlyPlusTimeOnly_IsEmpty(t *testing.T) {
	// BUG-10: mutually exclusive, must return empty string when both set.
	got := dt.New(refISO, dt.DateOnly(), dt.TimeOnly())
	if got != "" {
		t.Errorf("DateOnly+TimeOnly = %q, want empty (mutually exclusive)", got)
	}
}

func TestOption_DateOnlyHonorsExplicitFormat(t *testing.T) {
	got := dt.New(refISO, dt.DateOnly(), dt.FormatAs(dt.Custom("DD/MM/YYYY")))
	if got != "15/01/2023" {
		t.Errorf("DateOnly+Custom = %q, want %q", got, "15/01/2023")
	}
}

func TestOption_ToTimezone_Conversion(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	// 12:00 UTC is 07:00 EST (-05:00) on Jan 15.
	got := dt.New(refISO, dt.ToTimezone(ny))
	if got != "2023-01-15T07:00:00-05:00" {
		t.Errorf("ToTimezone(NY) = %q, want %q", got, "2023-01-15T07:00:00-05:00")
	}
}

func TestOption_ToTimezone_NilIsNoOp(t *testing.T) {
	// BUG-13: nil location must not panic and must be equivalent to no option.
	got := dt.New(refISO, dt.ToTimezone(nil))
	if got != "2023-01-15T12:00:00Z" {
		t.Errorf("ToTimezone(nil) = %q, want %q", got, "2023-01-15T12:00:00Z")
	}
}

func TestOption_DateOnly_TimezoneDoesNotRollDate(t *testing.T) {
	// BUG-2: v0.1.0 zeroed the clock in the local zone, then formatted via
	// UTC, rolling the date across a DST or offset boundary. Kiritimati is
	// +14:00 — at 06:00Z on Jan 1 the local date is 20:00 on Jan 1, and the
	// DateOnly output must be 2023-01-01 (not the old-bug value 2022-12-31).
	k, err := time.LoadLocation("Pacific/Kiritimati")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	got := dt.New("2023-01-01T06:00:00Z", dt.ToTimezone(k), dt.DateOnly())
	if got != "2023-01-01" {
		t.Errorf("DateOnly+Kiritimati = %q, want %q", got, "2023-01-01")
	}
}

func TestOption_FormatAs_Unix(t *testing.T) {
	got := dt.New(refISO, dt.FormatAs(dt.Unix()))
	if got != "1673784000000" {
		t.Errorf("FormatAs(Unix) = %q, want %q", got, "1673784000000")
	}
}

func TestOption_FormatAs_ISOPreservesOffset(t *testing.T) {
	// BUG-24: ISO() must use time.RFC3339 and preserve the offset, not force
	// UTC with a literal "Z".
	got := dt.New("2023-01-15T12:00:00-05:00", dt.FormatAs(dt.ISO()))
	if got != "2023-01-15T12:00:00-05:00" {
		t.Errorf("ISO() on -05:00 input = %q, want %q", got, "2023-01-15T12:00:00-05:00")
	}
}

// ---------------------------------------------------------------------------
// Custom pattern tokenizer — BUG-1 regression battery
// ---------------------------------------------------------------------------

func TestCustomPattern_LiteralWordsPreserved(t *testing.T) {
	// Each case below mangled to garbage under the v0.1.0 cascading
	// ReplaceAll implementation.
	tests := []struct {
		pattern string
		want    string
	}{
		{"Month: MM", "Month: 01"},
		{"Hour: HH", "Hour: 12"},
		{"MMMM DD, YYYY", "January 15, 2023"},
		{"HH:mm:ss.SSS", "12:00:00.000"},
		{"Today is MMMM DD", "Today is January 15"},
		{"[YYYY literal stays]", "[2023 literal stays]"},
		{"DD-MM-YYYY HH:mm:ss", "15-01-2023 12:00:00"},
		{"MMM D, YYYY", "Jan 15, 2023"},
	}
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			got := dt.New(refISO, dt.FormatAs(dt.Custom(tc.pattern)))
			if got != tc.want {
				t.Errorf("Custom(%q) = %q, want %q", tc.pattern, got, tc.want)
			}
		})
	}
}

func TestCustomPattern_FractionalSeconds(t *testing.T) {
	// Input has 123ms. v0.1.0 leaked "SSS" unchanged.
	got := dt.New("2023-01-15T12:00:00.123Z", dt.FormatAs(dt.Custom("HH:mm:ss.SSS")))
	if got != "12:00:00.123" {
		t.Errorf("Custom(HH:mm:ss.SSS) = %q, want %q", got, "12:00:00.123")
	}
}

func TestCustomPattern_BareMillis(t *testing.T) {
	// SSS without a leading dot emits zero-padded millis as three digits.
	got := dt.New("2023-01-15T12:00:00.007Z", dt.FormatAs(dt.Custom("HHmmssSSS")))
	if got != "120000007" {
		t.Errorf("Custom(HHmmssSSS) = %q, want %q", got, "120000007")
	}
}

func TestCustomPattern_Empty(t *testing.T) {
	got := dt.New(refISO, dt.FormatAs(dt.Custom("")))
	if got != "" {
		t.Errorf("Custom(\"\") = %q, want empty string", got)
	}
}

func TestCustomPattern_AllLiteral(t *testing.T) {
	got := dt.New(refISO, dt.FormatAs(dt.Custom("hello world")))
	if got != "hello world" {
		t.Errorf("Custom(\"hello world\") = %q, want literal", got)
	}
}

// ---------------------------------------------------------------------------
// Ordinal day (DDD) patterns
// ---------------------------------------------------------------------------

func TestCustomPattern_OrdinalDay(t *testing.T) {
	tests := []struct {
		pattern  string
		input    string
		want     string
	}{
		{"MMM DDD, YYYY", "2023-01-01", "Jan 1st, 2023"},
		{"YYYY-MM-DDD", "2023-01-02", "2023-01-2nd"},
		{"DDD MMM YYYY", "2023-01-03", "3rd Jan 2023"},
		{"MMM DDD", "2023-01-04", "Jan 4th"},
		{"DDD/MM/YYYY", "2023-01-21", "21st/01/2023"},
		{"DDD-MM-YY", "2023-01-22", "22nd-01-23"},
		{"DDD of MMM", "2023-01-23", "23rd of Jan"},
		{"DDD MMM", "2023-01-31", "31st Jan"},
		// Teens edge: 11, 12, 13 all take "th", not "st/nd/rd"
		{"DDD", "2023-01-11", "11th"},
		{"DDD", "2023-01-12", "12th"},
		{"DDD", "2023-01-13", "13th"},
	}
	for _, tc := range tests {
		t.Run(tc.pattern+"/"+tc.input, func(t *testing.T) {
			got := dt.New(tc.input, dt.FormatAs(dt.Custom(tc.pattern)))
			if got != tc.want {
				t.Errorf("Custom(%q) on %q = %q, want %q", tc.pattern, tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ordinal day parsing (input side)
// ---------------------------------------------------------------------------

func TestParse_OrdinalInputs(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"jan 4th 25", time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)},
		{"feb 21st 2025", time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC)},
		{"mar 2nd 2025", time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)},
		{"apr 3rd 2025", time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC)},
		{"may 11th 2025", time.Date(2025, 5, 11, 0, 0, 0, 0, time.UTC)},
		{"dec 31st 2025", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
		{"january 4th 2025", time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)},
		{"Jan 4TH 25", time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)},
		// Mixed case: the regex-based approach handles any case combination.
		{"Jan 2nD 2025", time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := dt.Parse(tc.input)
			if !got.Equal(tc.want) {
				t.Errorf("Parse(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// String/int parity + wider numeric type support
// ---------------------------------------------------------------------------

func TestParityStringVsInt64(t *testing.T) {
	// BUG-4: v0.1.0 rejected 9-digit numeric strings while accepting them as
	// int64. The two paths must agree.
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"str 9 digit", "167257440", "1975-04-20T20:24:00Z"},
		{"int64 9 digit", int64(167257440), "1975-04-20T20:24:00Z"},
		{"str 10 digit", "1673784000", "2023-01-15T12:00:00Z"},
		{"int64 10 digit", int64(1673784000), "2023-01-15T12:00:00Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dt.New(tc.v)
			if got != tc.want {
				t.Errorf("New(%v) = %q, want %q", tc.v, got, tc.want)
			}
		})
	}
}

func TestAllIntegerWidths(t *testing.T) {
	// BUG-9: v0.1.0 default branch silently dropped uint/int32/etc.
	const want = "2000-01-01T00:00:00Z"
	sec := int64(946684800) // 2000-01-01T00:00:00Z
	cases := []struct {
		name string
		v    any
	}{
		{"int", int(sec)},
		{"int32", int32(sec)},
		{"int64", sec},
		{"uint32", uint32(sec)},
		{"uint64", uint64(sec)},
		{"uint", uint(sec)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dt.New(tc.v)
			if got != want {
				t.Errorf("New(%s %v) = %q, want %q", tc.name, tc.v, got, want)
			}
		})
	}
}

func TestFloatNaNAndInfRejected(t *testing.T) {
	// NaN must not silently produce a garbage time.Time. Every comparison
	// with NaN is false so the range checks in parseUnixFloat would fall
	// through to int64(NaN) (typically MinInt64 on amd64) without an
	// explicit IsNaN guard. Same reasoning for ±Inf.
	bad := []float64{
		math.NaN(),
		math.Inf(1),
		math.Inf(-1),
	}
	for _, v := range bad {
		if got := dt.Parse(v); !got.IsZero() {
			t.Errorf("Parse(%v) = %v, want zero time", v, got)
		}
		if got := dt.New(v); got != "" {
			t.Errorf("New(%v) = %q, want empty string", v, got)
		}
		if _, err := dt.ParseAny(v); err == nil {
			t.Errorf("ParseAny(%v) err = nil, want non-nil", v)
		}
	}
	// float32 NaN path must also be caught (it widens to float64 NaN).
	if got := dt.Parse(float32(math.NaN())); !got.IsZero() {
		t.Errorf("Parse(float32 NaN) = %v, want zero time", got)
	}
}

func TestFloatSubSecondPrecision(t *testing.T) {
	// BUG-7: v0.1.0 did int64(v), discarding the fractional second.
	got := dt.Parse(float64(1673784000.999))
	// Allow a few nanoseconds of float slack but assert millisecond precision.
	wantMs := int64(1673784000999)
	if gotMs := got.UnixMilli(); gotMs != wantMs {
		t.Errorf("Parse(float with .999) UnixMilli = %d, want %d", gotMs, wantMs)
	}
}

// ---------------------------------------------------------------------------
// RFC3339Nano / length cap
// ---------------------------------------------------------------------------

func TestParse_RFC3339Nano(t *testing.T) {
	// BUG-3: v0.1.0 rejected inputs > 30 chars and had no nanosecond layout.
	input := "2023-01-02T15:04:05.123456789-07:00"
	got := dt.Parse(input)
	if got.IsZero() {
		t.Fatalf("Parse(%q) returned zero time", input)
	}
	if got.Nanosecond() != 123456789 {
		t.Errorf("nanoseconds = %d, want 123456789", got.Nanosecond())
	}
	_, offset := got.Zone()
	if offset != -7*3600 {
		t.Errorf("offset = %d, want %d", offset, -7*3600)
	}
}

// ---------------------------------------------------------------------------
// Legacy API regression tests
// ---------------------------------------------------------------------------

func TestLegacyParseDatetime_RangeAligned(t *testing.T) {
	// BUG-6: v0.1.0 ParseDatetime accepted up to 14 digits while
	// parseUnixTimestamp capped at 13. They must agree now.
	if _, err := dt.ParseDatetime(int64(99999999999999), "unix", ""); err == nil {
		t.Error("ParseDatetime(14-digit int64) = nil, want range error")
	}
	if _, err := dt.ParseDatetime(int64(9999999999999), "unix", ""); err != nil {
		t.Errorf("ParseDatetime(13-digit int64) err = %v, want nil", err)
	}
}

func TestLegacyFormatDatetime_UnknownFormatTreatedAsGoLayout(t *testing.T) {
	// BUG-5: v0.1.0 silently returned Unix millis for unknown format codes.
	// Now, if the format contains no dt pattern tokens, it is treated as a
	// raw Go time layout.
	got, err := dt.FormatDatetime(1673784000000, "2006-01-02", "")
	if err != nil {
		t.Fatalf("FormatDatetime err = %v", err)
	}
	if got != "2023-01-15" {
		t.Errorf("FormatDatetime(Go layout) = %q, want %q", got, "2023-01-15")
	}
}

func TestLegacyFormatDatetime_CustomWithDTPattern(t *testing.T) {
	got, err := dt.FormatDatetime(1673784000000, "custom", "YYYY-MM-DD")
	if err != nil {
		t.Fatalf("FormatDatetime err = %v", err)
	}
	if got != "2023-01-15" {
		t.Errorf("FormatDatetime(custom YYYY-MM-DD) = %q, want %q", got, "2023-01-15")
	}
}

func TestLegacyFormatDatetime_CustomWithGoLayout(t *testing.T) {
	// v0.1.0 callers passed Go layouts as customPattern; that path is
	// preserved for source compatibility.
	got, err := dt.FormatDatetime(1673784000000, "custom", "2006-01-02")
	if err != nil {
		t.Fatalf("FormatDatetime err = %v", err)
	}
	if got != "2023-01-15" {
		t.Errorf("FormatDatetime(custom 2006-01-02) = %q, want %q", got, "2023-01-15")
	}
}

func TestLegacyFormatDatetime_CustomEmptyPatternErrors(t *testing.T) {
	_, err := dt.FormatDatetime(1673784000000, "custom", "")
	if err == nil {
		t.Error("FormatDatetime(custom, empty) err = nil, want error")
	}
}

func TestLegacyDetectDatetimeFormat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1673784000", "unix"},
		{"1673784000000", "unix"},
		{"2023-01-15T12:00:00Z", "iso"},
		{"2023-01-15T12:00:00.123Z", "iso"},
		{"2023-01-15T12:00:00-05:00", "iso-tz"},
		{"2023-01-15T12:00:00.123-05:00", "iso-tz"},
		{"2023-01-15", "date"},
		{"01/15/2023", "custom"},
		{"", "custom"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := dt.DetectDatetimeFormat(tc.input)
			if got != tc.want {
				t.Errorf("DetectDatetimeFormat(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestLegacyIsValidDatetimeFormat(t *testing.T) {
	// BUG-8: v0.1.0 returned true for everything except literal "invalid".
	tests := []struct {
		format string
		want   bool
	}{
		{"unix", true},
		{"iso", true},
		{"iso-tz", true},
		{"date", true},
		{"custom", true},
		{"YYYY-MM-DD", true},
		{"DDD of MMM", true},
		// Junk with no recognized tokens:
		{"lolwut", false},
		{"!!!", false},
		{"invalid", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			got := dt.IsValidDatetimeFormat(tc.format)
			if got != tc.want {
				t.Errorf("IsValidDatetimeFormat(%q) = %v, want %v", tc.format, got, tc.want)
			}
		})
	}
}

func TestLegacyParseDatetime_Cases(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		format  string
		pattern string
		wantErr bool
	}{
		{"int64 timestamp", int64(1673784000), "unix", "", false},
		{"float64 timestamp", float64(1673784000.5), "unix", "", false},
		{"int timestamp", int(1673784000), "unix", "", false},
		{"ISO string", "2023-01-15T12:00:00Z", "iso", "", false},
		{"date string", "2023-01-15", "date", "", false},
		{"custom format label", "01/15/2023", "custom", "", false},
		{"unsupported type", []int{1, 2, 3}, "custom", "", true},
		{"negative timestamp", int64(-1), "unix", "", true},
		{"time.Time input", time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC), "iso", "", false},
		{"zero time.Time rejected", time.Time{}, "iso", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := dt.ParseDatetime(tc.value, tc.format, tc.pattern)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseDatetime(%v) err = nil, want error", tc.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDatetime(%v) unexpected err: %v", tc.value, err)
			}
			if result.UnixMillis <= 0 {
				t.Errorf("ParseDatetime(%v) UnixMillis = %d, want > 0", tc.value, result.UnixMillis)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Deprecated wrappers still work (source compat for v0.1.0 callers)
// ---------------------------------------------------------------------------

func TestDeprecated_NewDatetimeWrapsNew(t *testing.T) {
	got := dt.NewDatetime(refISO)
	want := dt.New(refISO)
	if got != want {
		t.Errorf("NewDatetime = %q, New = %q (should match)", got, want)
	}
}

func TestDeprecated_ToDatetime(t *testing.T) {
	got := dt.ToDatetime("2023-01-15T12:00:00Z")
	if got.IsZero() {
		t.Error("ToDatetime returned zero time for valid input")
	}
}

func TestDeprecated_ParseTimeString(t *testing.T) {
	if _, err := dt.ParseTimeString("2023-01-15T12:00:00Z"); err != nil {
		t.Errorf("ParseTimeString err = %v", err)
	}
	if _, err := dt.ParseTimeString("not a date"); err == nil {
		t.Error("ParseTimeString err = nil, want error")
	}
}

// ---------------------------------------------------------------------------
// Junk-rejection: strings with trailing codes should not parse
// ---------------------------------------------------------------------------

func TestIsDatetime_TrailingJunkRejected(t *testing.T) {
	rejected := []string{
		"12/31/25 m353",
		"01/15/23 X123",
		"2023-01-01 A1B2",
		"12/31/25 12345",
		"01/15/23 123456",
		"12/31/25 A",
		"01/15/23 AB",
	}
	for _, input := range rejected {
		t.Run(input, func(t *testing.T) {
			if got := dt.New(input); got != "" {
				t.Errorf("New(%q) = %q, want empty (trailing junk)", input, got)
			}
		})
	}
}

func TestIsDatetime_ShortYearsAccepted(t *testing.T) {
	accepted := []string{
		"jan 4 25",
		"jan 4 2025",
		"jan 15 2025",
	}
	for _, input := range accepted {
		t.Run(input, func(t *testing.T) {
			if got := dt.New(input); got == "" {
				t.Errorf("New(%q) = empty, want parsed value", input)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Round trip
// ---------------------------------------------------------------------------

func TestRoundTrip(t *testing.T) {
	cases := []string{
		"2024-01-15T10:30:00Z",
		"2024-01-15",
		"1705320600000",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			parsed := dt.Parse(input)
			if parsed.IsZero() {
				t.Fatalf("Parse(%q) returned zero time", input)
			}
			formatted := dt.New(parsed)
			// The round-trip should be idempotent for time.Time inputs.
			again := dt.Parse(formatted)
			if !again.Equal(parsed) {
				t.Errorf("round-trip mismatch: %v → %q → %v", parsed, formatted, again)
			}
		})
	}
}
