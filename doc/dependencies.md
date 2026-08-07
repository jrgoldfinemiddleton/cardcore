# Approved Dependencies

This document lists the external dependencies approved for use in `cardcore`.
The engine itself uses only the Go standard library — no external runtime
dependencies (see [ADR-003](decisions/003-language-choice.md)), enforced by
`TestNoExternalDeps` in `convention_test.go`. New dependencies require
discussion and explicit approval before introduction.

## Runtime

None — standard library only.

## Dev Tools

| Module | Purpose | License |
|---|---|---|
| `github.com/golangci/golangci-lint` | Linter aggregator (via `go tool`) | GPL-3.0 |
| `golang.org/x/perf` | `benchstat` benchmark comparison (via `go tool`) | BSD-3-Clause |
| `golang.org/x/pkgsite` | Local documentation browser (via `go tool`) | BSD-3-Clause |
| `golang.org/x/vuln` | `govulncheck` vulnerability scanner (via `go tool`) | BSD-3-Clause |
