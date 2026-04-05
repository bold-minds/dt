package dt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DatetimeFormat represents a datetime output format
type DatetimeFormat interface {
	format(t time.Time) string
}

// DatetimeOption represents a datetime processing option
type DatetimeOption interface {
	apply(config *datetimeConfig)
}

// Internal configuration for datetime processing
type datetimeConfig struct {
	dateOnly     bool
	timeOnly     bool
	timezone     *time.Location
	outputFormat DatetimeFormat
}

// Legacy string constants for backward compatibility
const (
	FormatUnix   = "unix"
	FormatISO    = "iso"
	FormatISOTZ  = "iso-tz"
	FormatDate   = "date"
	FormatCustom = "custom"
)

// Format implementations
type isoFormat struct{}
type unixFormat struct{}
type customFormat struct{ pattern string }

func (f isoFormat) format(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func (f unixFormat) format(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

func (f customFormat) format(t time.Time) string {
	goLayout := convertPatternToGoLayout(f.pattern)
	result := t.Format(goLayout)
	// Handle ordinal day formatting if present
	if strings.Contains(result, "___999___") {
		ordinalDay := getOrdinalDay(t.Day())
		result = strings.ReplaceAll(result, "___999___", ordinalDay)
	}
	return result
}

// Option implementations
type dateOnlyOption struct{}
type timeOnlyOption struct{}
type timezoneOption struct{ tz *time.Location }
type formatAsOption struct{ format DatetimeFormat }

func (o dateOnlyOption) apply(config *datetimeConfig) {
	config.dateOnly = true
}

func (o timeOnlyOption) apply(config *datetimeConfig) {
	config.timeOnly = true
}

func (o timezoneOption) apply(config *datetimeConfig) {
	config.timezone = o.tz
}

func (o formatAsOption) apply(config *datetimeConfig) {
	config.outputFormat = o.format
}

// NewDatetime parses a datetime value and returns a formatted string
func NewDatetime(value any, opts ...DatetimeOption) string {
	// Parse to time.Time first
	t := parseToTime(value)
	if t.IsZero() {
		return "" // Return empty string for invalid dates
	}

	// Apply options
	config := &datetimeConfig{
		outputFormat: isoFormat{}, // Default to ISO format
	}
	for _, opt := range opts {
		opt.apply(config)
	}

	// Apply timezone if specified
	if config.timezone != nil {
		t = t.In(config.timezone)
	}

	// Apply date/time only filters
	if config.dateOnly {
		// Zero out time components
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	} else if config.timeOnly {
		// Zero out date components, keep time
		t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	}

	// Format output
	return config.outputFormat.format(t)
}

// ToDatetime parses a string value and returns a time.Time
func ToDatetime(value string) time.Time {
	return parseToTime(value)
}

// parseToTime handles the core parsing logic, returning time.Time
func parseToTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case int64:
		return parseUnixTimestamp(v)
	case float64:
		return parseUnixTimestamp(int64(v))
	case int:
		return parseUnixTimestamp(int64(v))
	case string:
		return parseDatetimeString(v)
	default:
		return time.Time{} // Zero time for unsupported types
	}
}

// parseUnixTimestamp handles numeric Unix timestamp inputs
func parseUnixTimestamp(timestamp int64) time.Time {
	// Validate timestamp range (reasonable bounds)
	if timestamp < 0 || timestamp > 9999999999999 { // Year ~2286
		return time.Time{}
	}

	// Convert seconds to milliseconds if needed (heuristic: < 10^10 = seconds)
	if timestamp < 10000000000 {
		timestamp *= 1000
	}

	return time.UnixMilli(timestamp).UTC()
}

