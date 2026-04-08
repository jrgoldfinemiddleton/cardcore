# Contributing to cardcore

## Prerequisites

[Go](https://go.dev/) 1.22+ and [golangci-lint](https://golangci-lint.run/):

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Development Workflow

1. Fork and clone the repository.
2. Create a topic branch from `main`.
3. Make your changes. Add or update tests as needed.
4. Run `make check` — must pass clean.
5. Commit using [Conventional Commits](#commit-messages) format.
6. Open a pull request against `main`.

All PRs are squash-merged, so feel free to commit frequently on your branch.

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

## Guidelines

- **No external dependencies.** This project uses only the Go standard library.
- **Tests are required.** Every code change should include corresponding tests.
- **Run `make check`** before pushing. It runs formatting, vetting, linting, and tests.
- **Update the changelog.** Add a note under the `## [Unreleased]` section in `CHANGELOG.md` for user-facing changes.

## Reporting Bugs

Use the [bug report template](https://github.com/jrgoldfinemiddleton/cardcore/issues/new?template=bug_report.yml) on GitHub.

## Suggesting Features

Open a [GitHub Discussion](https://github.com/jrgoldfinemiddleton/cardcore/discussions) to propose and discuss feature ideas before opening a PR.
