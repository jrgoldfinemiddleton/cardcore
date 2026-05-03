# AI Agent Guidance (AGENTS.md)

## 1. Project Summary
Cardcore is a universal card game engine in Go. It is a library (no `main` package). Hearts is the first game. The design philosophy is suckless: minimal, composable, zero runtime dependencies, abstractions are deferred until they become necessary.

Module: `github.com/jrgoldfinemiddleton/cardcore`

## 2. Codebase Map
```
cardcore/
├── card.go              # Suit, Rank, Card, Deck — engine atoms
├── doc.go               # Package documentation
├── hand.go              # Hand — player's cards
├── games/
│   └── hearts/          # Hearts card game
│       ├── ai/          # Computer-controlled players
│       │   ├── doc.go       # Package documentation
│       │   ├── analysis.go  # Per-decision game-state analysis (card counts, voids, Q♠ location, moon threat)
│       │   ├── heuristic.go # Rule-based heuristic player
│       │   └── random.go    # Random legal move player
│       ├── doc.go       # Package documentation
│       ├── hearts.go    # Game logic
│       └── player.go    # Player interface
├── doc/
│   ├── design.md        # Design principles
│   ├── architecture.md  # System architecture
│   ├── decisions/       # ADRs — read these before making architectural changes
│   └── games/
│       └── hearts/
│           └── rules.md # Hearts rules specification (RDD)
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.yml
│   │   └── config.yml   # Redirects features/questions to Discussions
│   └── workflows/
│       ├── pr.yml             # PR validation: title check, make check, changelog nudge
│       ├── main.yml           # Push to main: make check
│       ├── release.yml        # Tag push: validate, test, create GitHub Release
│       ├── labels-sync.yml    # Push to main: provision repository label set
│       └── labels-apply.yml   # PR events: auto-apply scope/state labels
├── scripts/
│   ├── sync-labels.sh   # Source of truth for the repository label set
│   └── apply-labels.sh  # Compute and apply labels for a PR
├── CONTRIBUTING.md      # Contribution guidelines
├── SECURITY.md          # Vulnerability reporting
├── Makefile             # Build/test/lint targets
├── .golangci.yml        # Linter config
└── README.md            # Project overview
```

