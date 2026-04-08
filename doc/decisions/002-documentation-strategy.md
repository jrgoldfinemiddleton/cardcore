# ADR-002: Documentation Structure

**Date:** 2026-04-03
**Status:** Accepted

## Context
The project needs documentation that serves multiple audiences: contributors, AI co-developers, and future maintainers. We want to avoid over-documenting early while still making key decisions legible.

## Decision
Three-tier doc structure: (1) `README.md` — project overview and quick start; (2) `doc/design.md` and `doc/architecture.md` — deeper design and system explanations; (3) `doc/decisions/*.md` — ADRs for significant choices. Go doc comments on all exported symbols provide API documentation via `pkgsite` or `go doc`.

## Consequences
(+) Each layer has a clear purpose. (+) README stays short. (+) ADRs are self-contained. (-) Requires writers to choose the right layer for each piece of information.