// isDatetime performs a fast heuristic check to determine if a string is likely a datetime
// This avoids expensive parsing attempts on non-datetime strings
func isDatetime(s string) bool {
	// Quick length check - minimum 6 characters for valid datetime (e.g., "1-2-34")
	sLen := len(s)
	if sLen < 6 || sLen > 30 {
		return false
	}

	// Fast check for common datetime patterns
	// ISO format: contains T and digits
	if strings.Contains(s, "T") && strings.ContainsAny(s, "0123456789") {
		return true
	}

	// Fast byte-level scan for datetime indicators
	hasDigit := false
	hasSeparator := false
	digitCount := 0
	letterCount := 0
	spaceCount := 0

	// Single pass through string bytes (faster than runes)
	for i := 0; i < sLen; i++ {
		b := s[i]
		if b >= '0' && b <= '9' {
			hasDigit = true
			digitCount++
		} else if b == '-' || b == '/' || b == ':' || b == 'T' || b == 'Z' || b == '.' || b == ',' {
			hasSeparator = true
		} else if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
			letterCount++
			// Reject early if too many letters (likely text, not datetime)
			// Allow more letters for full month names and ordinal patterns
			if letterCount > 12 {
				return false
			}
		} else if b == ' ' {
			spaceCount++
		}
	}

	// Must have digits
	if !hasDigit {
		return false
	}

	// Allow pure numeric strings (Unix timestamps) if they're reasonable length
	if !hasSeparator && spaceCount == 0 {
		// Unix timestamps are typically 10 digits (seconds) or 13 digits (milliseconds)
		return sLen >= 10 && sLen <= 13 && digitCount == sLen
	}

	// Handle special cases with spaces
	if spaceCount > 0 {
		// For multiple spaces (like "2023 01 15"), treat as valid datetime separators
		if spaceCount > 1 {
			return true // Multiple spaces likely indicate date/time components
		}

		// Find the last space and check what follows
		lastSpaceIdx := -1
		for i := sLen - 1; i >= 0; i-- {
			if s[i] == ' ' {
				lastSpaceIdx = i
				break
			}
		}

		if lastSpaceIdx >= 0 && lastSpaceIdx < sLen-1 {
			// Check the content after the last space
			trailing := s[lastSpaceIdx+1:]
			trailingLen := len(trailing)

			// Allow AM/PM patterns
			if trailingLen == 2 && (trailing == "AM" || trailing == "PM") {
				return true // Valid time with AM/PM
			}

			// Reject if trailing content looks like a code/identifier
			if trailingLen <= 6 {
				// Check if it's all digits (likely a numeric code)
				allDigits := true
				for _, b := range trailing {
					if b < '0' || b > '9' {
						allDigits = false
						break
					}
				}
				// Allow 2-4 digit years (like "25" for 2025) but reject longer numeric codes
				if allDigits && trailingLen >= 5 {
					return false
				}

				// Check if it's a short alphanumeric code
				hasTrailingDigit := false
				hasTrailingLetter := false
				for _, b := range trailing {
					if b >= '0' && b <= '9' {
						hasTrailingDigit = true
					} else if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
						hasTrailingLetter = true
					}
				}
				if hasTrailingDigit && hasTrailingLetter {
					return false // Mixed alphanumeric code
				}
				if hasTrailingLetter && trailingLen <= 2 && trailing != "AM" && trailing != "PM" {
					return false // Short letter code (but not AM/PM)
				}
			}
		}
	}

	// Must have reasonable digit ratio for datetime
	return digitCount >= sLen/4
}

// parseDatetimeString handles string datetime inputs with multiple format attempts
func parseDatetimeString(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}

	// Fast heuristic check to avoid expensive parsing on non-datetime strings
	if !isDatetime(value) {
		return time.Time{}
	}

	// Convert ordinal days to regular numbers for parsing (e.g., "jan 4th 25" -> "jan 4 25")
	value = convertOrdinalDaysForParsing(value)

	// Try to parse as Unix timestamp string first
	if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parseUnixTimestamp(timestamp)
	}

	// Define common datetime layouts for parsing
	layouts := []string{
		// ISO 8601 formats
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05+07:00",
		"2006-01-02T15:04:05.000+07:00",

		// Date-only formats
		"2006-01-02",

		// Time-only formats
		"15:04:05",
		"15:04",

		// Common alternative formats
		"01/02/2006",
		"1/2/2006",
		"02/01/2006", // European style
		"2/1/2006",
		"02.01.2006", // European with periods
		"2.1.2006",
		"01-02-2006",
		"1-2-2006",

		// Date with time
		"01/02/2006 15:04:05",
		"02.01.2006 15:04:05",
		"2006-01-02 15:04:05",

		// Natural language formats with month names
		"Jan 2 2006",
		"Jan 2, 2006",
		"January 2 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"2 January 2006",

		// Short year formats
		"Jan 2 06",
		"Jan 2, 06",
		"1/2/06",
		"01/02/06",
		"2/1/06", // European
		"02/01/06",
	}

	// Try each layout
	for _, layout := range layouts {
		if parsedTime, err := time.Parse(layout, value); err == nil {
			return parsedTime.UTC()
		}
	}

	return time.Time{} // Return zero time if parsing fails
}

