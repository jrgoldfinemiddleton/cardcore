# cardcore

A universal card game engine in [Go](https://go.dev/).

<!-- badges go here -->

## About

Cardcore is an engine for building card games with an API-first design. It is intended to serve as a backend for CLI, web, desktop, and mobile clients. The library uses only the Go standard library to maintain minimal dependencies. It is designed to support games like Hearts, Poker, Tiến Lên, and others.

## Design Philosophy

- Minimal: abstractions are deferred until they become necessary.
- Zero runtime dependencies: standard library only.
- API-first: the engine exposes an API; clients are thin.
- [Suckless](https://suckless.org/philosophy/): small, readable, and composable.

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

Go 1.22+ (uses `math/rand/v2`)

## Getting Started

```bash
make check
```

This runs formatting, vetting, linting, and tests. See [Makefile Targets](#makefile-targets) for individual targets.

Note: both tools require `$(go env GOPATH)/bin` on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

`make check` requires `golangci-lint`:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

`make doc` requires `pkgsite`:

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest
```

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

## License

MIT
