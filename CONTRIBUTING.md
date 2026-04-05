# Contributing to `dt`

Thanks for your interest in contributing. This guide covers the operational process. For the **why** — the design principles every contribution is tested against — see **[bold-minds/oss/PRINCIPLES.md](https://github.com/bold-minds/oss/blob/main/PRINCIPLES.md)**.

## 🎯 Before You Start

Every contribution is measured against the four Bold Minds principles: **outcome naming**, **one way to do each thing**, **get out of the way**, and **non-goals explicit**. If your proposed change doesn't honor these, it will not be merged — not because the maintainers are precious, but because these principles are what make the libraries worth using.

**Read [PRINCIPLES.md](https://github.com/bold-minds/oss/blob/main/PRINCIPLES.md) first.** It's the load-bearing document.

## 🔧 Development Setup

**Requirements:** Go 1.21 or later, Git, Bash.

```bash
git clone https://github.com/bold-minds/dt.git
cd dt
go test ./...              # unit tests
go test -race ./...        # race detection
go test -bench=. ./...     # benchmarks
./scripts/validate.sh      # full validation pipeline (local mode)
./scripts/validate.sh ci   # strict CI mode
```

Your contribution must pass `./scripts/validate.sh ci` before submitting.

## 📁 Project Structure

```
dt/
├── dt.go                  # Implementation (single file)
├── dt_test.go             # Unit tests
├── bench_test.go          # Benchmarks
├── scripts/
│   └── validate.sh        # Validation pipeline
├── README.md
├── CONTRIBUTING.md        # This file
├── CHANGELOG.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
├── LICENSE
└── go.mod
```

Keep it flat. No `internal/` directory unless the library grows significantly.

## 🎨 Code Style

### Naming
- Outcome naming per PRINCIPLES.md. If you reach for a dispatcher name (`Apply`, `Mutate`, `Process`, `Handle`), stop and rename.

### Error Handling
- Parser functions **must not panic**. Malformed input returns a zero `time.Time` or an explicit error.
- `Or` variants take a fallback argument and return it on failure.
- **No `Must*` variants.**

### Documentation
- Every exported function has a doc comment starting with the function name, describing the outcome (not the implementation), and noting edge cases — especially which layouts are attempted and in what order.
- Package-level doc comment in `dt.go`.

### Dependencies
- **Zero external dependencies.** `dt` is pure stdlib (`fmt`, `regexp`, `strconv`, `strings`, `time`).

### Parser Changes
- Any change to the layout list, format-detection heuristic, or Unix-timestamp range must include:
  - A regression test with the exact input string
  - A rationale in the PR description for why the new case is worth the added ambiguity
- **Additive only.** Removing a supported layout is a breaking change and requires a major version bump.

## 🧪 Testing

**Coverage target: 100% of exported functions.**

```bash
go test -v ./...                   # verbose
go test -race ./...                # race detection
go test -cover ./...               # coverage
go test -bench=. -benchmem ./...   # benchmarks with allocations
```

- Table-driven tests preferred for parser-layout matrices
- Every exported function has a corresponding benchmark in `bench_test.go`
- Time-sensitive tests must use fixed reference times, never `time.Now()`

## 📝 Pull Request Process

### PR Checklist

Before submitting, verify your PR against the four principles:

- [ ] **Outcome naming** — does the function name describe what the caller gets?
- [ ] **One way** — does any existing function (this library or stdlib) already do this? If yes, stop.
- [ ] **Get out of the way** — can a Go dev use this from the signature alone?
- [ ] **Non-goals** — does this violate any of the library's stated non-goals?

Additionally:
- [ ] Tests cover 100% of new code
- [ ] Benchmarks added for new exported functions
- [ ] README updated (if adding or changing exported functions)
- [ ] CHANGELOG.md updated
- [ ] `./scripts/validate.sh ci` passes locally

### PR Scope
- **One function per PR** when adding new functionality
- Bug fixes can be grouped if they share a root cause
- Documentation-only changes can be batched

### PR Description Template

```
## What
One sentence describing the change.

## Why
Real-world evidence of the pain this solves. Link to code, open-source example,
or specific stdlib gap.

## Principles Check
- Outcome naming: [how the name passes the "say it aloud" test]
- One way: [verified no stdlib or existing function does this]
- Get out of the way: [signature alone is enough]
- Non-goals: [confirmed no non-goal violated]

## Breaking Changes
None / [describe]
```

## 🆕 Adding a New Function

`dt` covers parsing, formatting, and format detection. New additions are rare and must clear a high bar:

1. Read the library's non-goals in [README.md](README.md) and [PRINCIPLES.md](https://github.com/bold-minds/oss/blob/main/PRINCIPLES.md). If the function violates one, stop.
2. Apply the four-principles checklist above.
3. **Prove the stdlib gap.** Go's `time` package already covers parsing and formatting with explicit layouts. `dt`'s value is the auto-detection and any→time coercion — your function must meaningfully extend that, not duplicate `time.Parse`.
4. **Show real-world evidence.** Either a codebase using the pattern today, or a verifiable open-source example. Theoretical usefulness is not enough.
5. Draft the function signature and README section first. Open a discussion issue for feedback before writing implementation.
6. Implement, test, benchmark, document.
7. Submit PR with one function per PR.

## 🏷️ Versioning and Releases

- **Semantic versioning**: `vMAJOR.MINOR.PATCH`
- **v0.x**: API may change between minor versions (pre-1.0 signaling)
- **v1.0+**: breaking changes require a major version bump
- Every release updates `CHANGELOG.md`
- Releases are tagged in git and published via `go mod` automatically

## 🙏 Code of Conduct

See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## 📄 License

By contributing, you agree your contributions are licensed under the MIT License (see [LICENSE](LICENSE)).

## Questions?

Open a discussion issue in this repository.
