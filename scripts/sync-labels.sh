#!/usr/bin/env bash
# sync-labels.sh — provision the canonical label set for this repository.
#
# This script is the single source of truth for which labels exist on the
# repository. It is idempotent: running it repeatedly converges the repo's
# label set to match this file.
#
# Run locally:    make create-labels
# Run in CI:      .github/workflows/labels-sync.yml (on push to main)
#
# Requires: gh CLI authenticated with `repo` scope.

set -euo pipefail

# scope:* labels mirror Conventional Commits scopes used in commit messages.
# Single blue family so they visually cluster in label dropdowns.
gh label create "scope:engine" --color "1d76db" --description "Root cardcore package (engine primitives)" --force
gh label create "scope:hearts" --color "1d76db" --description "Hearts game logic (games/hearts/, excluding ai/)" --force
gh label create "scope:ai" --color "1d76db" --description "Game AI implementations (games/<game>/ai/)" --force
gh label create "scope:docs" --color "1d76db" --description "Documentation (doc/, README, AGENTS, CHANGELOG, CONTRIBUTING, ADRs)" --force
gh label create "scope:ci" --color "1d76db" --description "Build, test, lint, workflows (Makefile, .github/, .golangci.yml, convention_test.go)" --force
gh label create "scope:meta" --color "1d76db" --description "Repository governance (labels, settings, scripts/)" --force

# State / type labels.
gh label create "bug" --color "d73a4a" --description "Something isn't working" --force
gh label create "breaking-change" --color "b60205" --description "Introduces a backwards-incompatible change to the public API" --force

# GitHub default labels that don't fit our workflow:
#   - enhancement, question: feature requests and questions go to Discussions
#     (see .github/ISSUE_TEMPLATE/config.yml).
#   - duplicate, invalid, wontfix: prefer close-with-comment over a label.
#   - good first issue, help wanted: revisit when actively seeking contributors.
#   - documentation: redundant with scope:docs.
#
# `delete_if_exists` only swallows the "label does not exist" case so the
# script remains idempotent without masking real failures (auth, network,
# rate-limiting). Any other error aborts the script.
delete_if_exists() {
	local label="$1"
	local err
	if err=$(gh label delete "$label" --yes 2>&1); then
		return 0
	fi
	if [[ "$err" == *"not found"* ]]; then
		return 0
	fi
	echo "$err" >&2
	return 1
}

delete_if_exists "enhancement"
delete_if_exists "question"
delete_if_exists "duplicate"
delete_if_exists "invalid"
delete_if_exists "wontfix"
delete_if_exists "good first issue"
delete_if_exists "help wanted"
delete_if_exists "documentation"

# `dependencies` and `github_actions` are auto-created and applied by
# Dependabot. We leave them alone.
