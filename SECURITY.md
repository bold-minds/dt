# Security Policy

## Supported Versions

We actively support the following versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability, please follow these steps:

### 1. **Do Not** Create a Public Issue

Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.

### 2. Report Privately

Send an email to **security@boldminds.tech** with the following information:

- **Subject**: Security Vulnerability in bold-minds/dt
- **Description**: Detailed description of the vulnerability
- **Steps to Reproduce**: Clear steps to reproduce the issue
- **Impact**: Potential impact and severity assessment
- **Suggested Fix**: If you have ideas for a fix (optional)

### 3. Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Resolution**: Varies based on complexity, typically within 30 days

### 4. Disclosure Process

1. We will acknowledge receipt of your vulnerability report
2. We will investigate and validate the vulnerability
3. We will develop and test a fix
4. We will coordinate disclosure timing with you
5. We will release a security update
6. We will publicly acknowledge your responsible disclosure (if desired)

## Security Considerations

`dt` is a parsing and formatting library with a bounded attack surface:

- **No network I/O.** `dt` does not make network calls.
- **No file I/O.** `dt` does not read or write files.
- **No reflection.** `dt` uses concrete type switches only.
- **No external dependencies.** `dt` is pure Go stdlib.
- **No mutation.** `dt` never modifies input values.

### Parser Resilience

`dt` parses datetime strings from potentially untrusted sources (user input, API payloads, config files). The parser is designed to degrade safely:

- **Never panics.** Malformed input returns a zero `time.Time` or an error, never a panic.
- **Rejects implausible Unix timestamps.** Values outside `[0, 9999999999999]` (year ~2286) are rejected rather than producing wildly out-of-range times.
- **Length-gated heuristic.** A fast pre-filter rejects strings shorter than 6 or longer than 30 characters before attempting any layout parse, preventing quadratic-time behavior on long non-datetime strings.
- **Regex-free on the hot path.** `parseDatetimeString` uses byte-level scanning for format detection, not compiled regexes, avoiding ReDoS concerns on untrusted input. The internal regexes used by `DetectDatetimeFormat` are bounded-size anchored patterns.

### Known Limitations

- **Ambiguous formats.** Several layouts overlap (e.g., `01/02/2006` could be US or European depending on intent). `dt` tries layouts in a fixed order; the first match wins. If your application is sensitive to locale, parse with an explicit layout via Go's `time.Parse` and skip `dt`.
- **No locale support.** Month names are English-only. Non-English month names will not parse.
- **Two-digit year interpretation.** `Jan 2 06` parses as year 2006 via Go's stdlib rule. If your input contains two-digit years from before 2000, normalize beforehand.

## Security Updates

Security updates will be:

- Released as patch versions (e.g., 0.1.1)
- Documented in the CHANGELOG.md
- Announced through GitHub releases
- Tagged with security labels

## Acknowledgments

We appreciate responsible disclosure and will acknowledge security researchers who help improve the security of this project.

Thank you for helping keep our project and users safe!
