#!/usr/bin/env bash
# prepare-release.sh — create the changelog-preparation PR for a release.
#
# This script is the only supported way to prepare a release's changelog
# section and its PR. Hand-edited prep PRs drift (wrong heading format, wrong
# PR body), and the release tooling depends on both. The script validates the
# repository, moves [Unreleased] into a dated [X.Y.Z] section, and opens the
# PR with the fixed title and body.
#
# Usage:   scripts/prepare-release.sh vX.Y.Z [--dry-run] [--yes]
# Example: scripts/prepare-release.sh v0.7.1 --dry-run   # rehearse, create nothing
#
# After the PR it opens is merged, cut the release with scripts/release.sh.

set -euo pipefail

cd "$(dirname "$0")/.."

DRY_RUN=false
ASSUME_YES=false
VERSION=""
VERSION_NUM=""
BRANCH=""
TODAY=""

usage() {
	cat <<'EOF'
Usage: scripts/prepare-release.sh vX.Y.Z [--dry-run] [--yes]

Prepares the changelog for the vX.Y.Z release and opens the prep PR:

  1. arguments and environment (strict semver, pre-1.0 policy, required tools)
  2. repository state (on main, clean tree, up to date, tag and branch absent)
  3. changelog readiness ([Unreleased] non-empty, no existing [X.Y.Z] section)
  4. edit rehearsal (dated [X.Y.Z] heading, verified with the same extraction
     the release tooling uses)
  5. branch, commit, push, and open the docs(changelog) PR with the fixed body

Options:
  --dry-run   run every check and print the plan, but create nothing
  --yes       skip the interactive confirmation prompt
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
	VERSION_NUM="${VERSION#v}"

	local major="${VERSION_NUM%%.*}"
	[[ "$major" == "0" ]] ||
		fail "releases v1.0.0 and higher are not permitted by project policy"

	local tool
	for tool in git gh awk sed; do
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
		fail "tag $VERSION already exists locally; nothing to prepare"
	fi
	[[ -z "$(git ls-remote --tags origin "$VERSION")" ]] ||
		fail "tag $VERSION already exists on origin; nothing to prepare"

	BRANCH="docs/prepare-${VERSION}-release"
	if git rev-parse -q --verify "refs/heads/$BRANCH" >/dev/null; then
		fail "branch $BRANCH already exists locally"
	fi
	[[ -z "$(git ls-remote --heads origin "$BRANCH")" ]] ||
		fail "branch $BRANCH already exists on origin"
}

check_changelog() {
	info "Phase 3: verifying CHANGELOG.md readiness"

	if grep -q "^## \[${VERSION_NUM}\]" CHANGELOG.md; then
		fail "CHANGELOG.md already has a [${VERSION_NUM}] section"
	fi

	local unreleased
	unreleased="$(sed -n "/^## \[Unreleased\]/,/^## \[/{/^## \[Unreleased\]/d;/^## \[/d;p;}" CHANGELOG.md | tr -d '[:space:]')"
	[[ -n "$unreleased" ]] ||
		fail "[Unreleased] is empty; there is nothing to release"

	TODAY="$(date +%F)"
}

# insert_heading writes CHANGELOG.md to $1 with the dated [X.Y.Z] heading
# inserted directly below the [Unreleased] heading. awk is used instead of
# sed -i because in-place sed syntax differs between BSD and GNU.
insert_heading() {
	awk -v heading="## [${VERSION_NUM}] - ${TODAY}" '
		!done && /^## \[Unreleased\]$/ { print; print ""; print heading; done=1; next }
		{ print }
	' CHANGELOG.md >"$1"
}

# verify_edit fails unless the file at $1 has a non-empty [X.Y.Z] section and
# an empty [Unreleased], using the same extraction as scripts/release.sh and
# .github/workflows/release.yml.
verify_edit() {
	local file="$1" notes unreleased
	notes="$(sed -n "/^## \[${VERSION_NUM}\]/,/^## \[/{/^## \[${VERSION_NUM}\]/d;/^## \[/d;p;}" "$file" | tr -d '[:space:]')"
	[[ -n "$notes" ]] || fail "self-check failed: [${VERSION_NUM}] section is empty after edit"
	unreleased="$(sed -n "/^## \[Unreleased\]/,/^## \[/{/^## \[Unreleased\]/d;/^## \[/d;p;}" "$file" | tr -d '[:space:]')"
	[[ -z "$unreleased" ]] || fail "self-check failed: [Unreleased] is not empty after edit"
}

rehearse() {
	info "Phase 4: rehearsing the changelog edit (dry run)"

	local tmp
	tmp="$(mktemp)"
	insert_heading "$tmp"
	verify_edit "$tmp"
	grep -m1 "^## \[${VERSION_NUM}\]" "$tmp" | sed 's/^/    heading: /'
	rm -f "$tmp"

	cat <<EOF

  version:    $VERSION
  branch:     $BRANCH
  commit:     docs(changelog): prepare ${VERSION} release
  pr title:   docs(changelog): prepare ${VERSION} release

EOF
	info "dry run: all checks passed; nothing was created"
}

pr_body() {
	cat <<EOF
## Summary

- Move \`[Unreleased]\` items to \`[${VERSION_NUM}]\` section with release date
- This is the final PR before tagging ${VERSION}
EOF
}

create_pr() {
	info "Phase 5: creating the branch, commit, and PR"

	git checkout -b "$BRANCH"

	local tmp
	tmp="$(mktemp)"
	insert_heading "$tmp"
	cat "$tmp" >CHANGELOG.md
	rm -f "$tmp"
	verify_edit CHANGELOG.md

	git add CHANGELOG.md
	git commit -m "docs(changelog): prepare ${VERSION} release"
	git push -u origin "$BRANCH"
	gh pr create --title "docs(changelog): prepare ${VERSION} release" --body "$(pr_body)"
	info "PR opened for $VERSION"

	cat <<EOF

Next:
  1. Merge the PR (CI: title check, make check, make vuln).
  2. Pull main, then cut the release:
     scripts/release.sh $VERSION
  Note: release.sh warns when the changelog heading date is not the tag
  date, so prepare and tag on the same day.
EOF
}

main() {
	parse_args "$@"
	check_version_arg
	check_repo_state
	check_changelog
	if [[ "$DRY_RUN" == true ]]; then
		rehearse
		exit 0
	fi
	confirm "Create branch $BRANCH and open the changelog-prep PR for $VERSION?"
	create_pr
}

main "$@"
