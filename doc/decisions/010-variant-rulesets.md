# ADR-010: Variant Rulesets Are Fixed and Named

**Date:** 2026-08-25
**Status:** Accepted

## Context

Tiến Lên is the first cardcore game with more than one way to play: the
Southern baseline (Tiến Lên Miền Nam) and Killer, its San Diego
descendant. Other games on the roadmap have the same shape — Hearts has
Omnibus Hearts (the Jack of Diamonds as a bonus card), Durak has
Perevodnoy (transfer Durak).

The engine needs a policy for representing variants before the first
multi-variant game ships. The candidate mechanisms:

- **Fixed, named rulesets** selected at construction (a variant enum).
- **Per-rule option flags**, freely composable ("mix-and-match").
- **Rules-as-data** — callers compose arbitrary rule combinations.

Two observations discipline the choice.

First, the rules inside a real variant *co-vary*. Killer's divergence
from the baseline — clockwise play, no chops when locked out, self-beat,
its chop ladder, its settlement — is a
correlated cluster, not a set of independent dials; the correlation is
what makes the variant a distinct game people actually play. Every
intermediate combination ("Southern chops with Killer lockout") is a
game nobody plays. Free composition therefore buys a combinatorial space of
untested, unreal games, and an unbounded burden on AI players, which
would have to cope with arbitrary rule interactions.

Second, genuine one-rule customizations still arrive as named rulesets
in the wild. "Hearts where the Jack of Diamonds is a bonus card" is not
a free-floating flag; it is Omnibus Hearts, a documented, genuinely
played ruleset. A bounded set of named rulesets serves real
customization demand without pretending every combination is a game.

## Decision

A **ruleset** is a complete, named, fully specified way to play a game,
documented in the game's rules doc (see
[ADR-006](006-rules-driven-development.md)). A **variant** is a
ruleset that shares the game's core skeleton. A game and its variants
together form a game **family**.

1. **Variants are fixed rulesets.** Each variant is named and fully
   specified in the game's rules doc, with no open options.
2. **Selection, not configuration.** A game package takes its variant
   at construction as an enum value. The API exposes no per-rule option
   flags. Construction parameters that scale the game without changing
   what plays are legal (e.g. seat count, stake unit, RNG seed) may be
   supplied. A dimension such as hand or deck size may be a construction
   parameter only when no rule keys on its value; a parameter that
   changes what plays are legal is a rule, and rules belong to a
   variant.
3. **Admission bar.** A variant is added only for a ruleset people
   genuinely play, documented in the rules doc first (per
   [ADR-006](006-rules-driven-development.md)), then added as one enum
   value. Invented combinations of rule values
   are not admitted.
4. **One state machine per family.** Variants share a single state
   machine, parameterized at the divergence points the rules doc
   documents. A separate state machine or package per variant is
   forbidden — duplication rots. A would-be variant that diverges
   structurally — partnerships, cards drawn from a stock during play, a
   bidding phase with its own player decisions — is not a variant at
   all: it is a separate game with its own package and rules doc.
5. **AI adapts legally for free, competently per variant.** AI asks the
   engine what is legal — enumerating legal moves, testing whether one
   combination beats another — rather than hardcoding rule details (see
   [ADR-009](009-ai-difficulty-and-personality.md)), so an AI built on
   those queries plays legally under every variant with no code
   changes. Playing well is per-variant work, at three depths: a variant
   that moves only scoring or parameters is absorbed automatically by
   AI that queries the engine's scoring; a variant that keeps the
   mechanics but shifts strategy retunes evaluation weights within the
   same AI types; a variant that adds mechanics gets new AI logic in
   the same pull request. Each variant carries its own legality and
   competence test coverage.
6. **Variants may evolve; permanence is not promised.** A shipped
   variant's rules are corrected or refined when real-world information
   or feedback warrants, even when the change is breaking. The
   restraint is discipline, not prohibition: churn is costly, so
   breaking rule changes are made deliberately and called out in the
   changelog. Clients that need stability pin to a specific release
   rather than rely on a variant's name being immutable.

Rejected: rules-as-data, where callers supply the rules themselves and
the engine becomes a language for describing games (premature — see
the rationale in [ADR-005](005-no-premature-abstractions.md)); and
per-rule option flags (they admit games that are not real and an
untestable cross product). A tier of documented,
orthogonal rule options is introduced only when a concrete customization
demand arises that no real-world named ruleset covers.

## Consequences

- (+) Every supported variant is a real, documented, testable game.
- (+) Extensibility stays cheap: a new variant is a rules-doc amendment,
  one enum value, and its divergence parameters — with the family's
  skeleton, test fixtures, and AI implementations shared.
- (+) The variant enum makes lobbies, saves, and replays
  self-describing.
- (+) AI move legality is variant-agnostic via engine validation.
- (-) Each accepted variant is an ongoing maintenance commitment —
  whole-game tests exercising rule interactions, AI tuning, and
  changelog discipline when its rules evolve; the cheapness of adding
  one enum value disguises the real cost.
- (-) The maintainer is the gatekeeper for house rules; a home game
  differing by one rule from the nearest variant is unserved until its
  real-world ruleset is adopted.
- (-) Variant-level AI competence is per-variant work; only legality is
  guaranteed to transfer.
- (-) The parameter-vs-structural line in principle 4 is judgment, and
  both failure modes are expensive: over-parameterized state machines
  rot, and premature package splits duplicate the skeleton.
- (-) One-rule customizations are not expressible as flags; they arrive
  through the named rulesets that contain them (for example, the Jack
  of Diamonds via Omnibus Hearts), which may bundle rules a given group
  does not want.