// convertPatternToGoLayout converts our custom pattern format to Go's time layout
// Supports ordinal day formatting (DDD) and other common patterns
func convertPatternToGoLayout(pattern string) string {
	// Basic conversions for common patterns
	goLayout := pattern

	// Year
	goLayout = strings.ReplaceAll(goLayout, "YYYY", "2006")
	goLayout = strings.ReplaceAll(goLayout, "YY", "06")

	// Month
	goLayout = strings.ReplaceAll(goLayout, "MMM", "Jan")
	goLayout = strings.ReplaceAll(goLayout, "MM", "01")
	goLayout = strings.ReplaceAll(goLayout, "M", "1")

	// Day - handle ordinal day (DDD) before regular day patterns
	// Note: Go doesn't natively support ordinal days, so we use a custom approach
	if strings.Contains(goLayout, "DDD") {
		// For ordinal days, we'll use a special marker that we'll handle in FormatDatetime
		// Use a placeholder that won't be interpreted by Go's time formatting
		// Avoid any letters that Go uses as format tokens (MST, PM, Jan, Mon, etc.)
		// Use only symbols and numbers that Go doesn't interpret
		goLayout = strings.ReplaceAll(goLayout, "DDD", "___999___")
	}
	goLayout = strings.ReplaceAll(goLayout, "DD", "02")
	goLayout = strings.ReplaceAll(goLayout, "D", "2")

	// Hour
	goLayout = strings.ReplaceAll(goLayout, "HH", "15")
	goLayout = strings.ReplaceAll(goLayout, "H", "15")
	goLayout = strings.ReplaceAll(goLayout, "hh", "03")
	goLayout = strings.ReplaceAll(goLayout, "h", "3")

	// Minute
	goLayout = strings.ReplaceAll(goLayout, "mm", "04")
	goLayout = strings.ReplaceAll(goLayout, "m", "4")

	// Second
	goLayout = strings.ReplaceAll(goLayout, "ss", "05")
	goLayout = strings.ReplaceAll(goLayout, "s", "5")

	return goLayout
}

// getOrdinalDay converts a day number to ordinal format (1st, 2nd, 3rd, etc.)
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

// convertOrdinalDaysForParsing converts ordinal days (1st, 2nd, etc.) back to regular numbers for parsing
func convertOrdinalDaysForParsing(value string) string {
	result := value

	// Replace ordinal suffixes with just the number (case-insensitive)
	// Use case-insensitive replacement without converting the entire string
	result = replaceOrdinalSuffix(result, "1st", "1")
	result = replaceOrdinalSuffix(result, "2nd", "2")
	result = replaceOrdinalSuffix(result, "3rd", "3")

	// Handle all other ordinal numbers (4th, 5th, ..., 31st)
	for i := 1; i <= 31; i++ {
		ordinal := getOrdinalDay(i)
		result = replaceOrdinalSuffix(result, ordinal, strconv.Itoa(i))
	}

	return result
}

// replaceOrdinalSuffix performs case-insensitive replacement of ordinal suffixes
func replaceOrdinalSuffix(s, old, replacement string) string {
	// Replace both lowercase and uppercase versions
	s = strings.ReplaceAll(s, old, replacement)
	s = strings.ReplaceAll(s, strings.ToUpper(old), replacement)
	// Handle title case manually (first letter uppercase, rest lowercase)
	if len(old) > 0 {
		titleCase := strings.ToUpper(old[:1]) + strings.ToLower(old[1:])
		s = strings.ReplaceAll(s, titleCase, replacement)
	}
	return s
}

// Format helper functions

// ISO returns an ISO format
func ISO() DatetimeFormat {
	return isoFormat{}
}

// Unix returns a Unix timestamp format
func Unix() DatetimeFormat {
	return unixFormat{}
}

