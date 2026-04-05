package dt_test

import (
	"testing"

	"github.com/bold-minds/dt"
)

func Benchmark_NewDatetime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.NewDatetime("2023-01-01T12:00:00Z")
	}
}

func Benchmark_ToDatetime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = dt.ToDatetime("2023-01-01T12:00:00Z")
	}
}

func Benchmark_ParseDateTime_ISO(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseDatetime("2023-01-01T12:00:00Z", dt.FormatISO, "")
	}
}

func Benchmark_ParseDateTime_Custom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseDatetime("01/02/2023", dt.FormatCustom, "")
	}
}

func Benchmark_FormatDatetime_ISO(b *testing.B) {
	timestamp := int64(1672574400000)
	for i := 0; i < b.N; i++ {
		_, _ = dt.FormatDatetime(timestamp, dt.FormatISO, "")
	}
}

func Benchmark_ParseTimeString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = dt.ParseTimeString("2023-01-01T12:00:00Z")
	}
}
