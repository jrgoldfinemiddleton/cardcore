# ADR-006: Rules-Driven Development for Game Implementations

**Date:** 2026-04-12
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
6. Variants are specified in a dedicated section of the same rules
   document.
7. Rules documents are living specifications. They may be updated after
   implementation to correct errors, improve clarity, or document new
   edge cases.
8. When a rules change alters specified behavior and an implementation
   exists, the same pull request (PR) must include corresponding
   implementation and test updates. Clarifications that do not change
   behavior may be submitted without code changes.
9. New variants may be added to the rules document without a
   corresponding implementation unless the variant fits into an
   already-implemented category (where omitting the implementation
   creates an obvious gap) or is a trivial extension of existing
   code.
10. For established games, the primary references cited in the merged
    rules document represent the accepted authority for that game.
    Proposing a different primary reference whose rules conflict with
    the current one requires a GitHub Discussion before any PR is
    created.

## Consequences
(+) Every rule has a single, traceable source of truth outside the code.
(+) Rules documents double as user-facing documentation.
(+) Variants have a clear specification independent of implementation
    status.
(+) Contributors can review rules correctness without reading Go.
(+) Implemented games have stable rules — behavioral changes require
    code updates in the same PR, preventing spec/code drift.
(+) Primary reference disputes are resolved through discussion, not
    competing PRs.
(-) Requires discipline to write the rules document first.
(-) Rules documents must be maintained in sync with code changes.
(-) Changing the behavior of an implemented game is costly — the
    contributor must update rules, implementation, and tests together.
