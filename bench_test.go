package dt_test

import (
	"testing"
	"time"

	"github.com/bold-minds/dt"
)

// ----- Canonical API --------------------------------------------------------

func BenchmarkNew_ISOString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.New("2023-01-15T12:00:00Z")
	}
}

func BenchmarkNew_UnixSeconds(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.New(int64(1673784000))
	}
}

func BenchmarkNew_WithOptions(b *testing.B) {
	ny, _ := time.LoadLocation("America/New_York")
	for i := 0; i < b.N; i++ {
		_ = dt.New("2023-01-15T12:00:00Z", dt.ToTimezone(ny), dt.DateOnly())
	}
}

func BenchmarkNew_CustomPattern(b *testing.B) {
	f := dt.Custom("MMMM DD, YYYY HH:mm:ss")
	for i := 0; i < b.N; i++ {
		_ = dt.New("2023-01-15T12:00:00Z", dt.FormatAs(f))
	}
}

func BenchmarkParse_ISO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.Parse("2023-01-15T12:00:00Z")
	}
}

func BenchmarkParse_RFC3339Nano(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.Parse("2023-01-02T15:04:05.123456789-07:00")
	}
}

func BenchmarkParseAny_Success(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseAny("2023-01-15T12:00:00Z")
	}
}

// ----- Reject paths and heuristics -----------------------------------------

// BenchmarkIsDatetime_RejectPath exercises the fast-reject heuristic in
// isDatetime against strings that look plausible but aren't datetimes.
// v0.1.0 had zero coverage here, even though it is the hottest early-exit
// path and the correctness of its rules dictates whether callers pay the
// layout-tryout cost.
func BenchmarkIsDatetime_RejectPath(b *testing.B) {
	rejects := []string{
		"",
		"hello",
		"12/31/25 m353",
		"2023-01-01 A1B2",
		"01/15/23 123456",
		"not-a-date-at-all",
	}
	for i := 0; i < b.N; i++ {
		_ = dt.New(rejects[i%len(rejects)])
	}
}

// BenchmarkParse_OrdinalInput covers the stripOrdinalSuffixes regex path.
func BenchmarkParse_OrdinalInput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.Parse("January 4th 2025")
	}
}

// ----- Custom pattern compilation ------------------------------------------

// BenchmarkCustom_Compile measures how fast a new Custom() constructor
// tokenizes its pattern. In v0.1.0 this conversion ran once per format call;
// now it runs once per constructor invocation and is cached in the struct.
func BenchmarkCustom_Compile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.Custom("MMMM DD, YYYY HH:mm:ss.SSS")
	}
}

// ----- Legacy API -----------------------------------------------------------

func BenchmarkLegacy_ParseDatetime_ISO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseDatetime("2023-01-15T12:00:00Z", dt.FormatISO, "")
	}
}

func BenchmarkLegacy_ParseDatetime_Custom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseDatetime("01/15/2023", dt.FormatCustom, "")
	}
}

func BenchmarkLegacy_FormatDatetime_ISO(b *testing.B) {
	ts := int64(1673784000000)
	for i := 0; i < b.N; i++ {
		_, _ = dt.FormatDatetime(ts, dt.FormatISO, "")
	}
}

func BenchmarkLegacy_DetectDatetimeFormat(b *testing.B) {
	inputs := []string{
		"1673784000",
		"2023-01-15T12:00:00Z",
		"2023-01-15T12:00:00-05:00",
		"2023-01-15",
		"01/15/2023",
	}
	for i := 0; i < b.N; i++ {
		_ = dt.DetectDatetimeFormat(inputs[i%len(inputs)])
	}
}

func BenchmarkLegacy_ParseTimeString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseTimeString("2023-01-15T12:00:00Z")
	}
}
