#!/usr/bin/env bash
# release.sh — verify readiness, then create and push an annotated release tag.
#
# This script is the only supported way to tag a release. Tags are permanent:
# the Go module proxy caches them on first fetch and the repository's tag
# ruleset forbids deletion and mutation, so every check runs before anything
# is created.
#
# Usage:   scripts/release.sh vX.Y.Z [--dry-run] [--yes]
# Example: scripts/release.sh v0.8.0 --dry-run   # rehearse, create nothing
#
# The pushed tag triggers .github/workflows/release.yml, which re-validates
# (tag on main, semver, changelog section, make check) before creating the
# GitHub Release. Cardcore is a library: the tag is the release artifact.

set -euo pipefail

cd "$(dirname "$0")/.."

DRY_RUN=false
ASSUME_YES=false
VERSION=""
NOTES_FILE=""

usage() {
	cat <<'EOF'
Usage: scripts/release.sh vX.Y.Z [--dry-run] [--yes]

Creates and pushes the annotated release tag vX.Y.Z after verifying the
repository is releasable:

  1. arguments and environment (strict semver, pre-1.0 policy, required tools)
  2. repository state (on main, clean tree, up to date, tag new and greater)
  3. changelog readiness (dated [X.Y.Z] section, empty [Unreleased])
  4. quality gates (make check)

Options:
  --dry-run   run every check and print the summary, but create nothing
  --yes       skip the interactive confirmation prompts
  -h, --help  show this help
EOF
}

fail() {
	echo "error: $*" >&2
	exit 1
}

info() {
	echo "==> $*"
}

warn() {
	echo "warning: $*" >&2
}

confirm() {
	local prompt="$1"
	if [[ "$ASSUME_YES" == true ]]; then
		return 0
	fi
	local reply
	read -r -p "$prompt [y/N] " reply || fail "no input; aborting"
	[[ "$reply" == "y" || "$reply" == "yes" ]] || fail "aborted by user"
}

parse_args() {
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--dry-run) DRY_RUN=true ;;
		--yes) ASSUME_YES=true ;;
		-h | --help)
			usage
			exit 0
			;;
		-*)
			usage >&2
			fail "unknown option: $1"
			;;
		*)
			[[ -z "$VERSION" ]] || {
				usage >&2
				fail "exactly one version argument is required"
			}
			VERSION="$1"
			;;
		esac
		shift
	done

	[[ -n "$VERSION" ]] || {
		usage >&2
		fail "a version argument is required"
	}
}

