# ADR-007: Automated Code Convention Enforcement

**Date:** 2026-04-17
**Status:** Accepted

## Context
Go's toolchain enforces formatting (`gofmt`) and catches common
mistakes (`go vet`), but it does not prescribe function ordering,
require doc comments on unexported symbols, or enforce structural
conventions within files. Without automation, these conventions rely
entirely on code review — which is unreliable as the codebase grows
across packages.

Beyond correctness, Cardcore has a philosophical goal: the project
should be approachable by contributors who are new to Go. Clear
conventions, enforced automatically, lower the barrier to entry.
A newcomer should be able to open any file and understand its
structure immediately — not because they already know Go idioms, but
because the code explains itself through consistent ordering and
doc comments on every function. The linter and the test suite hand
contributors feedback on a golden platter rather than expecting them
to internalize conventions by reading existing code.

We need (1) clear conventions that contributors can follow, (2)
automated enforcement so violations are caught before merge, and (3)
conventions that serve readability for newcomers, not just experienced
Go developers.

## Decision
We enforce code conventions automatically via tests in the root
`cardcore` package (`convention_test.go`) that run as part of
`go test ./...` (and therefore `make check`).

### What is enforced

#### 1. Declarations before functions
All type, const, and var declarations must appear before any function
or method declarations in every `.go` file. Import declarations are
exempt (the Go spec requires them at the top).

#### 2. Function ordering
**Production files:** Functions are ordered by group — constructors,
exported methods, exported package-level functions, unexported methods,
unexported package-level functions. Methods on the same receiver type
must be contiguous.

**Test files:** Declarations are ordered — compile-time interface
checks, unit tests, integration tests, helpers.

#### 3. Doc comments
Every function and method — exported and unexported — must have a
doc comment. The comment must begin with the function or method name,
following Go convention (e.g., `// FuncName does X`).

How this is enforced is an implementation detail — it may be a
custom test in `convention_test.go`, a third-party linter such as
`godoclint` (available in golangci-lint v2), or both. The requirement
itself is what this ADR establishes.

Requiring doc comments on unexported symbols goes beyond standard Go
practice, which only mandates them on exported identifiers. We enforce
it because:
- A short comment stating intent helps contributors unfamiliar with
  the codebase (or with Go itself) understand what a function does
  without reading its body.
- It costs seconds to write and saves minutes to read.
- Automated enforcement removes the judgment call of "is this function
  obvious enough to skip?" — every function gets a comment, no
  exceptions.

### Checker placement
The checker lives in the root package (`convention_test.go`) rather
than in a separate `internal/lint` or `tools/` package. It walks the
module directory tree to discover all `.go` files across every nested
package.

**Rationale:**
- A root-package test can walk the entire module directory tree without
  import cycles or build tag gymnastics.
- It runs automatically with `go test ./...` — no Makefile target, CI
  step, or external tool to maintain.
- It uses only the standard library (e.g., `go/ast`, `go/parser`,
  `go/token`, `path/filepath`) — consistent with the project's
  zero-dependency principle.
- Placing it alongside the code it guards (rather than in a separate
  tool) means it cannot be accidentally skipped.

### What is not enforced
The relative ordering of type, const, and var declarations among
themselves is not automated. The codebase uses a natural "type-first
grouping" pattern (type definition → associated consts → associated
vars) that would be awkward to express as a strict linear rule. This
remains a style convention enforced during review.

## Consequences
(+) Convention violations are caught by `make check` before they reach
    review.
(+) Contributors have a single, readable reference (CONTRIBUTING.md)
    for code conventions, backed by this ADR as the authority.
(+) New contributors get immediate, specific feedback instead of
    learning conventions through trial and error.
(+) The checker requires no external tooling or installation.
(+) Adding a new package automatically includes it in the check.
(-) The root package gains a test file unrelated to its public API.
(-) Changes to conventions require updating both CONTRIBUTING.md and
    the checker.
(-) Requiring doc comments on unexported symbols is stricter than Go
    community norms, which may surprise contributors coming from other
    Go projects.
(-) The directory-walking approach means the test reads the filesystem,
    making it slightly slower than a pure AST test on a single package.
