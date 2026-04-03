# ADR-005: No Generic Abstractions Before Two Games

**Date:** 2026-04-03
**Status:** Accepted

## Context
A common instinct in engine design is to immediately create generic `Player`, `GameState`, `Rules`, etc. interfaces. These feel clean in theory but are shaped by the first game's assumptions, and rarely fit the second game without significant rework.

## Decision
We will not extract generic Player, GameState, Zone, or Rules abstractions until we have implemented at least two complete games (Hearts + one more, likely Tiến Lên or Durak). At that point, real duplication will be visible and abstractions can be extracted from actual code rather than imagined requirements. Hand is the only shared collection type; it was necessary from the start.

## Consequences
(+) No wasted effort on abstractions that may not fit. (+) Each game's logic is self-contained and easy to read. (+) Second game implementation will reveal what's truly common. (-) Some code duplication between games until extraction. (-) Must resist the urge to generalize early.
