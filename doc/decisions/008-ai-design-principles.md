# ADR-008: AI Design Principles

**Date:** 2026-04-14
**Status:** Accepted

## Context
Cardcore games need computer-controlled players to fill empty seats in
local and online play. AI for card games involves unique challenges:
imperfect information (hidden hands), variant-specific strategy, and a
wide range of desired difficulty levels. Without clear principles, AI
implementations risk becoming tangled with engine internals, dependent
on external frameworks, or inconsistent across games.

## Decision
We adopt the following principles for all game AI in Cardcore.

### Location and Ownership
1. AI for each game lives in a subpackage of that game:
   `games/<game>/ai/`. The root package contains no AI logic.
2. The dependency direction is one-way: AI depends on the engine, never
   the reverse. The engine must function without any AI package.
3. Generic AI abstractions (shared interface, base type, etc.) are
   deferred until at least two games have working AI implementations,
   consistent with ADR-005.

### Interface
4. Each game package defines the interface its AI must satisfy.
5. AI must not mutate the live game state. A read-only AI satisfies
   this trivially. An AI that needs to simulate or look ahead must
   work on its own copy rather than on the live state.
6. Every AI decision is a pure function of visible state: given the
   same inputs, a deterministic AI must return the same output.
   Randomized AIs (such as Monte Carlo methods) accept an explicit
   random number source as a parameter so that callers can seed it
   for reproducible results. AI must not depend on global mutable
   state (package-level random sources, wall-clock time, environment
   variables, or filesystem).
7. AI interface methods do not return errors. The caller orchestrates
   the game loop and is responsible for calling methods at the correct
   time (for example, ChoosePass only during the pass phase, ChoosePlay
   only on the seat's turn). Implementations panic on precondition
   violations, consistent with the project's error handling principle
   (see [Error Handling](../design.md#error-handling)).

### Difficulty and Personality
8. Each difficulty level is a separate Go type that satisfies the
   game's AI interface. Difficulty is not a runtime parameter on a
   single type. Sharing implementation across difficulty levels via
   unexported helpers, embedded structs, or composition is encouraged
   — the rule constrains the public API surface, not the internals.
9. Difficulty levels control how deeply or broadly the AI searches
   (for example: number of determinizations, search depth, time
   budget). A determinization is a single random assignment of unknown
   cards to opponents; the AI simulates play in many such assignments
   and picks the move that performs best on average (see
   [Bax 2020, §2.3](https://studenttheses.uu.nl/bitstream/handle/20.500.12932/37736/Thesis_draft.pdf?sequence=1)
   for a clear explanation of this technique in trick-taking games).
   Personality — the AI's playing style — is controlled by evaluation
   weights within a given difficulty level. Difficulty and personality
   are independent axes.
10. Every game must provide at minimum a random-legal-move
    implementation. This is the baseline for testing, development, and
    filling seats when no smarter AI is available.

### Algorithms and Dependencies
11. AI implementations use only the Go standard library. No external
    machine-learning frameworks, neural network libraries, or
    third-party dependencies. Algorithmic approaches (heuristics, tree
    search, Monte Carlo methods) are preferred.
12. Third-party developers may build alternative AI implementations in
    their own repositories using any tools they choose, provided they
    satisfy the same interface. The built-in AI and third-party
    alternatives are independent — neither constrains the other's
    implementation approach.

### Variant and Rule Change Adaptation
13. AI should evaluate moves by querying the engine's scoring and
    validation logic rather than hardcoding rule details. When a
    variant changes only scoring or parameters (for example, a
    different point value for a card), the AI adapts automatically
    without code changes.
14. When a rule change introduces new mechanics or actions that
    existing AI strategies have no logic for (for example, mid-round
    passing or bidding), AI updates are required. Contributors must
    assess whether existing AI strategies still produce legal and
    reasonable play under the new rules. If not, AI updates are
    required in the same pull request as the rule change.

### Testing
15. Each AI difficulty level must have tests verifying it makes only
    legal moves across a statistically meaningful number of games.
16. Higher difficulty levels should be tested for basic strategic
    competence (for example, a heuristic AI should outperform random
    play over many games).

## Consequences
(+) AI is cleanly separated from engine logic — the engine works
    without it.
(+) The "no live mutation" rule isolates AI bugs from the engine:
    a buggy AI cannot corrupt the game it is playing in.
(+) Separate types per difficulty level make each level independently
    testable and prevent god functions.
(+) Zero external dependencies keep AI aligned with the project's
    suckless philosophy.
(+) Third-party developers can build alternative AI implementations
    against the same interface.
(+) Rule changes that affect AI are caught during review, not
    discovered after release.
(-) Each game's AI is self-contained, which means some structural
    patterns will be duplicated across games until a shared
    abstraction is justified (per ADR-005).
(-) The stdlib-only constraint (principle 11) limits AI to algorithmic
    approaches, which may hit a ceiling for games where learned
    strategies significantly outperform search (for example, poker).
