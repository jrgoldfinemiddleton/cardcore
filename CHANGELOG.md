# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added
- Tiến Lên package foundation (`games/tienlen/`): combination classification and comparison (singles, pairs, triples, four-of-a-kind, straights, pair sequences), the variant-gated chop relation with the Killer chop table wired (ADR-010), game-local rank/suit ordering (3 low … 2 high, ♠<♣<♦<♥), and `ErrWrongPhase`/`ErrOutOfTurn`/`ErrIllegalMove` sentinels

## [0.7.1] - 2026-08-20

### Added
- `scripts/prepare-release.sh` is now the only supported way to prepare a release's changelog section and its PR: it validates the repository, moves `[Unreleased]` into a dated `[X.Y.Z]` section (verified with the same extraction the release tooling uses), and opens the `docs(changelog)` PR with the fixed title and body. `doc/releasing.md` documents the script-based preparation flow. The release workflow also writes its extracted notes to `runner.temp` instead of the checkout, matching cardcore-server's workflow
- `scripts/release.sh` is now the only supported way to tag a release: it validates strict semver and the pre-1.0 policy, verifies the tree is on a clean, current `main`, requires a dated, non-empty changelog section with an empty `[Unreleased]` (using the same extraction as the release workflow), runs `make check`, and only then creates an annotated tag (`git tag -a`) and pushes it. `doc/releasing.md` documents the full maintainer release process, including post-release verification and recovery. All future tags are annotated; v0.1.0–v0.6.0 were lightweight
- `nakedret` linting: naked returns are now forbidden in every function regardless of length (`max-func-lines: 0`), so named result parameters serve purely as signature documentation and every return statement stays explicit
- `TestConstAndFieldDocComments` convention test: every exported const in a group (including iota groups) and every field of an exported struct must have a doc comment starting with its name; doc comments added across `card.go`, `hand.go`, `games/hearts/`, and `games/hearts/ai/`, with the rules documented in CONTRIBUTING.md
- Security scanning: `gosec` is enabled in the default lint config, and `govulncheck` runs locally and in CI via the new `make vuln` target
- Per-package `AGENTS.md` guides (`games/hearts/`, `games/hearts/ai/`, `doc/decisions/`) are now committed to the repository; `convention_test.go` (`TestAgentsMDPaths`) verifies that paths referenced in nested `AGENTS.md` files exist
- `doc/dependencies.md`: approved external dependency list (runtime: none — standard library only; dev tools pinned via the `go.mod` `tool` directive)

### Changed
- Minimum Go version bumped from 1.25.12 to 1.26.6 to align with cardcore-server's minimum; no language or standard-library changes affect this module
- `Player` interface-method doc comments brought to the "behavior + type-specific why" standard: `Random.ChoosePass`/`ChoosePlay` and `Heuristic.ChoosePass`/`ChoosePlay` now describe each type's selection mechanism, tie-breaking, and panic preconditions, and the rollout-test `firstLegalPolicy.ChoosePlay` states its determinism rationale — replacing thin restatements of the `Player` interface contract. `PIMC`'s methods and `firstLegalPolicy.ChoosePass` already met the standard and are unchanged
- Named result parameters adopted where positional results were ambiguous: the Hearts AI `currentWinner` helper now returns `(winnerSeat hearts.Seat, winnerRank cardcore.Rank)` — the names its callers already used — and the `firstNonTwoOfClubs` rollout-test helper returns `(card cardcore.Card, ok bool)`. All other multi-result signatures were reviewed and deliberately left positional: `Deck.Deal`, `Game.LegalMoves`, and `Game.Winner` return `(T, error)` where the function name already documents `T`, and the six `build*` bench fixtures return the distinct, self-describing pair `(*hearts.Game, hearts.Seat)`. Signature-only refactor with no behavior change
- Minimum Go version bumped from 1.25.9 to 1.25.12 to stay current on standard-library security patches and align with cardcore-server's minimum; no vulnerabilities were reachable from this module at 1.25.9

## [0.7.0] - 2026-07-26

### Added
- Typed sentinel errors `ErrWrongPhase`, `ErrOutOfTurn`, and `ErrIllegalMove` in the Hearts package (`games/hearts/`). Rule-violation errors from `LegalMoves`, `Deal`, `SetPass`, `PlayCard`, `EndRound`, `Winner`, `ResolveTrick`, and the lead/follow validators now wrap the appropriate sentinel so callers can classify rejections with `errors.Is`. No function signatures changed.

## [0.6.0] - 2026-07-11

### Changed
- Hearts trick play and resolution are now separate steps: `PlayCard` no longer auto-resolves completed tricks. Instead, it leaves the full trick in `Game.Trick` and sets `Game.TrickPendingResolution`. Callers must invoke `Game.ResolveTrick()` to score the trick, record it in `TrickHistory`, and advance to the next trick or round scoring. This enables clients to observe the completed trick before it is cleared. (`games/hearts/`)

## [0.5.0] - 2026-06-10

### Fixed
- Seeded games are now fully deterministic: the same seed produces identical game outcomes. Previously `Deck.Shuffle` used the global `math/rand/v2` source, which is seeded from process entropy and cannot be reproduced. Now the caller provides a `*rand.Rand` that drives both the deck shuffle and the AI players. Breaking change: `Deck.Shuffle` now requires `*rand.Rand`; `hearts.New` now requires `*rand.Rand`.

## [0.4.0] - 2026-05-03

### Added
- PIMC AI player with parallel Monte Carlo sampling (`games/hearts/ai/`)

### Changed
- Minimum Go version bumped from 1.24.1 to 1.25.9 (requires `sync.WaitGroup.Go`)

## [0.3.1] - 2026-04-21

### Fixed
- Release workflow no longer fails when changelog entries contain shell metacharacters such as apostrophes

## [0.3.0] - 2026-04-21

### Added
- AI prerequisites: Clone, LegalMoves, and Player interface (`games/hearts/`)
- Random AI player (`games/hearts/ai/`)
- Heuristic AI player (`games/hearts/ai/`)
- Trick history and pass history to Hearts game state (`games/hearts/`)

### Fixed
- Prevent uint8 underflow panic in heuristic AI's moon-shoot detection when all hearts have been played (`games/hearts/ai/`)

## [0.2.0] - 2026-04-14

### Added
- Hearts card game engine (`games/hearts/`)

## [0.1.0] - 2026-04-08

### Added
- Core engine primitives: Suit, Rank, Card, Deck, and Hand
