# ADR-006: Rules-Driven Development for Game Implementations

**Date:** 2026-04-09
**Status:** Accepted

## Context
Card games have complex, precisely defined rules with edge cases that
are easy to miss in code but obvious in plain language. Without an
authoritative reference, implementations drift based on individual
memory or conflicting online sources, and there is no clear way to
verify correctness. We need a development approach that keeps the rules
as the source of truth.

## Decision
We adopt Rules-Driven Development (RDD) for all game implementations.
RDD means:

1. Before implementing a game, write a complete rules document in
   `doc/games/<game>/rules.md`. This document is the authoritative
   specification for the game's behavior.
2. The rules document must be precise and unambiguous, yet readable by
   non-technical card game players. It defines key terms and covers all
   edge cases. For established games, it cites primary references. For
   original games, the rules document itself is the authoritative
   definition.
3. The implementation is built against the rules document.
4. Tests verify that the implementation matches the rules.
5. When the rules document and code disagree, the rules document is the
   source of truth and the code must be fixed — unless the rules
   document itself needs an intentional update.
6. Variant rules are documented in a dedicated section of the same
   rules file, providing a clear specification for future variant
   support.

## Consequences
(+) Every rule has a single, traceable source of truth outside the code.
(+) Rules documents double as user-facing documentation.
(+) Variant support has a clear specification before any code is
    written.
(+) Contributors can review rules correctness without reading Go.
(-) Requires discipline to write the rules document first.
(-) Rules documents must be maintained in sync with code changes.