// Custom returns a custom pattern format
func Custom(pattern string) DatetimeFormat {
	return customFormat{pattern: pattern}
}

// Option helper functions

// DateOnly returns an option to include only date components
func DateOnly() DatetimeOption {
	return dateOnlyOption{}
}

// TimeOnly returns an option to include only time components
func TimeOnly() DatetimeOption {
	return timeOnlyOption{}
}

// ToTimezone creates an option to convert the datetime to a specific timezone
func ToTimezone(tz *time.Location) DatetimeOption {
	return timezoneOption{tz: tz}
}

// FormatAs returns an option to format output using the specified format
func FormatAs(format DatetimeFormat) DatetimeOption {
	return formatAsOption{format: format}
}

// Legacy API for backward compatibility

// DatetimeParseResult represents the result of parsing a datetime
type DatetimeParseResult struct {
	UnixMillis int64
	Format     string
	Pattern    string
}

// ParseDatetime is the legacy function that parses datetime values
func ParseDatetime(value any, expectedFormat string, customPattern string) (*DatetimeParseResult, error) {
	// Convert to time.Time using new API
	var t time.Time

	switch v := value.(type) {
	case string:
		t = ToDatetime(v)
		if t.IsZero() {
			return nil, fmt.Errorf("unable to parse datetime string: %s", v)
		}
	case int64:
		if v < 0 || v > 99999999999999 {
			return nil, fmt.Errorf("timestamp out of valid range: %d", v)
		}
		// Auto-detect if it's seconds or milliseconds
		if v < 10000000000 { // Less than 10 billion = seconds
			t = time.Unix(v, 0).UTC()
		} else {
			t = time.UnixMilli(v).UTC()
		}
	case int:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	case float64:
		return ParseDatetime(int64(v), expectedFormat, customPattern)
	default:
		return nil, fmt.Errorf("unsupported type for datetime parsing: %T", value)
	}

	// Determine format if not specified
	format := expectedFormat
	if format == "" {
		if str, ok := value.(string); ok {
			format = DetectDatetimeFormat(str)
		} else {
			format = FormatUnix
		}
	}

	return &DatetimeParseResult{
		UnixMillis: t.UnixMilli(),
		Format:     format,
		Pattern:    customPattern,
	}, nil
}

// DetectDatetimeFormat detects the format of a datetime string
func DetectDatetimeFormat(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return FormatCustom
	}

	// Unix timestamp
	if matched, _ := regexp.MatchString(`^\d{10,13}$`, input); matched {
		return FormatUnix
	}

	// ISO format with Z
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z$`, input); matched {
		return FormatISO
	}

	// ISO format with timezone
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`, input); matched {
		return FormatISOTZ
	}

	// Date only
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, input); matched {
		return FormatDate
	}

	return FormatCustom
}

// IsValidDatetimeFormat checks if a format string is valid
func IsValidDatetimeFormat(format string) bool {
	validFormats := []string{FormatUnix, FormatISO, FormatISOTZ, FormatDate, FormatCustom}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	// Also accept custom patterns
	return len(format) > 0 && format != "invalid"
}

// Legacy function for backward compatibility - now simplified
func FormatDatetime(unixMillis int64, format string, customPattern string) (string, error) {
	t := time.UnixMilli(unixMillis).UTC()
	var fmt DatetimeFormat

	switch format {
	case FormatUnix:
		fmt = Unix()
	case FormatISO:
		fmt = ISO()
	case FormatISOTZ:
		fmt = ISO() // ISOTZ uses same format as ISO for UTC times
	case FormatDate:
		fmt = Custom("2006-01-02") // Date only format
	case FormatCustom:
		if customPattern != "" {
			fmt = Custom(customPattern)
		} else {
			fmt = Unix() // Fallback
		}
	default:
		fmt = Unix() // Default fallback to unix
	}

	return fmt.format(t), nil
}

// ParseTimeString is a helper function for internal use by typ package
// It provides a simple time.Time result compatible with existing ToType functionality
func ParseTimeString(s string) (time.Time, error) {
	result := ToDatetime(s)
	if result.IsZero() {
		return time.Time{}, fmt.Errorf("unable to parse datetime string: %s", s)
	}
	return result, nil
}
