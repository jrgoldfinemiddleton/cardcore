# AI Agent Guidance (AGENTS.md)

## OVERVIEW

Cardcore is a universal card game engine in Go. It is a library (no `main` package). Hearts is the first game. The design philosophy is suckless: minimal, composable, zero runtime dependencies, abstractions are deferred until they become necessary.

Module: `github.com/jrgoldfinemiddleton/cardcore`

## STRUCTURE

```
cardcore/
├── card.go              # Suit, Rank, Card, Deck — engine atoms
├── hand.go              # Hand — player's cards
├── convention_test.go   # AST-based convention enforcement (not optional)
├── games/
│   └── hearts/          # Hearts card game
│       ├── doc.go       # Package documentation
│       ├── hearts.go    # Game state machine
│       ├── player.go    # Player interface
│       └── ai/          # Computer-controlled players (Random, Heuristic, PIMC)
├── doc/
│   ├── design.md        # Design principles
│   ├── architecture.md  # System architecture
│   ├── decisions/       # ADRs — read before architectural changes
│   └── games/
│       └── hearts/
│           ├── rules.md # Hearts rules specification (RDD)
│           └── ai-pimc-design.md # PIMC algorithm design
├── scripts/
│   ├── sync-labels.sh   # Source of truth for the repository label set
│   └── apply-labels.sh   # Auto-apply labels from changed paths
├── Makefile             # Build/test/lint targets
├── .golangci.yml        # Linter config
└── .golangci-extra.yml  # Optional stricter lint config
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add a new game | `games/<newgame>/`, `doc/games/<newgame>/rules.md` | Write rules doc first per ADR-006 |
| Add/modify AI | `games/hearts/ai/`, `doc/games/hearts/ai-pimc-design.md` | Read-only, stdlib-only, one type per technique |
| Change core atoms | `card.go`, `hand.go` | Keep root package free of game logic |
| Change Hearts rules | `games/hearts/hearts.go`, `doc/games/hearts/rules.md` | Update rules doc before implementation |
| Architectural change | `doc/decisions/` | Read ADR-001; write new ADR if needed |
| Build/CI | `Makefile`, `.github/workflows/` | `make check` is the single gate |

## CODE MAP

| Symbol | Type | Location | Refs | Role |
|--------|------|----------|------|------|
| `Card` | struct | `card.go` | 403 | Universal atom — every game package depends on it |
| `NewHand` | func | `hand.go` | 192 | Hand constructor used across engine + AI tests |
| `Hand` | struct | `hand.go` | 180+ | Player card container; core to game state and fixtures |
| `Game` | struct | `games/hearts/hearts.go` | 57 | Hearts state machine (deal, pass, trick, score) |
| `Player` | interface | `games/hearts/player.go` | implicit | AI contract: `ChoosePass`, `ChoosePlay` |
| `Random` | struct | `games/hearts/ai/random.go` | 8 | Baseline legal-move player |
| `Heuristic` | struct | `games/hearts/ai/heuristic.go` | 14 | Rule-based strategy player |
| `PIMC` | struct | `games/hearts/ai/pimc.go` | 57 | Perfect Information Monte Carlo player |
| `convention_test.go` | test | root | 6 tests | AST-based enforcement of ordering, docs, aliases, nolint, imports |

## CONVENTIONS

- Run `make check` before considering any change complete.
- Add or update tests whenever you add or change code.
- Write Go doc comments on all exported symbols (and unexported functions too — enforced by `convention_test.go`).
- Keep the root package free of game-specific logic.
- Place game-specific logic in subpackages under `games/`.
- Read the relevant ADRs in `doc/decisions/` before making architectural decisions.
- Follow Rules-Driven Development (ADR-006) when adding a game — write `doc/games/<game>/rules.md` first.
- Place AI in `games/<game>/ai/` and follow ADR-009: read-only, stdlib-only, one type per technique.
- Use prefixed rank/suit aliases in game tests (`rAce`..`rKing`, `sClubs`..`sSpades`) — enforced by `convention_test.go`.
- Keep the Go version in `go.mod` aligned with `README.md`.
- Use `want`/`got` for expected/actual values and `"got X, want Y"` error messages.
- Use `// Trick N:` (1-indexed, spelled out) comments in trick-taking test fixtures.
- For stochastic tests, use the `tries=N` loop pattern — never trust a single random comparison.
- Write commit messages following [Conventional Commits](https://www.conventionalcommits.org/): `<type>(<scope>): <description>`.

## ANTI-PATTERNS

- Never add external dependencies — stdlib only.
- Never use third-party GitHub Actions — `actions/*` only.
- Never put game logic in the root `cardcore` package.
- Never extract generic abstractions (Player, GameState, Rules, etc.) until at least two games are implemented.
- Never commit with failing tests or lint errors.
- Never edit the substantive content of an ADR file after its initial commit — write a new one instead.
- Never use `//nolint` — fix the code.
- Never write a decrementing `for` loop over an unsigned named type like `cardcore.Rank` or `cardcore.Suit` with a `>=` condition.
- Never tag a v1.0.0 or higher release.
- Never manually apply `scope:*` labels — they are computed by `scripts/apply-labels.sh`.
- Never write multi-line commit messages — one-line subject only.
- Never cite `AGENTS.md` as the source of a rule from any other file in the repo.

## UNIQUE STYLES

- **AST-based convention enforcement**: `convention_test.go` is not optional — it enforces function ordering, doc comments, no `//nolint`, no external deps, no game imports in root, and rank/suit aliases in game tests.
- **Dev tools via `go.mod`**: `golangci-lint`, `pkgsite`, and `benchstat` are declared via the `tool` directive; they compile on first use.
- **Technique-named AI players**: `Random`, `Heuristic`, and `PIMC` are named by algorithm, not by difficulty tier.
- **Rules-driven development**: every game starts with a `doc/games/<game>/rules.md` specification before code is written.
- **Seeded RNG everywhere**: every stochastic type accepts `*rand.Rand` for deterministic tests.

## COMMANDS

```bash
# Single gate before committing
make check

# Individual targets
make test      # Run all tests
make fmt       # gofmt
make vet       # go vet
make lint      # golangci-lint (uses go.mod tool directive)
make build     # Compile all packages
make lint-extra # Optional stricter lint

# Local docs / benchmarks
make doc       # pkgsite local docs
make bench     # Benchmarks with benchstat
make stats     # AI statistical profile (TestShootTheMoonFrequency)

# Labels (requires gh CLI authenticated)
make create-labels
make apply-labels PR=<number>
```

## NOTES

- `make check` is the only required gate; CI (`pr.yml`, `main.yml`) runs it.
- PR titles are validated against Conventional Commits by `pr.yml`; labels are auto-computed by `scripts/apply-labels.sh`.
- Release tags must be on `main`, valid semver, and have a matching `CHANGELOG.md` entry.

### Maintainer Runbook
If `doc/maintainer-runbook.md` exists locally, read it for release procedures, PR review workflow, repository settings reference, and recovery steps. Proactively remind the maintainer of relevant runbook procedures during release, review, and recovery scenarios.

### Implementation Plans
If `doc/implementation-plans.md` exists locally, read it for details pertaining to ongoing implementations, including plans, guidelines, and architecture details.

### Future Considerations
If `doc/future-considerations.md` exists locally, read it for a list of proposed features and improvements, including triggers for when they may be relevant for further consideration or implementation.
