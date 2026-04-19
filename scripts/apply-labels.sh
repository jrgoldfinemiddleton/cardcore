#!/usr/bin/env bash
# apply-labels.sh — compute and apply the canonical label set for a PR.
#
# Recomputes labels from PR title and changed file paths, then sets the
# PR's label state to match (adds missing, removes stale). Idempotent.
#
# Run locally:    make apply-labels PR=<n>
# Run in CI:      .github/workflows/labels-apply.yml (on pull_request_target)
#
# Requires: gh CLI authenticated with `repo` scope. Compatible with bash 3.2
# (macOS default) — uses no associative arrays or `mapfile`.

set -euo pipefail

PR="${1:-${PR:-}}"
if [[ -z "$PR" ]]; then
	echo "usage: $0 <pr-number>" >&2
	echo "       PR=<pr-number> $0" >&2
	exit 1
fi

# Managed labels — the set this script controls. Labels outside this set
# (e.g., Dependabot's `dependencies`, `github_actions`, or human-applied
# triage labels) are left untouched.
managed=(
	"scope:engine"
	"scope:hearts"
	"scope:ai"
	"scope:docs"
	"scope:ci"
	"scope:meta"
	"bug"
	"breaking-change"
)

# `want` accumulates desired labels as a space-separated set. The leading
# and trailing spaces let us test membership with `[[ "$want" == *" $label "* ]]`
# without false positives from prefix collisions (e.g., scope:ai vs scope:airline).
want=" "

add_want() {
	if [[ "$want" != *" $1 "* ]]; then
		want="$want$1 "
	fi
}

# Fetch PR metadata in one call.
meta=$(gh pr view "$PR" --json title,body,files,labels)
title=$(jq -r .title <<<"$meta")
body=$(jq -r .body <<<"$meta")

# Extract paths and current labels into checked variables. Running jq inside
# process substitution would hide failures from `set -euo pipefail`.
paths=$(jq -r '.files[].path' <<<"$meta")
current_labels=$(jq -r '.labels[].name' <<<"$meta")

# --- Path → scope mapping --------------------------------------------------
while IFS= read -r f; do
	case "$f" in
		.github/workflows/*|Makefile|.golangci.yml|convention_test.go|go.mod|go.sum)
			add_want "scope:ci" ;;
		.github/PULL_REQUEST_TEMPLATE.md|.github/ISSUE_TEMPLATE/*|scripts/*|.gitignore|LICENSE|SECURITY.md)
			add_want "scope:meta" ;;
		doc/*)
			add_want "scope:docs" ;;
		games/hearts/ai/*)
			add_want "scope:ai"
			add_want "scope:hearts" ;;
		games/*/ai/*)
			add_want "scope:ai" ;;
		games/hearts/*)
			add_want "scope:hearts" ;;
		*.md)
			# Top-level *.md (README, CHANGELOG, CONTRIBUTING, AGENTS, etc.).
			add_want "scope:docs" ;;
		*.go)
			# Top-level *.go is engine code (card.go, hand.go, etc.).
			# Subpackage *.go is matched by earlier cases.
			if [[ "$f" != */* ]]; then
				add_want "scope:engine"
			fi ;;
	esac
done <<<"$paths"

# --- Title → state label mapping -------------------------------------------
# `fix:`, `fix(scope):`, `fix!:`, `fix(scope)!:` → bug
if [[ "$title" =~ ^fix(\([^\)]+\))?!?: ]]; then
	add_want "bug"
fi

# `<type>!:` or `<type>(scope)!:` → breaking-change (Conventional Commits `!`)
if [[ "$title" =~ ^[a-z]+(\([^\)]+\))?!: ]]; then
	add_want "breaking-change"
fi

# `BREAKING CHANGE:` or `BREAKING-CHANGE:` footer in body → breaking-change
if grep -qE '^BREAKING[ -]CHANGE:' <<<"$body"; then
	add_want "breaking-change"
fi

# --- Diff against current labels and apply ---------------------------------
# Check current-label membership exactly (label-by-label), since unmanaged
# labels may contain spaces (e.g. "good first issue") and would create false
# positives in a space-padded substring check.
label_in_current() {
	local l
	while IFS= read -r l; do
		[[ "$l" == "$1" ]] && return 0
	done <<<"$current_labels"
	return 1
}

add=()
remove=()

for label in "${managed[@]}"; do
	in_want=0
	in_current=0
	[[ "$want" == *" $label "* ]] && in_want=1
	label_in_current "$label" && in_current=1
	if [[ $in_want -eq 1 && $in_current -eq 0 ]]; then
		add+=("$label")
	elif [[ $in_want -eq 0 && $in_current -eq 1 ]]; then
		remove+=("$label")
	fi
done

if [[ ${#add[@]} -eq 0 && ${#remove[@]} -eq 0 ]]; then
	echo "PR #$PR: labels already in desired state"
	exit 0
fi

args=()
for l in ${add[@]+"${add[@]}"}; do args+=(--add-label "$l"); done
for l in ${remove[@]+"${remove[@]}"}; do args+=(--remove-label "$l"); done

echo "PR #$PR: add=[${add[*]+${add[*]}}] remove=[${remove[*]+${remove[*]}}]"
gh pr edit "$PR" "${args[@]}"
