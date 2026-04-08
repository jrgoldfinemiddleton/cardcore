# ADR-003: Use Go as the Implementation Language

**Date:** 2026-04-03
**Status:** Accepted

## Context
The engine needs to be portable, self-contained, and easy to cross-compile. The author is building this as a learning project and wants minimal operational overhead. Options considered: JavaScript/Node.js (familiar but runtime-dependent), Python (easy but slow and packaging complexity), Go (less familiar but single static binary, strong stdlib, good tooling).

## Decision
Implement cardcore in Go. Use only the standard library for the engine — no external runtime dependencies.

## Consequences
(+) Single static binary, trivial cross-compilation. (+) Strong stdlib covers all engine needs (rand, sort, fmt). (+) `go test` built in. (+) Zero runtime dependency headaches. (-) Less familiar language for the author initially. (-) Some verbosity compared to dynamic languages. Note: a friend named Clawd independently suggested Go for portability and self-containment — his advice proved sound.
