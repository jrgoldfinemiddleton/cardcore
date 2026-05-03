# cardcore

A universal card game engine in [Go](https://go.dev/).

[![CI](https://github.com/jrgoldfinemiddleton/cardcore/actions/workflows/main.yml/badge.svg)](https://github.com/jrgoldfinemiddleton/cardcore/actions/workflows/main.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jrgoldfinemiddleton/cardcore.svg)](https://pkg.go.dev/github.com/jrgoldfinemiddleton/cardcore)
[![Go Report Card](https://goreportcard.com/badge/github.com/jrgoldfinemiddleton/cardcore)](https://goreportcard.com/report/github.com/jrgoldfinemiddleton/cardcore)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## About

Cardcore is an engine for building card games with an API-first design. It is intended to serve as a backend for CLI, web, desktop, and mobile clients. The library uses only the Go standard library to maintain minimal dependencies. It is designed to support games like Hearts, Poker, Tiến Lên, and others.

## Design Philosophy

- Minimal: abstractions are deferred until they become necessary.
- Zero runtime dependencies: standard library only.
- API-first: the engine exposes an API; clients are thin.
- [Suckless](https://suckless.org/philosophy/) code design: small, readable, and composable.
- Contributor-friendly: thorough docs, automated checks, and clear conventions lower the barrier to entry.

## Project Layout

```text
cardcore/
├── card.go      # Deck, Card, Suit, and Rank primitives
├── hand.go      # Hand — a player's cards
└── games/
    ├── <game1>/ # Game-specific logic (e.g., Hearts)
    └── <game2>/ # Another game (e.g., Tiến Lên)
```

## Requirements

Go 1.25.9+ (uses `sync.WaitGroup.Go`; dev tools managed via the `tool` directive)

## Getting Started

```bash
make check
```

This runs formatting, vetting, linting, and tests. See [Makefile Targets](#makefile-targets) for individual targets.

Dev tools like [golangci-lint](https://golangci-lint.run/) and [pkgsite](https://pkg.go.dev/golang.org/x/pkgsite) are declared in `go.mod` via the Go 1.25 `tool` directive and are compiled automatically on first use — no manual installation required.

## Makefile Targets

| Target | Description |
|---|---|
| `make test` | Run all tests |
| `make fmt` | Format code with [gofmt](https://pkg.go.dev/cmd/gofmt) |
| `make vet` | Run [go vet](https://pkg.go.dev/cmd/vet) |
| `make lint` | Run [golangci-lint](https://golangci-lint.run/) |
| `make build` | Compile all packages |
| `make doc` | Browse docs locally via [pkgsite](https://pkg.go.dev/golang.org/x/pkgsite) |
| `make check` | Run fmt, vet, lint, and test |
| `make bench` | Run all benchmarks |
| `make stats` | Run AI statistical profiles |
| `make help` | Show available targets |

## License

MIT
