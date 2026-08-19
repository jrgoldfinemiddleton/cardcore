# Releasing cardcore

This document is the authoritative release process for maintainers. Keep it
in sync with `scripts/release.sh` and `.github/workflows/release.yml`.

## Principles

- **Semantic versioning**, pre-1.0. Releases v1.0.0 and higher are never
  tagged; the release script rejects them.
- **Tags are permanent.** The Go module proxy caches a version on first
  fetch, the repository's tag ruleset restricts `refs/tags/v*` to
  administrators, and GitHub release immutability is enabled. Never re-tag,
  never reuse a version number, never delete a tag.
- **Annotated tags only.** The script creates annotated tags
  (`git tag -a`). Lightweight tags are a historical accident (v0.1.0–v0.6.0),
  not a supported path.
- **Fix forward.** If a release has a problem, land the fix on `main` and
  tag the next patch version. Do not attempt to repair a published release.
- **Releases are never cut by hand.** `scripts/release.sh` is the single
  entry point; it runs every check before creating anything.

## Release artifacts

Cardcore is a library: the tag itself is the release. The Go module proxy
caches the tagged source on first fetch, and the release workflow creates a
GitHub Release whose notes are the hand-curated `## [X.Y.Z]` section of
`CHANGELOG.md`. There are no binaries and no uploaded assets.

## Release procedure

### 1. Prepare the changelog (its own PR)

1. Ensure everything for the release is merged to `main`.
2. In `CHANGELOG.md`, create a `## [X.Y.Z] - YYYY-MM-DD` heading (today's
   date) directly below `## [Unreleased]`, move all `[Unreleased]` items
   into the new section, and leave `[Unreleased]` empty.
3. Open a PR titled `docs(changelog): prepare vX.Y.Z release` and merge it.
4. Optional: run the benchmark spot-check against the previous tag
   (`make bench` + `go tool benchstat`) and note any >2x regressions in the
   release notes.

### 2. Cut the release

```bash
scripts/release.sh vX.Y.Z            # or: --dry-run to rehearse first
```

The script verifies, in order:

1. The version is strict semver, pre-1.0, and greater than every existing
   tag.
2. The tree is on `main`, clean, and exactly at `origin/main`.
3. `CHANGELOG.md` has a dated, non-empty `[X.Y.Z]` section and an empty
   `[Unreleased]` (using the same extraction as the release workflow).
4. `make check` passes.

Only then does it create the annotated tag (`git tag -a vX.Y.Z -m "Release
vX.Y.Z"`) and push it. The interactive confirmation gates can be bypassed
with `--yes`.

### 3. The release workflow

The pushed tag triggers `.github/workflows/release.yml`, which:

1. Verifies the tag points at a commit on `main`.
2. Validates the tag is proper semver (no leading zeros).
3. Runs `make check`.
4. Extracts the `[X.Y.Z]` changelog section to `release-notes.md`.
5. Creates the GitHub Release titled `vX.Y.Z` with the changelog notes.

### 4. Post-release verification

The script prints this checklist; do every step.

1. The release workflow run is green.
2. The GitHub Release exists with the changelog section as its notes.
3. The Go module proxy serves the version (the script already curls
   `https://proxy.golang.org/github.com/jrgoldfinemiddleton/cardcore/@v/vX.Y.Z.info`).
4. Visit `https://pkg.go.dev/github.com/jrgoldfinemiddleton/cardcore@vX.Y.Z`
   to trigger documentation indexing.

## Recovery

**The release workflow failed after the tag was pushed.** The tag is
permanent; assume the module proxy has already cached it. Fix the cause on
`main` (workflow or changelog), then tag the next patch version. Do not
delete or move the failed tag.

**Security fix.** Land the fix, then immediately release a new patch
version. Never silently patch.
