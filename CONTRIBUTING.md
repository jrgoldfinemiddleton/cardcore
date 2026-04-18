# Contributing to cardcore

## Prerequisites

[Go](https://go.dev/) 1.24.1+. Dev tools like [golangci-lint](https://golangci-lint.run/) are managed via the `tool` directive in `go.mod` and compiled automatically on first use.

## Development Workflow

1. Fork and clone the repository.
2. Create a topic branch from `main`.
3. Make your changes. Add or update tests as needed.
4. Run `make check` — must pass clean.
5. Commit using [Conventional Commits](#commit-messages) format.
6. Open a pull request against `main`.

All pull requests (PRs) are squash-merged, so feel free to commit frequently on your branch.

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). PR titles must follow one of these formats:

```
<type>: <description>
<type>(<scope>): <description>
```

**Allowed types:**

| Type | Purpose |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `chore` | Maintenance (CI, build, tooling) |

An optional `!` after the type/scope indicates a breaking change: `feat(card)!: change Card field order`.

**Note on versioning:** The project is pre-v1.0.0. Breaking changes may occur in any release. A formal versioning and stability policy will be established as the API matures through multiple game implementations.

## Adding a Game

This project follows **Rules-Driven Development** (see
[ADR-006](doc/decisions/006-rules-driven-development.md)). When adding a
new game:

1. **Document the rules.** Write `doc/games/<game>/rules.md` with the
   complete standard rules and any known variants. The rules document is
   the specification — it must be precise, unambiguous, and readable by
   non-technical players. For established games, cite primary references.
   For original games, the rules document itself is the authoritative
   definition. The rules document may be submitted as its own PR before
   any implementation work begins.
2. **Implement the game.** Create a package under `games/<game>/`. Build
   the implementation against the rules document.
3. **Add a package doc.** Create `games/<game>/doc.go` with a concise
   overview (see `games/hearts/doc.go` for the pattern).
4. **Write tests.** Test every rule and edge case described in the rules
   document. Include an integration test that plays a complete game from
   start to finish, verifying structural invariants (e.g., hand depletion,
   phase transitions) hold throughout. Implemented variants require
   integration tests.
5. **Run `make check`.** Must pass clean before opening a PR.

## Changing Existing Rules

Rules documents are living specifications. Changes fall into three
categories:

**Rules-only PR (no code changes required):**
- Clarifications that do not change specified behavior (rewording,
  calling out edge cases the code already handles)
- Adding a new variant category that is not yet implemented
- Adding, changing, or removing details within a variant that has no
  implementation

**Rules + implementation required in the same PR:**
- Changes that alter the specified behavior of an implemented game
- Adding a variant to an already-implemented variant category (where
  omitting it creates an obvious gap)
- Adding a variant that is a trivial extension of existing variant code
- Changing or removing an implemented variant's specification —
  removal should not be done lightly, but when it
  happens, look for opportunities to simplify the engine code that
  no longer needs to support the removed variant

**Not a PR — file a bug or open a Discussion instead:**
- You believe the rules are wrong but cannot contribute the code changes
- You want to propose a rule change for community feedback before
  committing to the work
- You want to propose a major overhaul of a game's standard rules —
  this has significant upstream impact on servers, clients, and tests,
  and should be discussed before any PR is created
- You want to change the primary reference for an established game —
  the references cited in the merged rules document are the accepted
  authority, and proposing a conflicting reference is a high bar to
  overcome, especially when an implementation already exists

## Guidelines

- **No external dependencies.** This project uses only the Go standard library.
- **Tests are required.** Every code change should include corresponding tests.
- **Run `make check`** before pushing. It runs formatting, vetting, linting, and tests.
- **Update the changelog.** Add a note under the `## [Unreleased]` section in `CHANGELOG.md` for user-facing changes.

## Code Conventions

This project enforces code conventions automatically via tests in the
root package (`convention_test.go`). All checks run as part of
`make check`. See
[ADR-007](doc/decisions/007-automated-convention-enforcement.md) for
the rationale behind automated enforcement.

Cardcore aims to be approachable by contributors who are new to Go.
Consistent structure and thorough doc comments help newcomers navigate
unfamiliar code without assuming prior Go experience. The conventions
below exist to lower the barrier to entry, not to create busywork —
`make check` gives you immediate, specific feedback when something is
out of place.

### Doc comments

Every function and method — exported and unexported — must have a doc
comment. The comment must begin with the function or method name:

```go
// processHand evaluates the cards and returns a score.
func processHand(h *Hand) int {
```

This applies to test helpers too. A one-line comment stating intent is
sufficient for small functions.

### Function ordering

#### Declarations before functions

All type, const, and var declarations must appear before any function
or method declarations. Never define a type, const, or var between
functions.

#### Production files

1. Constructor functions (`New*`)
2. Exported methods — grouped by receiver type
3. Exported package-level functions
4. Unexported methods — grouped by receiver type
5. Unexported package-level functions

Methods on the same receiver must be contiguous — never interleave
methods from different receivers.

#### Test files

1. Compile-time interface checks (`var _ T = (*Impl)(nil)`)
2. Unit tests (`func Test*`)
3. Integration tests (`func Test*Integration`, `func Test*FullGame*`)
4. Test helpers and setup functions (at the bottom)

### Before every commit

Scan the **entire** file you changed, not just newly added sections.
The convention tests cover all `.go` files in the module — a violation
anywhere will fail `make check`.

## Reporting Bugs

Use the [bug report template](https://github.com/jrgoldfinemiddleton/cardcore/issues/new?template=bug_report.yml) on GitHub.

## Suggesting Features

Open a [GitHub Discussion](https://github.com/jrgoldfinemiddleton/cardcore/discussions) to propose and discuss feature ideas before opening a PR.
