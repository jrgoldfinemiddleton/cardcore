# AI Agent Guidance (AGENTS.md)

## 1. Project Summary
Cardcore is a universal card game engine in Go. It is a library (no `main` package). Hearts is the first game. The design philosophy is suckless: minimal, composable, zero runtime dependencies, abstractions are deferred until they become necessary.

Module: `github.com/jrgoldfinemiddleton/cardcore`

## 2. Codebase Map
```
cardcore/
├── card.go              # Suit, Rank, Card, Deck — engine atoms
├── hand.go              # Hand — player's cards
├── games/
│   └── hearts/          # Hearts card game
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
│       ├── pr.yml       # PR validation: title check, make check, changelog nudge
│       ├── main.yml     # Push to main: make check
│       └── release.yml  # Tag push: validate, test, create GitHub Release
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
- Keep the Go version in `go.mod` aligned with the minimum version stated in `README.md`

## 4. Never Do
- Never add external dependencies — stdlib only
- Never use third-party GitHub Actions — first-party (`actions/*`) are acceptable
- Never put game logic in the root `cardcore` package
- Never extract generic abstractions (Player, GameState, Rules, etc.) until at least two games are implemented
- Never commit with failing tests or lint errors
- Never edit an ADR file after its initial commit — write a new one instead
- Never use `//nolint` directives to silence lint errors — fix the code instead
- Never tag a v1.0.0 or higher release — the root package is not yet stable enough for a v1.0.0 commitment

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
- **Formatting**: `gofmt` is enforced by `make check`; never manually format — let the tool do it
- **Comments**: exported symbols need doc comments; unexported ones are optional but welcome

## 7. Architecture Decisions
Read `doc/decisions/` for the rationale behind key choices. Important ADRs:
- ADR-003: Why Go
- ADR-004: Why API-first
- ADR-005: Why no generic abstractions yet
- ADR-006: Rules-Driven Development for games

## 8. When to Check In With the Human
- Before making any architectural change not covered by an ADR
- Before adding a new game package
- Before extracting any abstraction from game code into the root package
- Before adding any external dependency to the project (the answer will almost always be "no")
- Before writing or modifying any file, propose the change and wait for explicit approval
- Before installing any dev tool (the answer will almost always be "no")

## 9. Maintainer Runbook
If `doc/maintainer-runbook.md` exists locally, read it for release procedures, PR review workflow, repository settings reference, and recovery steps. Proactively remind the maintainer of relevant runbook procedures during release, review, and recovery scenarios.