## 3. Always Do
- Run `make check` before considering any change complete (dev tools are managed via the `tool` directive in `go.mod` — no manual installation required)
- Add or update tests whenever you add or change code — never leave tests behind
- Write Go doc comments on all exported symbols
- Keep the root package (`cardcore`) free of game-specific logic
- Place game-specific logic in subpackages under `games/` (e.g., `games/hearts/`)
- Follow existing naming conventions: exported types are PascalCase, unexported are camelCase
- Read the relevant ADRs in `doc/decisions/` before making architectural decisions
- Follow Rules-Driven Development ([ADR-006](doc/decisions/006-rules-driven-development.md)) when adding a game — write the rules document before implementing
- Place AI in `games/<game>/ai/` subpackages
- Follow [ADR-009](doc/decisions/009-ai-difficulty-and-personality.md) when implementing AI — read-only access to live game state, stdlib-only, separate type per difficulty
- When adding benchmarks, use stdlib `testing.B` only and share deterministic fixtures via `*_helpers_test.go` builders. Place `Benchmark*` functions after `Test*` and before `Fuzz*`/`Example*` in the file (enforced by `convention_test.go`).
- Within any file, all type/var/const declarations must precede all function declarations (enforced by `convention_test.go`).
- Within any test file, helpers must come last — after all unit tests, integration tests, benchmarks, fuzz tests, and examples (enforced by `convention_test.go`).
- Stochastic test assertions must use the `tries=N` loop pattern. When verifying "different inputs produce different outputs" of a randomized function, try N distinct inputs and require at least one disagreement; a single comparison can collapse by RNG luck even when the function is correct. Examples in `games/hearts/ai/pimc_test.go` and `pimc_aggregate_test.go`.
- **Trick-taking games only**: in test fixtures that build trick history, **comments** that label tricks should use the form `// Trick N:` (spelled out, 1-indexed), where `Trick 1` is the first trick of the round. When a fixture uses `validFirstTrick()` (or equivalent opener helper), annotate the call with `// Trick 1: validated 2♣ opener.` (or the game's equivalent opener description). This applies to comments only — engine code may use whatever indexing it wants (e.g., `g.TrickNum` is 0-indexed).
- Keep the Go version in `go.mod` aligned with the minimum version stated in `README.md`
- Read `CONTRIBUTING.md` for general project conventions (naming, changelog rules, code style, doc comments) before making changes — many rules live there rather than being duplicated here.

## 4. Never Do
- Never add external dependencies — stdlib only
- Never use third-party GitHub Actions — first-party (`actions/*`) are acceptable
- Never put game logic in the root `cardcore` package
- Never extract generic abstractions (Player, GameState, Rules, etc.) until at least two games are implemented
- Never commit with failing tests or lint errors
- Never edit the substantive content of an ADR file after its initial commit — write a new one instead. The Status line is the exception: update it when an ADR is superseded or deprecated (otherwise the `Superseded` / `Deprecated` status values would never apply).
- When superseding an ADR, the new ADR must be self-contained: carry forward every part of the old ADR that remains valid, and amend or replace only the parts that change. A reader should never have to consult the superseded ADR to understand current policy. (Deprecation is different — a deprecated ADR has no successor carrying its content forward.)
- Never use `//nolint` directives to silence lint errors — fix the code instead
- Never write a decrementing `for` loop over a `cardcore.Rank`, `cardcore.Suit`, or any other unsigned named type using a condition like `r >= Two` — at zero the value wraps to a huge number and the loop runs forever or indexes out of bounds. If you need to iterate descending, use the headerless form with an explicit guarded break before the decrement that would underflow (see `games/hearts/ai/analysis.go` for an example, introduced in [PR #15](https://github.com/jrgoldfinemiddleton/cardcore/pull/15)).
- Never tag a v1.0.0 or higher release — the root package is not yet stable enough for a v1.0.0 commitment
- Never manually apply `scope:*` labels to PRs — they are computed automatically from changed paths by `scripts/apply-labels.sh`. Edit the script's path rules if a label is wrong.
- Never write multi-line commit messages — use a one-line subject only and put all detail in the PR description.
- Never cite `AGENTS.md` as the source of a rule from any other file in the repo — `AGENTS.md` may reference other files, but not vice versa. Mere mention or listing of `AGENTS.md` as a file (e.g., scripts that operate on it) is fine; this rule is about citation/deference, not mention.

## 5. Development Workflow
1. Make a change
2. Run `make check` — must pass clean
3. If lint errors appear, fix the code (do not suppress with `//nolint`)
4. Commit only when all checks pass
5. Write commit messages following [Conventional Commits](https://www.conventionalcommits.org/)
   - Format: `<type>(<scope>): <description>`
   - Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`
   - Example: `feat(card): add Deck shuffle method`

## 6. Key Conventions
- **Error handling**: functions return `error` as the last return value; callers must check it
- **Precondition violations**: programming errors by the caller trigger panics; functions return `error` only for conditions the caller cannot prevent — see [Error Handling](doc/design.md#error-handling)
- **No global state**: all state is in structs passed explicitly
- **Testing**: use standard `testing` package; test files are `*_test.go` in the same package
  - Every test must execute its assertion — if there's a conditional path to the assertion, the test can silently pass without testing anything; prefer deterministic setups that guarantee the condition under test is reached
  - Test data must be realistic — if the domain has invariants (e.g., round points sum to 26), test data must respect them; impossible states can mask bugs
  - When testing for errors, verify *which* error — checking `err != nil` is brittle when the function under test has multiple validation checks; verify the error message
  - Accumulation needs a nonzero starting point — any test for `+=` behavior must start with existing state, otherwise `=` and `+=` are indistinguishable
  - Never rely on random setup to exercise a specific code path — if you need a particular hand configuration, construct it explicitly
  - Each game engine must include an integration test that exercises the full state machine lifecycle (start to terminal state) and verifies structural invariants hold across rounds (e.g., point conservation, hand depletion, phase transitions, no state leaks between rounds)
  - Scenarios that require multiple subsystems cooperating (e.g., trick resolution → point accumulation → score detection → special scoring logic) need their own integration tests — unit tests on each piece in isolation are not sufficient
  - Implemented variants require integration tests — exceptions need significant justification
  - Test files in game packages (and their subpackages) must define prefixed const aliases for `cardcore` ranks and suits — `rAce`, `rTwo`, …, `rKing` for ranks and `sClubs`, `sDiamonds`, `sHearts`, `sSpades` for suits. Use these aliases in all test code instead of qualified `cardcore.Rank`/`cardcore.Suit` constants. Place the alias definitions in a shared test helpers file (e.g., `helpers_test.go`). The root `cardcore` package is exempt since it defines the constants directly.
  - In tests, name expected-value variables `want` (and corresponding actual-value variables `got`) — never `expected`/`actual`. This matches Go standard library convention and pairs naturally with `"got X, want Y"` error messages.
  - Test failure messages use `"got X, want Y"` form — never `"expected X, got Y"` (wrong order) or `"expected X"` (passive). No colon after `got` (`"got %v, want %v"`, not `"got: %v, want %v"`). Soft convention; not enforced by `convention_test.go`.
- **Formatting**: `gofmt` is enforced by `make check`; never manually format — let the tool do it
- **Function ordering**: follow the conventions in [CONTRIBUTING.md](CONTRIBUTING.md#code-conventions) — `convention_test.go` enforces them automatically via `make check`
- **Comments**: every function and method needs a doc comment starting with its name — `convention_test.go` enforces this automatically via `make check`

## 7. Architecture Decisions
Read `doc/decisions/` for the rationale behind key choices. Important ADRs:
- ADR-003: Why Go
- ADR-004: Why API-first
- ADR-005: Why no generic abstractions yet
- ADR-006: Rules-Driven Development for games
- ADR-007: Automated code convention enforcement
- ADR-009: AI difficulty and personality (supersedes ADR-008)

## 8. When to Check In With the Human
- Before making any architectural change not covered by an ADR
- Before adding a new game package
- Before extracting any abstraction from game code into the root package
- Before adding any external dependency to the project (the answer will almost always be "no")
- Before writing or modifying any file, propose the change and wait for explicit approval
- Before installing any dev tool (the answer will almost always be "no")

## 9. Maintainer Runbook
If `doc/maintainer-runbook.md` exists locally, read it for release procedures, PR review workflow, repository settings reference, and recovery steps. Proactively remind the maintainer of relevant runbook procedures during release, review, and recovery scenarios.