check_version_arg() {
	info "Phase 1: validating version argument and environment"

	[[ "$VERSION" =~ ^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] ||
		fail "not valid semver (expected vX.Y.Z with no leading zeros): $VERSION"
	[[ "$VERSION" == v* ]] || VERSION="v$VERSION"

	local major="${VERSION#v}"
	major="${major%%.*}"
	[[ "$major" == "0" ]] ||
		fail "releases v1.0.0 and higher are not permitted by project policy"

	local tool
	for tool in git go make curl sed; do
		command -v "$tool" >/dev/null || fail "required tool not found: $tool"
	done
}

check_repo_state() {
	info "Phase 2: verifying repository state"

	[[ "$(git rev-parse --abbrev-ref HEAD)" == "main" ]] ||
		fail "not on the main branch"

	[[ -z "$(git status --porcelain)" ]] ||
		fail "working tree is not clean; commit or stash everything first"

	git fetch --quiet origin main
	[[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/main)" ]] ||
		fail "HEAD is not up to date with origin/main"

	if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
		fail "tag $VERSION already exists locally"
	fi
	if [[ -n "$(git ls-remote --tags origin "$VERSION")" ]]; then
		fail "tag $VERSION already exists on origin"
	fi

	LATEST_TAG="$(git tag --list 'v[0-9]*' --sort=-version:refname | head -n 1)"
	if [[ -z "$LATEST_TAG" ]]; then
		info "no existing tags; this will be the first release"
		return 0
	fi

	local highest
	highest="$(printf '%s\n%s\n' "$LATEST_TAG" "$VERSION" | sort -V | tail -n 1)"
	[[ "$highest" == "$VERSION" ]] ||
		fail "$VERSION is not greater than the latest tag $LATEST_TAG"

	local prev_minor prev_patch new_minor new_patch
	prev_minor="$(cut -d. -f2 <<<"$LATEST_TAG")"
	prev_patch="$(cut -d. -f3 <<<"$LATEST_TAG")"
	new_minor="$(cut -d. -f2 <<<"$VERSION")"
	new_patch="$(cut -d. -f3 <<<"$VERSION")"
	if [[ "$new_minor" -ne "$prev_minor" && "$new_minor" -ne $((prev_minor + 1)) ]] ||
		[[ "$new_minor" -eq "$prev_minor" && "$new_patch" -ne $((prev_patch + 1)) ]] ||
		[[ "$new_minor" -ne "$prev_minor" && "$new_patch" -ne 0 ]]; then
		warn "version skips ahead of $LATEST_TAG"
		confirm "Release $VERSION anyway?"
	fi
}

check_changelog() {
	info "Phase 3: verifying CHANGELOG.md readiness"

	local version_num="${VERSION#v}"

	# This extraction must match .github/workflows/release.yml exactly: if it
	# passes here, the workflow cannot fail on a missing changelog section.
	sed -n "/^## \[${version_num}\]/,/^## \[/{/^## \[${version_num}\]/d;/^## \[/d;p;}" \
		CHANGELOG.md >"$NOTES_FILE"
	[[ -n "$(tr -d '[:space:]' <"$NOTES_FILE")" ]] ||
		fail "no changelog entry found for ${version_num}; add a '## [${version_num}]' section first"

	local unreleased
	unreleased="$(sed -n "/^## \[Unreleased\]/,/^## \[/{/^## \[Unreleased\]/d;/^## \[/d;p;}" CHANGELOG.md | tr -d '[:space:]')"
	[[ -z "$unreleased" ]] ||
		fail "[Unreleased] is not empty; move its items into the [${version_num}] section first"

	local header today
	header="$(grep -m1 "^## \[${version_num}\] - " CHANGELOG.md || true)"
	[[ -n "$header" ]] ||
		fail "the [${version_num}] heading is not dated (expected '## [${version_num}] - YYYY-MM-DD')"
	today="$(date +%F)"
	if [[ "$header" != *"$today"* ]]; then
		warn "changelog heading date is not today ($today): $header"
		confirm "Release anyway?"
	fi

	local subject
	subject="$(git log -1 --format=%s)"
	if [[ ! "$subject" =~ ^docs\(changelog\):\ prepare\ ${VERSION}\ release ]]; then
		warn "HEAD commit subject is not the changelog-prepare commit: $subject"
		confirm "Tag this commit anyway?"
	fi
}

run_quality_gates() {
	info "Phase 4: running quality gates (make check)"
	make check
}

summarize_and_confirm() {
	info "Release summary"
	cat <<EOF

  version:    $VERSION
  previous:   ${LATEST_TAG:-none (first release)}
  commit:     $(git rev-parse HEAD)
  changelog:  $(wc -l <"$NOTES_FILE" | tr -d ' ') lines of release notes
  tag:        annotated, message "Release $VERSION"

EOF
	if [[ "$DRY_RUN" == true ]]; then
		info "dry run: all checks passed; no tag was created"
		exit 0
	fi
	confirm "Create and push the tag $VERSION? This cannot be undone."
}

tag_and_push() {
	git tag -a "$VERSION" -m "Release $VERSION"
	git push origin "$VERSION"
	info "tag $VERSION pushed"
}

post_release() {
	info "Phase 5: post-release"
	if ! curl -sf "https://proxy.golang.org/github.com/jrgoldfinemiddleton/cardcore/@v/${VERSION}.info"; then
		warn "Go module proxy request failed; retry manually after the release lands"
	fi
	cat <<EOF

Next:
  1. Watch the workflow:
     https://github.com/jrgoldfinemiddleton/cardcore/actions/workflows/release.yml
  2. Verify the release:
     https://github.com/jrgoldfinemiddleton/cardcore/releases/tag/$VERSION
     Expect a GitHub Release titled $VERSION with the changelog section as
     its notes. There are no uploaded assets; the tag is the artifact.
  3. Trigger documentation indexing:
     https://pkg.go.dev/github.com/jrgoldfinemiddleton/cardcore@$VERSION
EOF
}

main() {
	parse_args "$@"
	NOTES_FILE="$(mktemp)"
	trap 'rm -f "$NOTES_FILE"' EXIT

	check_version_arg
	check_repo_state
	check_changelog
	run_quality_gates
	summarize_and_confirm
	tag_and_push
	post_release
}

main "$@"
