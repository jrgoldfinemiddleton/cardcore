# Hearts AI: PIMC Design

This document describes the Perfect Information Monte Carlo (PIMC) implementation of the Hearts `Player` interface. `PIMC` is one of several Hearts AI implementations; per [ADR-009](../../decisions/009-ai-difficulty-and-personality.md), each implementation is its own Go type, and the mapping of types to client-facing difficulty labels is a decision made by clients of this library.

The intended audience is a contributor reading the `games/hearts/ai/pimc*.go` source for the first time, or a maintainer modifying it. The doc assumes Go fluency and familiarity with the [Hearts rules](rules.md) and the project's [AI principles](../../decisions/009-ai-difficulty-and-personality.md). It does **not** assume prior background in game-playing AI or game theory.

**Reading convention.** Glossary terms appear **[in bold and linked](#10-glossary)** at first use; click to jump to the [Glossary](#10-glossary). Plain bold (no link) is just emphasis.

## Contents

1. [Overview](#1-overview)
2. [Algorithm](#2-algorithm)
3. [Information leakage discipline](#3-information-leakage-discipline)
4. [API & package layout](#4-api--package-layout)
5. [Testing strategy](#5-testing-strategy)
6. [Benchmarking methodology](#6-benchmarking-methodology)
7. [Reproducibility & testing infrastructure](#7-reproducibility--testing-infrastructure)
8. [Design rationale](#8-design-rationale)
9. [Future work](#9-future-work)
10. [Glossary](#10-glossary)
- [Appendix A: Exact uniform DP sampler — worked example](#appendix-a-exact-uniform-dp-sampler--worked-example)

## 1. Overview

### What PIMC does

Hearts is a game of imperfect information: when it's your turn to play a card, you know your own hand and the cards already played, but you do not know how the remaining unseen cards are distributed among your three opponents. A perfect-information solver — one that knows every hand — could compute the optimal play exactly. The Hearts rules forbid this.

`PIMC` bridges the gap by **repeatedly imagining a perfect-information game**:

1. **Sample.** Randomly distribute the unseen cards (those not in `seat`'s own hand and not yet played) among opponents in a way consistent with everything publicly known (cards played, voids revealed, pass information you legitimately hold). Each such distribution is called a **[determinized deal](#determinized-deal)** — note that this is a partial deal of the unseen cards at decision time, not the full 52-card distribution from the start of the round.
2. **Solve each deal.** For each candidate card you could legally play, simulate the rest of the round under the sampled deal and record the outcome. This forward simulation is called a **[rollout](#rollout)**.
3. **Average and pick.** Across many sampled deals, compute each candidate's mean outcome. Play the candidate with the best average.

The high-level intuition is that **averaging across many plausible worlds approximates reasoning about the actual world**, even though no individual sample is the real one.

### Why this works

`PIMC`'s correctness rests on three things:

1. **Sample consistency.** If samples respect public information, the average is over plausible worlds, not impossible ones.
2. **[Common random numbers](#common-random-numbers).** Every candidate is evaluated against the *same* set of sampled deals, so noise affects them equally and cancels in comparison. This is the same idea as paired A/B testing: paired noise is much smaller than independent noise.
3. **Convergence.** As sample count grows, the estimate of each candidate's mean approaches its true mean (under the assumed opponent model). The remaining gap is **sampling noise** — the random difference between the average over our `N` sampled deals and the true average over all possible deals. This noise shrinks roughly as 1/√N: quadrupling the sample count halves it.

`PIMC`'s main structural limitation, **[strategy fusion](#strategy-fusion)**, is discussed in [§2](#strategy-fusion-pimcs-structural-ceiling).

### What PIMC is not

To set expectations:

- **PIMC is not a perfect player.** It is bounded by strategy fusion ([§2](#strategy-fusion-pimcs-structural-ceiling)) and by the realism of its **[rollout policy](#rollout-policy)**. It will sometimes make plays that look incorrect to a strong human.
- **PIMC does not learn.** It has no training phase, no model that improves over time, no memory across games. Each decision is computed from scratch.
- **PIMC does not optimize game score.** It optimizes expected *round* score per ADR-009. Game-level strategy — choosing to incur points now to deny an opponent a winning hand later — would require a leaf score that incorporates `g.Scores`, or a different algorithm entirely.
- **PIMC does not handle the pass phase.** `ChoosePass` is delegated to an internal `Heuristic`. Pass-phase PIMC is out of scope for this implementation; see [§9](#9-future-work) for what an alternative approach would require.

### Expected quality

`PIMC` is expected to outperform `Heuristic` on average over many rounds. The mechanism: `PIMC` simulates specific future card distributions and picks the move that performs best across them, while `Heuristic` applies static rules without any forward simulation.

The claim is *average* superiority, not per-round superiority. `PIMC` will sometimes lose individual rounds to `Heuristic` due to sampling variance and strategy fusion ([§2](#strategy-fusion-pimcs-structural-ceiling)). The expected gap will be measured empirically by the tournament harness once it exists; until then, the claim is justified by the algorithm's design rather than by data.

## 2. Algorithm

### Inputs to a decision

A single call to `ChoosePlay(g, seat)` receives:

- `g *hearts.Game` — the live game state. `PIMC` must read only fields that `seat` is legitimately allowed to know (own hand, own pass history, all public history). The `Player` interface technically permits reading any field; `PIMC` restricts itself by convention. See [§3](#3-information-leakage-discipline).
- `seat hearts.Seat` — whose turn it is.

`PIMC` also has, from its constructor, captured **[RNG](#rng-random-number-generator)** seed material, a sample count `N`, a rollout policy factory, and a worker count `W`. See [§4](#4-api--package-layout). The seed material is split into three independent slots — sampling, tiebreaks, and pass delegation — that cannot perturb each other; per-call RNGs are derived freshly from those slots plus a [game-state fingerprint](#game-state-fingerprint), as described in [§7](#7-reproducibility--testing-infrastructure).

### Step 1: Analyze public state

`PIMC` calls `analyze(g, seat)` (existing function in `analysis.go`) to extract the constraints that all sampled deals must satisfy:

- Cards already played in completed tricks (per suit, per rank).
- Known voids per opponent and suit (`voids[opponent][suit]`). An opponent is marked void in a suit if it failed to follow suit on a completed trick, or if all remaining cards of that suit are accounted for between played cards and the PIMC seat's hand (suit exhaustion).
- Counts of cards remaining in each opponent's hand.

The sampler must additionally apply two layers that `analyze()` does not currently cover:

- **Current-trick state.** Cards already played to the in-progress trick are visible to all and must be excluded from the unseen pool. Hand counts for those opponents must be decremented accordingly. If an opponent failed to follow suit during the current trick, that void is also public and must be honored.
- **Hard pass constraint from `g.PassHistory[seat]`.** For each card `seat` passed, if that card has not been played (in any completed trick or in the current trick), the recipient still holds it. This is a **hard** constraint, not a probabilistic prior: in this engine cards leave a hand only by public play, so an unplayed passed card is definitely in the recipient's hand. Every sampled deal must place such cards with the recipient. `PIMC` must use only `g.PassHistory[seat]` and never `g.PassHistory[other]` for any `other != seat`.

### Step 2: For each sample i = 1..N

Samples are independent and `PIMC` runs them in parallel across `W` worker goroutines (`W` is a constructor parameter; see [§4](#4-api--package-layout)). Each worker pulls samples from a shared queue and writes results into an index-keyed slot ([§7](#7-reproducibility--testing-infrastructure)). The per-sample work:

1. **Derive a per-sample RNG.** `RNG_i` is built freshly from `(sampleSeed, fingerprint, i)` where `sampleSeed` is the captured material from `NewPIMC` and `fingerprint` is a deterministic digest of `(g, seat)` computed once at the top of `ChoosePlay`. Same captured material + same input ⇒ same `RNG_i`, regardless of worker count, completion order, or any prior `ChoosePass`/`ChoosePlay` calls; details in [§7](#7-reproducibility--testing-infrastructure).

2. **Generate a consistent deal.** Randomly assign the unseen cards to opponents respecting all constraints from step 1 (completed-trick history, voids, hand counts, current-trick state, and hard pass constraint). The output is a `[4]cardcore.Hand` where `seat`'s entry is `seat`'s real hand and the other three entries are sampled. Sampling draws **uniformly** from the set of feasible deals: it first applies hard pass-card assignments to fix recipient capacity, then uses a suit-by-seat dynamic-programming pass over the remaining unseen cards to count feasible completions per branch and walk that DP using `RNG_i` proportionally to those counts. The DP is computed once per `ChoosePlay` (constraints are identical across samples) and reused for all `N` samples. Rationale for choosing this over rejection sampling is in [§8](#why-exact-uniform-dp-sampling-instead-of-rejection); a worked example is in [Appendix A](#appendix-a-exact-uniform-dp-sampler--worked-example); full algorithm in `pimc_sample.go`.

3. **For each candidate card `c` in the legal moves (indexed `j = 0..len(legal)-1`):**
   1. Derive a fresh rollout RNG: `RNG_ij = deriveRNG(sampleSeed, fingerprint, i, j)`. Build a fresh rollout policy via `rolloutFactory(RNG_ij)`. Both are scoped to this single `(sample, candidate)` pair.
   2. Clone `g`.
   3. Install the sampled deal into the clone.
   4. Play `c` from `seat`.
   5. Run the rollout policy on all subsequent decisions until the round ends.
   6. Read the leaf score (below).
   7. Record the result keyed by `(i, j)` in the per-sample slot allocated for this worker; aggregation shape in [§7](#7-reproducibility--testing-infrastructure).

The rollout policy is created fresh for each `(sample, candidate)` pair via `rolloutFactory(RNG_ij)`. All seats in the rollout use the same **[policy](#policy)** (symmetric assumption) — including `seat` itself for all decisions *after* the candidate `c`. `PIMC` commits to `c` as `seat`'s next move, then defers all of `seat`'s subsequent moves to the rollout policy. This keeps every seat on the same behavior model and avoids the question of how `PIMC` would simulate its own future decisions (see [§8](#why-use-the-rollout-policy-for-seats-own-future-moves)).

Per-pair freshness is what makes [common random numbers](#common-random-numbers) actually pair candidates: every candidate within sample `i` faces the same sampled deal *and* a rollout policy built from RNG material that depends only on `(i, j)`, never on the order in which candidates were evaluated or how many random draws prior candidates made.

The sampling stage relies on one engine invariant: the feasible-deal set is non-empty whenever `analyze(g, seat)` returns successfully on a reachable game state. The sampler treats an empty feasible set as a programmer error (an unreachable or malformed game state was supplied) and panics, per the project's error-handling convention. A test in [§5](#5-testing-strategy) validates this behavior.

The choice of rollout policy is a model of opponent behavior; in practice we assume opponents play approximately like our `Heuristic`. The reason rollouts cannot themselves use PIMC is covered in [§8](#why-dont-rollouts-use-pimc-itself).

### Step 3: Compute the leaf score

After the rollout reaches the end of the round, the cloned game's `RoundPts[s]` field holds the raw points each seat has accumulated this round (before any moon flip is applied). The **[leaf score](#leaf-score)** for `seat` is:

```
shooter = the seat s with RoundPts[s] == 26, if any
if shooter exists:
    if shooter == seat: leafScore = 0
    else:               leafScore = 26
else:
    leafScore = RoundPts[seat]
```

This formula re-implements the moon flip locally inside `PIMC` rather than reading `g.Scores`. The reasoning is in [§8](#why-compute-leaf-score-from-roundpts-instead-of-scores).

Lower leaf scores are better.

### Step 4: Aggregate and choose

After all `N` samples have produced their per-candidate leaf scores, `PIMC` selects the candidate with the **lowest total leaf score**. (Every candidate is evaluated on the same `N` deals, so lowest total is equivalent to lowest mean; summing avoids floating-point arithmetic.)

Ties are possible. When two or more candidates share the lowest total, `PIMC` picks one uniformly at random from the tied set. The randomness for this draw comes from an RNG derived from `PIMC`'s captured `tiebreakSeed`, which is independent of `sampleSeed`, so the tiebreak cannot be perturbed by which deals were sampled. The exact tiebreak rule is an implementation detail covered in [§4](#tiebreak-rule); the independence mechanism is covered in [§7](#7-reproducibility--testing-infrastructure).

### Strategy fusion: PIMC's structural ceiling

PIMC computes the best average move *as if* the cards' true distribution were known at decision time. But in real Hearts, your strategy must work *without* that knowledge — you cannot, for example, play one card if East has the Q♠ and a different card if West has it. You must commit to one plan that's robust across all possibilities.

PIMC ignores this constraint. It picks the move that wins *most* sampled deals, even if that move only wins by exploiting deal-specific information no real player would have. This is **strategy fusion**: PIMC fuses strategies that depend on hidden information into a single recommendation, as *as if* you could decide differently based on facts you cannot see.

The consequence: PIMC sometimes recommends moves that would be correct *if* you had perfect information, but are dominated by safer moves under genuine uncertainty. This is the price of using sampling-then-solving instead of a true imperfect-information solver.

There is no fix within the PIMC framework itself. Removing strategy fusion would require a different algorithm — for example, [information-set MCTS](#information-set-mcts) or [counterfactual regret minimization](#counterfactual-regret-minimization-cfr) — that operates on information sets rather than samples of perfect-information games. Belief modeling can refine *which* deals are sampled but does not by itself remove strategy fusion. Either direction is out of scope here; see [§9](#9-future-work).

### Sample budget

`N` (sample count) is a constructor parameter. Higher `N` gives more accurate estimates (variance shrinks as 1/√N) at proportionally higher cost. Choosing a good `N` for a given latency budget is covered in [§6](#6-benchmarking-methodology).

There is exactly **one rollout per (sample, candidate) pair**. The reasoning for this budget allocation is in [§8](#why-one-rollout-per-sample-candidate-pair).

All candidates are evaluated against the *same* set of sampled deals (common random numbers / paired comparison). Without this pairing, distinguishing two similarly-scoring candidates would require many more samples.

## 3. Information leakage discipline

The `hearts.Player` interface receives `*hearts.Game` and can technically read any field. `PIMC` restricts itself to fields `seat` is legitimately allowed to know.

**Allowed reads:**

- `seat`'s own hand.
- `g.PassHistory[seat]` — the cards `seat` passed and to whom.
- All public engine state: `Phase`, `Turn`, `TrickNum`, `HeartsBroken`, `PassDir`, `Round`, `RoundPts`, `Scores`, completed tricks, the in-progress trick, public voids derived from observed play.
- Any other engine field that encodes only information `seat` could legitimately know from observing the game.

**Forbidden reads:**

- Opponent hands (`g.Hands[other]` for any `other != seat`).
- `g.PassHistory[other]` for any `other != seat`.
- Any other field that encodes information `seat` could not legitimately have.

This discipline is enforced two ways:

1. **Code review.** `PIMC`'s source touches `g` in a small, auditable set of places. Any new read must be justified.
2. **Fuzz test.** A unit test ([§5](#5-testing-strategy)) constructs a game state, runs `ChoosePlay`, then mutates *any* field not on the allowed list to arbitrary legal-but-different values, runs `ChoosePlay` again with the same captured seed material, and asserts the chosen card is unchanged. If `PIMC` ever reads a forbidden field, this test catches it.

## 4. API & package layout

### Package location

`games/hearts/ai/`, alongside `Random` and `Heuristic`.

### File split

Production code splits across four files, each with a matching `_test.go`:

| File | Responsibility |
|---|---|
| `pimc.go` | `PIMC` type, `NewPIMC`, `ChoosePass`, `ChoosePlay`, top-level orchestration of the per-decision pipeline. |
| `pimc_sample.go` | Constraint-satisfying random partition of unseen cards across opponents. |
| `pimc_rollout.go` | Rollout execution: clone game, install sampled deal, play candidate, run rollout policy to terminal state, extract leaf score. |
| `pimc_aggregate.go` | Per-candidate score totaling and deterministic tiebreak. |

This split mirrors the four phases of [§2](#2-algorithm) (analyze in `pimc.go`, sample, rollout, aggregate). Each file is independently testable; see [§5](#5-testing-strategy).

### Exported surface

```go
// PIMC is a Hearts player that chooses plays via Perfect Information
// Monte Carlo sampling.
type PIMC struct { /* unexported */ }

// NewPIMC constructs a PIMC player.
//
// Panics if rng is nil, samples <= 0, rolloutFactory is nil, or
// workers <= 0.
func NewPIMC(
    rng *rand.Rand,
    samples int,
    rolloutFactory func(*rand.Rand) hearts.Player,
    workers int,
) *PIMC

func (p *PIMC) ChoosePass(g *hearts.Game, seat hearts.Seat) [hearts.PassCount]cardcore.Card
func (p *PIMC) ChoosePlay(g *hearts.Game, seat hearts.Seat) cardcore.Card
```

That is the entire exported surface: one type, one constructor, two methods. No convenience constructors, no functional options, no setters. Rationale in [§8](#why-no-default-convenience-constructor).

`ChoosePass` delegates to an internal `Heuristic` instance constructed at `NewPIMC` time. Its RNG is derived per call from the captured `passSeed` and the [game-state fingerprint](#game-state-fingerprint) of `(g, seat)`, independently of the captured material used for sampling and tiebreaks; calling `ChoosePass` cannot perturb the determinism of subsequent `ChoosePlay` calls, regardless of order or how many times either method is called. The independence mechanism is covered in [§7](#7-reproducibility--testing-infrastructure). Pass-phase PIMC is out of scope.

### Internal types

```go
// sampledDeal is one constraint-satisfying assignment of unseen cards.
// Index by hearts.Seat. The PIMC player's own entry equals its real hand.
type sampledDeal [4]cardcore.Hand

// sampleResult holds the leaf score for a single (sample, candidate) pair.
type sampleResult struct {
    candidate cardcore.Card
    leafScore int
}

// Per-sample results are written into slot i of a [N][]sampleResult,
// where the inner slice has one entry per legal candidate (indexed by j).
// Index-keyed writes (no append) make worker completion order irrelevant
// to aggregation (§7).
```

### Tiebreak rule

After total leaf scores are computed, the candidate with the lowest total wins. If two or more candidates tie at the minimum, `PIMC` picks one uniformly at random from the tied set.

The randomness for this draw comes from an RNG derived per call from the captured `tiebreakSeed` and the [game-state fingerprint](#game-state-fingerprint), independently of the captured material used for sampling. Sampling cannot perturb tiebreak outcomes, and the tiebreak is deterministic given the seed material captured at `NewPIMC` time. The independence mechanism is covered in [§7](#7-reproducibility--testing-infrastructure).

### Constructor argument validation

`PIMC` follows the project's error-handling convention: caller programming errors panic; only conditions the caller cannot prevent return errors. `NewPIMC` has no error-returning conditions, so it returns just `*PIMC`. The four panic conditions listed in the docstring are all programmer errors: passing `nil` where a value is required, or a non-positive count where a positive count is required.

## 5. Testing strategy

### Four-layer decomposition

The four-file split from [§4](#4-api--package-layout) carries through to tests: each file has a matching `_test.go`, and each layer is testable without the others. The layers and their test files:

| File | Test file | Layer focus |
|---|---|---|
| `pimc.go` | `pimc_test.go` | End-to-end `ChoosePlay` on constructed positions; determinism contract; information-leakage fuzz. |
| `pimc_sample.go` | `pimc_sample_test.go` | Sampling: constraint satisfaction, distribution non-degeneracy, determinism. |
| `pimc_rollout.go` | `pimc_rollout_test.go` | Rollout: leaf-score correctness, termination, input non-mutation. |
| `pimc_aggregate.go` | `pimc_aggregate_test.go` | Aggregation: score totaling, min selection, tiebreak. |

### Per-layer tests

**Sampling (`pimc_sample_test.go`).** Given fixed analysis output and a fixed RNG seed, sample `N` deals and assert:

- Every sampled deal respects all constraints: cards already played are absent from all hands; voids are honored; per-opponent hand counts are correct; the `PIMC` player's own hand is unchanged.
- The sample distribution is non-degenerate. Across `N >= 100` samples, more than one distinct deal appears (rules out trivially broken sampling).
- **Uniformity over a known-small fixture.** Using the small-void-constraints fixture (42 feasible deals), sample 10,000 deals and assert that at least 30 distinct deals appear and no single deal exceeds 2× the expected frequency.
- **Infeasible-position detection.** Constructing the DP table for a position where void constraints forbid every possible distribution produces a root count of zero.
- Determinism: same seed produces the same sequence of deals.

**Rollout (`pimc_rollout_test.go`).** Given a constructed mid-round game state, a fixed sampled deal, and a deterministic rollout policy, assert:

- The leaf score matches a hand-computed expected value for at least one fully-worked example.
- The rollout terminates (round actually completes; no infinite loop).
- The input `*hearts.Game` is not mutated. Compare a deep snapshot before and after.

**Aggregation (`pimc_aggregate_test.go`).** Given fabricated `[]sampleResult` slices (no game engine involvement), assert:

- Total leaf score per candidate is computed correctly.
- The candidate with the lowest total is selected.
- Tiebreak: when two candidates tie at the minimum, the choice is uniform across the tied set using an RNG derived from `PIMC`'s captured `tiebreakSeed`, and is deterministic given the seed material captured at construction.

**Top-level (`pimc_test.go`).** Three categories:

- **Constructed-[position](#position) tests.** Build a small game state where the optimal play is unambiguous (e.g., a forced trick win, an obvious low-card slough into a known-safe trick) and assert `PIMC` chooses it.
- **Determinism contract test ([§7](#7-reproducibility--testing-infrastructure)).** Same `(seed, g, seat)` produces the same card across multiple invocations and across worker counts.
- **Information-leakage fuzz test ([§3](#3-information-leakage-discipline)).** Run `ChoosePlay`, then mutate forbidden fields (opponent hands, `g.PassHistory[other]`) to arbitrary legal-but-different values, run again with the same seed, assert the chosen card is unchanged.

A `TestPIMCFullGameIntegration` test in `pimc_test.go` exercises a full PIMC-vs-PIMC game from start to terminal state, mirroring the existing `TestFullGameIntegration` (Random) and `TestHeuristicFullGameIntegration` patterns in the same package.

### Cross-layer property coverage

Two contracts span layers:

1. **Sample consistency.** `TestSampleDealStructure` verifies that every sampled deal satisfies all constraints (voids, hand counts, unseen-card pool) on a fixed seed and game state.
2. **Determinism under parallelism.** `TestPIMCChoosePlayDeterminismAcrossWorkers` verifies that the same `(seed, g, seat)` produces the same card for `W` in {1, 2, 4, 8, 16}.

### What is deliberately not unit-tested

- **Strategy quality.** Whether `PIMC` plays *well* is a tournament-harness concern (Phase 2; [§9](#9-future-work)), not a unit-test concern. Unit tests check structure and contracts; the harness checks performance against other players over many rounds.
- **Specific card choices in non-trivial positions.** Asserting "`PIMC` plays the J♠ here" would couple tests to implementation details (RNG consumption order, sampling algorithm internals) that may legitimately change. Constructed-position tests are restricted to positions where the choice is forced or near-forced regardless of internals.

### Test fixtures

Reuse the existing `games/hearts/ai/` test conventions: `rAce..rKing` and `sClubs..sSpades` aliases (defined in `helpers_test.go`), deterministic `*rand.Rand` constructors, and shared trick-history helpers like `validFirstTrick()`. `PIMC` tests should add to the shared helpers file rather than duplicating fixture code.

## 6. Benchmarking methodology

### What we measure

The headline metric is **per-decision wall time** for `ChoosePlay`. This is what determines whether `PIMC` is fast enough to use in a real game. Allocations per decision (`-benchmem`) are a secondary metric, useful for spotting allocation hotspots that drive GC pressure under sustained play.

### Per-phase benchmarks

`PIMC` ships six benchmarks: four per-decision benchmarks in `pimc_bench_test.go` and two round/game-level benchmarks in `round_bench_test.go`.

| Benchmark | Measures |
|---|---|
| `BenchmarkPIMCPlay` | End-to-end `ChoosePlay` (N=30, W=4), iterating the six canonical `benchFixtures()`. |
| `BenchmarkPIMCPlayWithClone` | Same, plus `Game.Clone()` per iteration. The diff against `BenchmarkPIMCPlay` quantifies clone cost in the `PIMC` hot path. |
| `BenchmarkPIMCPlaySamples` | Sample-count sweep (N ∈ {10, 30, 100}) on the `follow_with_points` fixture. |
| `BenchmarkPIMCPlayWorkers` | Worker-count sweep (W ∈ {1, 2, 4, 8}) on the `follow_with_points` fixture. |
| `BenchmarkRoundPIMC` | Per-decision cost amortized across a full round of `PIMC` play. |
| `BenchmarkFullGamePIMC` | Per-decision cost amortized across a full game. |

### Sweeps

Two sweeps, each varying one dimension while holding the others fixed:

- **Sample count.** `N ∈ {10, 30, 100}` at fixed `W=4`. Establishes the per-decision time vs. sample count curve, which is approximately linear in `N`.
- **Worker count.** `W ∈ {1, 2, 4, 8}` at fixed `N=30`. Establishes the parallel speedup curve. Sub-linear scaling is expected (memory bandwidth, scheduler overhead); the question is *how* sub-linear.

Per-[position](#position) cost variation is already exercised by the six `benchFixtures()` sub-benchmarks — there is no need for a separate position sweep.

### Fixtures

Reuse `benchFixtures()` from the existing `bench_helpers_test.go` for all per-decision `PIMC` benchmarks. Per-decision cost varies dramatically across the six fixtures (more legal moves and deeper rollouts both inflate cost), so the per-fixture breakdown is informative on its own. `PIMC` adds a constructor helper for fixed `(samples, workers, seed)` tuples to make benchmark runs reproducible.

### What we don't benchmark

- **Strategy quality.** Whether `PIMC` plays well is a tournament-harness concern ([§9](#9-future-work)), not a benchmark concern. Benchmarks measure speed; the harness measures skill.
- **Total game wall time.** A full game involves dozens of decisions plus engine work; the resulting single number (seconds per game) is not actionable for tuning `PIMC`. `BenchmarkPIMCFullGame` measures the related but different quantity *per-decision* time amortized across a full game, which *is* actionable.

### Tooling

`make bench` runs all benchmarks in the package. Comparison across runs uses `go tool benchstat` (`golang.org/x/perf/cmd/benchstat`), which is declared in the project's `tool` directive and compiled automatically on first use.

## 7. Reproducibility & testing infrastructure

This section is about how `PIMC` is made reproducible — for testing, debugging, and tournament replay. A reader who only wants to understand how `PIMC` chooses a card can skip it. The "how it plays" story ends at [§6](#6-benchmarking-methodology); this section is for the "how it's tested" story.

### The guarantee

Given the same seed material captured at `NewPIMC` time and the same `(g, seat)` input, `ChoosePlay` returns the same card every time. This holds regardless of:

- Worker count `W`.
- OS thread scheduling.
- The order in which per-sample workers finish.
- Wall-clock time taken per sample.
- Whether `ChoosePass` was called previously on the same `PIMC` instance.

This is the property that makes `PIMC` testable: a failing test can be replayed exactly, a tournament result can be re-derived from its seed, and a regression can be bisected against a known seed.

### How it's achieved

Parallelism is what makes determinism non-trivial. `PIMC` runs its `N` samples in parallel across `W` workers ([§2 Step 2](#step-2-for-each-sample-i--1n)); without care, results could depend on which worker happens to finish first. Three steps together prevent that:

1. **Capture seed material at construction.** `NewPIMC` reads private seed material from the caller's RNG exactly once, then never touches it again. The captured material lives in three immutable slots on the `PIMC` value: `sampleSeed`, `tiebreakSeed`, and `passSeed` (each a `[2]uint64`, sized to seed a PCG — the algorithm Go's `math/rand/v2` uses by default — independently). Nothing in `PIMC` ever mutates these slots after construction.

2. **Derive per-call RNGs from `(slot, fingerprint, indices)`.** At the top of each `ChoosePlay`, `PIMC` computes a [game-state fingerprint](#game-state-fingerprint) — a deterministic 64-bit digest of the public-and-own-hand parts of `(g, seat)`. Whenever the algorithm needs randomness, it derives a fresh `*rand.Rand` by calling `deriveRNG(slot, fingerprint, indices...)`, where `slot` selects which captured material to use and `indices` distinguish call sites within the decision (e.g., `i` for the per-sample RNG, `(i, j)` for the per-`(sample, candidate)` rollout RNG, no extra index for the per-call tiebreak and pass RNGs). The derivation is a pure function: same inputs ⇒ same RNG, no matter how many times it's called or in what order. Workers never share an RNG, and no RNG's state depends on what any other RNG produced.

3. **Index-keyed aggregation.** Per-sample results are written into a fixed-size `[N][]sampleResult` slot keyed by sample index, not appended in arrival order. The aggregation step ([§2 Step 4](#step-4-aggregate-and-choose)) sums in index order, so the total is independent of completion order.

The tiebreak draw is just one more `deriveRNG(tiebreakSeed, fingerprint)` call. Because `tiebreakSeed` and `sampleSeed` are independent captured slots, sampling cannot perturb tiebreak outcomes — there is no shared mutable state between them to perturb.

### Why capture seed material at construction instead of holding the caller's RNG?

A `*math/rand/v2.Rand` is mutable: every call to its methods advances its internal state. If `PIMC` simply held the caller's RNG and consumed from it on every `ChoosePass` or `ChoosePlay`, the determinism contract would have to be stated in terms of "calls so far in this `PIMC` instance", which is brittle:

- **Pass/play coupling.** Calling `ChoosePass` would advance the caller's RNG, which would then change the per-sample seeds used in subsequent `ChoosePlay` calls. Two callers that both invoke `ChoosePlay` at the same game state with the same construction seed would get different answers depending on whether either had previously called `ChoosePass`.
- **Worker-count coupling via RNG consumption.** If per-sample RNGs were derived by calling `callerRNG.Uint64()` once per sample at dispatch time, the order in which workers pulled samples could affect RNG consumption order in subtle ways even with index-keyed aggregation.
- **Tiebreak fragility.** The tiebreak draw, which happens after sampling, would need the caller's RNG to be in a specific state. That requires a maintenance discipline: any future code added between sample dispatch and tiebreak must not call the RNG, or determinism silently breaks. This kind of unwritten invariant is brittle.

Capturing seed material once at `NewPIMC` time and deriving per-call RNGs functionally from `(slot, fingerprint, indices)` eliminates all three problems by construction:

- The caller's RNG is touched only at construction, never again. The caller can do whatever they like with it afterward; `PIMC`'s behavior is unaffected.
- Each captured slot is immutable. Per-call RNGs are pure functions of `(slot, fingerprint, indices)`, so no RNG depends on what any other RNG produced or in what order.
- Per-sample and per-pair RNGs are derived from `(sampleSeed, fingerprint, i)` and `(sampleSeed, fingerprint, i, j)` respectively, so they are stable regardless of worker dispatch order.

`math/rand/v2` does not expose a way to extract the seed of an arbitrary `*rand.Rand`, so `PIMC` reads enough fresh `Uint64()` values from the caller's RNG at construction to fill each captured slot independently. This is the minimum-clean construction available without leaking implementation choices into the public API.

### What the contract does not promise

Determinism is bounded by what `PIMC` controls. It does **not** promise:

- **Cross-Go-version determinism.** `math/rand/v2` is not guaranteed stable across Go releases. A different Go version may produce different RNG outputs from the same seed.
- **Cross-cardcore-version determinism.** Engine internals (e.g., the order in which `analyze` reports voids, or the order legal moves are enumerated) may change between releases.
- **Determinism with nondeterministic rollout policies.** If the caller injects a `rolloutFactory` that produces a policy with its own non-RNG nondeterminism (e.g., reads the system clock, consults a network service), `PIMC`'s determinism is forfeit. The shipped `Heuristic` and `Random` policies are deterministic given their RNG.

Test fixtures that depend on cross-version stability should pin both the Go toolchain and the cardcore version.

## 8. Design rationale

### Why don't rollouts use PIMC itself?

A natural question on first encountering PIMC: if PIMC plays well, why don't the *rollouts* also use `PIMC`? Wouldn't that make simulated opponents stronger, and the overall result sharper?

Two reasons it's a bad idea:

- **Cost explodes.** A single `ChoosePlay` call runs `N` rollouts. If each of those rollouts used `PIMC`, every simulated decision inside would itself run `N` nested rollouts, and so on recursively. The cost multiplies at each trick. Even modest `N` (say 100) produces astronomical decision counts within a few tricks.
- **Rollouts need a fixed opponent model.** PIMC's correctness rests on averaging over many deals under a *specified* assumption about how opponents play. If the rollout policy is itself `PIMC`, the opponent model becomes "opponents are running `PIMC` with this rollout policy, which is `PIMC` with this rollout policy, which is…" — a recursive definition with no ground truth. A fixed, simpler policy (we use `Heuristic`) gives the outer `PIMC` a well-defined opponent model to average over.

The tradeoff: `PIMC`'s strength is bounded by how realistically `Heuristic` models real opponents. If `Heuristic` plays nothing like real players, `PIMC`'s sampled worlds are internally consistent but empirically wrong. This is a known limitation of PIMC as a family; the research fix is not recursion but belief modeling ([§9](#9-future-work)).

### Why one rollout per (sample, candidate) pair?

The sample budget rule — exactly one rollout per `(sample i, candidate c)` pair — may look arbitrary. Why not run, say, ten rollouts per pair and average them to reduce noise further?

Because on a given sampled deal, the rollout is deterministic given the RNG of the rollout policy. The noise in `PIMC` comes from the *deal*, not from randomness inside a single rollout. Ten rollouts on the same deal with the same rollout policy either produce identical results (fully deterministic policy) or minor variations around a single number (slightly stochastic policy). In both cases the noise they average over is small compared to the variance from deal to deal.

The efficient way to reduce variance is to draw *more deals*, not to rerun the same deal. Budget that would go into extra rollouts per deal should go into more samples instead.

The single-rollout rule also makes [common random numbers](#common-random-numbers) ([§1](#1-overview)) exact: every candidate within sample `i` is evaluated on the same deal *and* against a rollout policy whose RNG depends only on `(i, j)`, never on prior candidates' draws. The comparison is genuinely paired.

### Why compute leaf score from `RoundPts` instead of `Scores`?

The Hearts engine has two point-tracking fields: `g.RoundPts[seat]`, updated incrementally as each trick resolves, and `g.Scores[seat]`, updated only when the round ends (at which point the moon flip is applied). `PIMC` computes its leaf score from `RoundPts` and applies the moon flip locally, ignoring `Scores`.

Two reasons:

- **Decouples `PIMC` from engine timing.** `scoreRound` runs only when the engine considers the round complete. Inside a rollout, that moment is a simulated event, and relying on it would tie `PIMC`'s leaf score to the engine's internal state-machine transitions. Reading `RoundPts` plus applying the moon flip in `PIMC`'s own code keeps the leaf-score logic self-contained.
- **The local formula is trivial.** Re-implementing it locally is cheaper than reasoning about engine timing at every leaf.

The local formula assumes the standard Hearts scoring rules described in [`rules.md`](rules.md). Variants such as Omnibus Hearts (J♦ scores -10), Spot Hearts (each heart scores its rank), or no-moon variants would require updating the formula or replacing it with an engine call. If `PIMC` is ever extended to support a Hearts variant whose scoring differs from the standard rules, the leaf-score formula in `pimc.go` is the single place that needs to change.

### Why no `Default()` convenience constructor?

A `NewPIMCDefault()` that picks reasonable values for `samples` and `workers` would save callers a few lines. It is still not provided.

- **There is no right default.** `samples` trades off latency against strategy quality; the right number depends on whether this is a real-time UI, a batch simulation, or a tournament. `workers` depends on the host hardware. Any baked-in default makes one use case convenient at the expense of every other.
- **The caller needs both RNG and `rolloutFactory` anyway.** `NewPIMC` already requires an RNG and a rollout factory, neither of which a `Default()` can reasonably guess. At that point, adding two more explicit arguments is not a burden worth a convenience API for.
- **Suckless.** Fewer exports is better. A caller who wants a default can write it themselves in one line: `NewPIMC(rand.New(rand.NewPCG(1, 2)), 100, heuristicFactory, runtime.NumCPU())`.

### Why a fresh rollout policy per `(sample, candidate)` pair?

`rolloutFactory` is called once per `(sample i, candidate j)` pair, producing a new `hearts.Player` instance each time. It would be cheaper to construct one policy at `NewPIMC` time and reuse it; cheaper still to construct one per sample and reuse it across that sample's candidates.

Three reasons not to:

- **Per-`(sample, candidate)` RNG isolation.** Each pair has its own `RNG_ij` derived from `(sampleSeed, fingerprint, i, j)` ([§7](#7-reproducibility--testing-infrastructure)). The rollout policy needs its own copy of that RNG; reconfiguring a shared policy per pair is awkward and error-prone.
- **CRN demands per-pair freshness, not per-sample.** `Heuristic` consumes RNG bytes during tiebreaks. If two candidates within sample `i` shared a policy (and therefore a single RNG), candidate `j+1`'s tiebreaks would draw from RNG state that depends on how many tiebreaks candidate `j` happened to perform — which depends on the cards in the sampled deal interacting with each candidate. Two candidates would no longer face an apples-to-apples comparison; common random numbers ([§1](#1-overview)) would be broken end-to-end. Per-pair freshness is what makes the comparison actually paired.
- **No cross-pair state accumulation.** A fresh policy per pair cannot accidentally carry state from one pair into the next. If the policy implementation kept any internal state that mutated during play (a cache of past decisions, an opponent model that updated as the game went on, anything similar), reusing one policy would let those mutations leak between pairs. The factory pattern makes per-pair isolation a structural property of the API rather than a runtime discipline the implementation has to remember.

The cost of constructing a policy is small (`Heuristic` is a few fields); the benefit is that pair `(i, j)` is genuinely independent of pair `(i, j')` regardless of what the policy implementation does internally.

### Why use the rollout policy for `seat`'s own future moves?

Once `PIMC` has committed to candidate `c` as `seat`'s current move, the rollout simulates *all* subsequent decisions — including `seat`'s own future moves — using the same rollout policy as for opponents.

The obvious-sounding alternative — use `PIMC` (or some stronger policy) for `seat`'s future decisions, and `Heuristic` for opponents — runs into the same recursion problem rollouts already had to avoid: nested `PIMC` calls inside a rollout multiply cost at every trick and produce a circular opponent model. Any policy strictly stronger than `Heuristic` for `seat`'s future moves biases the leaf score upward (`PIMC` thinks it will play well later), inflating mean scores for candidates that depend on good future play. Keeping `seat` on the same policy as opponents removes that asymmetry.

This is a modeling choice, not a claim that `Heuristic` is actually how `seat` will play in the real game: the outer `PIMC` will still run at `seat`'s next turn and reconsider. The rollout only needs a reasonable *approximation* of future play, and symmetry is the cleanest one.

### Why exact uniform DP sampling instead of rejection?

The naive way to sample a constraint-satisfying deal is **rejection sampling**: shuffle the unseen cards, deal them out by hand-count quotas, check whether all constraints (voids, hard pass cards) hold, and retry on failure. It is simple, easy to reason about, and correct in the sense that every accepted deal is uniformly drawn from the feasible set.

It is also unusable in practice for `PIMC`, because acceptance rates collapse under realistic Hearts constraints.

Two regimes illustrate the problem. Early in the round, before any voids are revealed, the dominant constraint is the **hard pass cards**: each unplayed pass card from `g.PassHistory[seat]` must land with its specific recipient. With three pass cards still unplayed, the probability that an unconstrained random shuffle happens to place all three with the right opponents is roughly **a few percent**, so most attempts get rejected. By the middle of the round, opponents have typically revealed one or two **voids** by failing to follow suit; combined with declining hand sizes, acceptance drops by another order of magnitude. By the late round — tight voids, small hands — acceptance can fall well **below one percent**, making rejection effectively a "shuffle until you get lucky" loop.

Even the best of those regimes is bad. Each `ChoosePlay` call needs `N` deals; multiplying that by tens or hundreds of rejected attempts each, before any rollouts begin, dwarfs the sampling budget.

The chosen approach side-steps rejection entirely by sampling **directly** from the feasible set in one pass. No retries. The construction is a dynamic program over (suit, remaining-capacity-per-opponent) states that counts how many feasible deals exist below each branch; given those counts, a single random walk down the DP picks one feasible deal uniformly at random. The DP is computed **once per `ChoosePlay`** because the constraints are identical across all `N` samples — only the per-sample random walks differ. Per-sample work is therefore a small constant.

The result: every sample produces a valid deal in deterministic time, regardless of how tight the constraints are. The determinism contract ([§7](#7-reproducibility--testing-infrastructure)) is preserved because each walk consumes only `RNG_i`. A worked example is in [Appendix A](#appendix-a-exact-uniform-dp-sampler--worked-example); the full algorithm is in `pimc_sample.go`.

## 9. Future work

### Pass-phase PIMC

`PIMC` currently delegates `ChoosePass` to an internal `Heuristic`. A PIMC-based pass strategy would extend the same sample-and-rollout approach to the pre-play phase, but with a complication that play-phase PIMC does not have: at the pass moment, *two* sources of hidden information bear on the post-pass world. The sampler must determinize both.

- **Pre-pass opponent hands.** No cards have been played yet, so all 39 non-`seat` cards are unseen and the only constraints are the standard deal invariants (13 cards per opponent). The hidden-information regime is much wider than at any later play decision.
- **Opponent pass choices.** Each opponent passes three cards according to *their* pass policy, not three random cards. The sampler must apply a model of opponent pass behavior — likely `Heuristic`'s own `ChoosePass` — to each determinized pre-pass hand and route the results per the round's pass direction. Skipping this step assumes opponents pass uniformly at random, which is empirically wrong (opponents tend to pass high spades, the Q♠, etc.).

For each candidate set of three cards `seat` could pass, the per-call loop is then: sample many pre-pass opponent hands, apply the opponent pass model, route passes per the pass direction, forward-simulate the full 13-trick round under the rollout policy, average leaf scores.

This is more expensive than play-phase PIMC for two compounding reasons. First, the candidate space is much larger: passing involves choosing three cards from the hand, so there are `C(13, 3) = 286` candidate pass sets versus typically fewer than 13 candidate plays. Second, each rollout must simulate an entire round (13 tricks) rather than the rest of the current trick onward, multiplying per-rollout cost. Combined with `N` samples per candidate, the total work scales roughly as `286 · N · 13` rollout-tricks per `ChoosePass` call — orders of magnitude more than `ChoosePlay`.

A practical pass-phase PIMC would likely need candidate pruning (e.g., apply heuristic filters to drop obviously-bad pass sets before sampling), reduced sample counts at the pass phase, or a different leaf-score formulation (e.g., evaluating just the first few tricks rather than the full round) to be feasible. None of that is implemented here.

### Double-dummy solver as an evaluation tool

A future evaluation tool worth considering: a **[double-dummy solver](#double-dummy-solver)** — a player that, given perfect information about all four hands, computes the truly optimal play via exhaustive game-tree search (the term comes from the Bridge world, where this technique is foundational). Such a player would not be a candidate for production play (it cheats), but it would serve as a per-decision *ground-truth optimum* against which any imperfect-information player can be measured. The natural metric is "how often does player X agree with the double-dummy choice, and when it disagrees, by how many points?"

Two important caveats:

- The double-dummy optimum assumes opponents *also* play with perfect information. Real opponents do not, so double-dummy "optimal" can occasionally diverge from the truly best move against real opponents. This is sometimes called **double-dummy bias**.
- Computational cost grows rapidly with the number of unplayed tricks. A double-dummy solver is most practical for late-round positions; full-round solving may require alpha-beta pruning, transposition tables, or limited search depth.

More capable Hearts AIs may be added in the future. Their design is out of scope for this document.

### Relative leaf scoring

The current `leafScore` returns only the seat's own round points (after the moon flip). This means "took zero points because I won nothing" and "took zero points because I shot the moon and gave every opponent 26" are valued identically. In practice the moon case is far better — it inflicts maximum damage on all three opponents — but PIMC has no way to prefer it.

A relative scoring function (e.g., own points minus mean opponent points) would distinguish these cases. It would push PIMC toward pursuing the moon when the opportunity arises, and conversely toward breaking an opponent's moon attempt even at some cost to its own round score. Both adjustments reflect how experienced human players think about the game.

## 10. Glossary

### Common random numbers

A variance-reduction technique: when comparing alternatives via simulation, use the *same* random inputs for each alternative so paired noise cancels. Equivalent to paired statistical comparison. Further reading: [Wikipedia: Common random numbers](https://en.wikipedia.org/wiki/Common_random_numbers).

### Counterfactual regret minimization (CFR)

A family of algorithms for solving imperfect-information games by iteratively minimizing *counterfactual regret* — the regret a player would have for not having played differently at each information set, weighted by the probability of reaching that information set. Unlike PIMC, CFR operates directly on information sets and produces strategies that account for hidden information rather than assuming it away. CFR and its variants (CFR+, MCCFR, Deep CFR) are the foundation of recent superhuman poker AIs. Practical for Hearts is an open question — the game's information-set count is large but bounded. Further reading: [Zinkevich, Johanson, Bowling, & Piccione (2007), "Regret Minimization in Games with Incomplete Information"](https://martin.zinkevich.org/publications/regretpoker.pdf).

### Determinized deal

PIMC's term for one *imagined* assignment of the currently-unseen cards to the three opponents at the moment of a decision. (The unseen cards are whatever has not been played and is not in `seat`'s own hand — typically far fewer than the 52 cards of the round-start deal.) An assignment is **determinized** in the sense that uncertainty about who holds what is resolved by guessing. PIMC samples many such deals to approximate the true (unknown) one. Further reading: [Wikipedia: Determinization (game theory context)](https://en.wikipedia.org/wiki/Game_tree#Imperfect_information_games).

### Double-dummy solver

A perfect-information game-tree solver: given all four hands, computes the optimal play. Originated in Bridge analysis. Used here as a possible future evaluation tool, not as a player. Further reading: [Wikipedia: Double dummy](https://en.wikipedia.org/wiki/Glossary_of_contract_bridge_terms#double_dummy).

### Game-state fingerprint

A deterministic 64-bit digest of the public-and-own-hand parts of `(g, seat)` — the inputs `PIMC` is allowed to read per [§3](#3-information-leakage-discipline). Computed once at the top of each `ChoosePlay` and `ChoosePass` call, then mixed with one of `PIMC`'s captured seed slots and any per-call indices to derive a `*math/rand/v2.Rand` for that call. Same `(g, seat)` ⇒ same fingerprint ⇒ same derived RNGs ⇒ same chosen card. The fingerprint is what makes the per-call RNG derivation in [§7](#7-reproducibility--testing-infrastructure) pure rather than stateful.

### Information-set MCTS

A variant of Monte Carlo Tree Search that builds the search tree over *information sets* — the equivalence classes of game states a player cannot distinguish from their own perspective — rather than over fully-observed game states. Unlike PIMC, IS-MCTS does not pre-determinize hidden information; it samples consistent worlds at each node visit, accumulating statistics in a way that respects the actual information structure of the game. Avoids strategy fusion in the limit, at the cost of much more complex bookkeeping than PIMC. Further reading: [Cowling, Powley, & Whitehouse (2012), "Information Set Monte Carlo Tree Search"](https://orca.cardiff.ac.uk/id/eprint/47925/1/Information%20Set%20MCTS%20%28journal%29.pdf).

### Leaf score

The numeric outcome of a rollout for a single candidate card, evaluated at the *leaf* of the simulated game tree (i.e., end of round). For PIMC, the leaf score is `seat`'s round points after applying the moon flip locally. Lower is better.

### Moon flip

The Hearts scoring rule that inverts a shooter's 26 points: instead of the shooter taking 26 points, the other three players each take 26. Defined in the [Hearts rules](rules.md). PIMC re-implements this locally in its leaf score rather than reading the engine's post-flip `Scores` field.

### Online planning

Planning one decision at a time, with full re-planning at the next decision; the algorithm makes no commitments beyond the current move. PIMC is online: each `ChoosePlay` call plans from scratch. Contrasts with *offline* learning, where a strategy is computed in advance. Further reading: [Wikipedia: Online algorithm](https://en.wikipedia.org/wiki/Online_algorithm).

### PIMC (Perfect Information Monte Carlo)

The algorithm this document describes: convert an imperfect-information problem into a sample of perfect-information problems, solve each, and average. Also called *Monte Carlo perfect-information sampling* in some literature. Further reading: [Frank & Basin (1998)](https://www.cs.cornell.edu/courses/cs672/2007sp/papers/frank-basin-search-incomplete.pdf) — the seminal critique that named strategy fusion as PIMC's core limitation.

### Policy

In game-playing AI, a *policy* is any rule that maps a game state to an action. Every implementation of `hearts.Player` is a policy. The term emphasizes the substitutable, function-shaped nature of decision rules. Further reading: [Wikipedia: Policy (reinforcement learning)](https://en.wikipedia.org/wiki/Reinforcement_learning#Definition).

### Position

The complete state of a game at a particular moment: hands, completed tricks, the in-progress trick, scores, whose turn. Standard term in game-playing AI; used here interchangeably with "game state".

### RNG (random number generator)

In Go: `*math/rand/v2.Rand`. `PIMC` reads seed material from a caller-supplied RNG once at construction and stores it in three immutable slots: `sampleSeed`, `tiebreakSeed`, and `passSeed`. Per-call RNGs are derived freshly from `(slot, fingerprint, indices)` whenever needed — never reused across calls, never sharing mutable state. See [§7](#7-reproducibility--testing-infrastructure).

### Rollout

A simulated forward play of the game from the current state to a terminal state (round end, in `PIMC`'s case). The simulation uses a fixed *rollout policy* to pick each move. Synonyms in the literature: *playout*, *simulation*. Further reading: [Wikipedia: Monte Carlo tree search § Playout](https://en.wikipedia.org/wiki/Monte_Carlo_tree_search#Principle_of_operation).

### Rollout policy

The policy used to choose each move during a rollout. Conceptually a model of how players (including opponents) will play. PIMC's default rollout policy is `Heuristic`. Also called *playout policy* or *default policy* in the MCTS literature.

### Sample

One iteration of `PIMC`'s outer loop: one determinized deal, one rollout per candidate, one set of leaf scores. `PIMC` runs `N` samples per `ChoosePlay` call.

### Strategy fusion

PIMC's core limitation: by solving each sampled deal *as if* it were the real one, PIMC implicitly assumes the player can adapt their strategy to hidden information. Real players cannot. This causes PIMC to overweight moves that exploit deal-specific information. Further reading: [Frank & Basin (1998)](https://www.cs.cornell.edu/courses/cs672/2007sp/papers/frank-basin-search-incomplete.pdf).

## Appendix A: Exact uniform DP sampler — worked example

This appendix walks through the DP sampler ([§8](#why-exact-uniform-dp-sampling-instead-of-rejection)) on a tiny position. The point is to make the algorithm visible end-to-end; the production code in `pimc_sample.go` is the same shape, just with full Hearts-sized inputs.

### The teaching position

After applying hard pass-card assignments and removing all played cards, suppose:

- **Unseen cards (6 total):** one spade (`A♠`), two hearts (`2♥, 3♥`), one diamond (`A♦`), two clubs (`2♣, 3♣`).
- **Hand sizes to fill (3 opponents):** West = 2, North = 2, East = 2.
- **Voids revealed:** West cannot hold spades; East cannot hold diamonds. North has no voids.

### What "uniformly random feasible deal" means

Across all possible ways to assign these 6 cards to W/N/E satisfying the void constraints, every individual deal must be equally likely. There are exactly **42** such deals (verified below). A correct sampler picks each of those 42 with probability 1/42.

The DP does this without ever enumerating the 42 deals.

### Suit-by-seat state

The DP's state is **how many cards of the current suit go to each opponent**, given the constraints. Because the only constraint that crosses suits is the per-opponent capacity (each must end up with exactly 2 cards), we process suits one at a time and track remaining capacity per opponent.

State: `(suitIndex, capW, capN, capE)`. Initial state: `(0, 2, 2, 2)`.

For each suit we enumerate every legal split `(w, n, e)` where `w + n + e` equals the number of cards of this suit, the per-opponent counts respect remaining capacity (`w ≤ capW`, `n ≤ capN`, `e ≤ capE`), and any opponent void in this suit gets zero (e.g., West void in spades forces `w = 0` for the spade split). Each split:

- Multiplies in the multinomial coefficient `(cards-of-this-suit)! / (w! · n! · e!)`. This counts the number of ways to actually deal the specific cards into hands consistent with the split `(w, n, e)`. For example, if the suit has 2 cards and the split is `(1, 1, 0)`, there are 2 ways: card A to W and card B to N, or vice versa. If the split is `(2, 0, 0)` there is only 1 way. For a suit with 1 card the multinomial is always 1.
- Recurses with reduced capacities to the next suit.

The base case is at the **last** suit: only one split remains valid (cards must exactly fill the remaining capacity), and we either return its multinomial or 0 if infeasible.

Process suits in a fixed order, e.g., `[♠, ♥, ♦, ♣]`.

### Walking the DP for the teaching position

Below, "split" means `(w, n, e)`; "weight" is the multinomial coefficient.

**Suit ♠ (1 card, W is void).** Legal splits: `(0,1,0)` and `(0,0,1)`. Each has weight 1. After this:
- `(0,1,0)` → reduced state `(♥, 2, 1, 2)`, weight 1.
- `(0,0,1)` → reduced state `(♥, 2, 2, 1)`, weight 1.

For each branch we recurse into hearts, then diamonds, then clubs, multiplying weights and summing leaf counts.

The full recursion produces, after summing, **42 card-level deals** distributed across 15 distinct suit-count configurations. A few examples (showing one path):

> ♠: `(0,1,0)` weight 1 → state `(♥, 2, 1, 2)`
> ♥: `(1,1,0)` weight 2 → state `(♦, 1, 0, 2)`
> ♦: `(1,0,0)` weight 1 (E is void in diamonds) → state `(♣, 0, 0, 2)`
> ♣: `(0,0,2)` weight 1 → leaf
>
> Path weight = 1·2·1·1 = 2.

The DP value at the root state `(♠, 2, 2, 2)` ends up as **42**, which equals `Σ` of all leaf path weights. (You can verify: 7 paths weight 4, 6 paths weight 2, 2 paths weight 1, summing to 28 + 12 + 2 = 42.)

### Per-sample walk

Once the DP table is filled — done **once per `ChoosePlay`** because constraints are identical across all `N` samples — generating a single sample is cheap:

1. Start at root state `(♠, 2, 2, 2)` with total weight 42.
2. Enumerate the legal splits at the current suit. Each split has a sub-weight equal to `(its multinomial) × (DP value of resulting reduced state)`.
3. Pick one split with probability proportional to its sub-weight, using `RNG_i`.
4. Once a split is chosen, randomly assign the actual cards of that suit to opponents consistent with the split (a small uniform sub-draw — e.g., for split `(1,1,0)` of two hearts, pick which of `2♥`/`3♥` goes to W, the other to N).
5. Move to the chosen reduced state and repeat for the next suit.
6. After all four suits, the resulting hand assignment is one uniformly random feasible deal.

The walk consumes only `RNG_i`, so two samples with the same index produce identical deals — the determinism property required by [§7](#7-reproducibility--testing-infrastructure).

### Why this is uniform

At every step, the probability of picking a particular split is proportional to the number of complete deals reachable below that branch. Multiplied along the path, the total probability of any specific complete deal is exactly `1 / 42` (in this example), uniform over all feasible deals. No deal is privileged or excluded; no rejection is needed.

### Why "once per `ChoosePlay`" matters

The DP table depends only on `(suit cards, hand sizes, voids)` — all derived from public state. It does **not** depend on the per-sample RNG. Computing the table once and walking it `N` times turns the per-sample cost from "shuffle until valid" (rejection) into a small constant walk. For typical Hearts [positions](#position) the DP has on the order of hundreds of states; the walks are essentially free relative to rollout cost.

### Reference implementation

See `pimc_sample.go` in `games/hearts/ai/`. The functions of interest are `sampleDealDP.build` (DP table builder) and `sampleDealDP.sample` (per-deal walk), which together implement the steps above for the full 4-suit × 3-opponent Hearts case.
