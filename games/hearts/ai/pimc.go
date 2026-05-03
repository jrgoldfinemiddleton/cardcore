package ai

import (
	"encoding/binary"
	"hash/fnv"
	"math/rand/v2"
	"sync"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// sampledDeal is one constraint-satisfying assignment of unseen cards
// across the four seats. Index by hearts.Seat. The PIMC player's own
// entry equals its real hand; the other three are sampled.
type sampledDeal [hearts.NumPlayers]cardcore.Hand

// sampleResult holds the leaf score for one (sample, candidate) pair.
// Slices of sampleResult are stored indexed by sample number so that
// aggregation is order-independent across samples: permuting the outer
// slice does not change the selected candidate.
type sampleResult struct {
	candidate cardcore.Card
	leafScore int
}

// PIMC is a Hearts player that chooses plays via Perfect Information
// Monte Carlo sampling. For each decision it samples N constraint-
// satisfying deals of the unseen cards, simulates a rollout per
// candidate move on each deal, and selects the candidate with the
// lowest total leaf score.
//
// PIMC delegates pass-phase decisions to a per-call [Heuristic];
// pass-phase PIMC is out of scope for this implementation.
//
// PIMC never mutates the live game state. All rollouts operate on
// clones, and all RNGs are derived deterministically from seed
// material captured at construction, so identical seeds produce
// identical play regardless of goroutine scheduling.
//
// For algorithm details and design rationale, see
// doc/games/hearts/ai-pimc-design.md.
type PIMC struct {
	sampleSeed   [2]uint64 // base material for per-(sample, candidate) rollout RNGs
	tiebreakSeed [2]uint64 // base material for the aggregation tiebreak RNG
	passSeed     [2]uint64 // base material for ChoosePass delegation RNGs

	samples        int
	rolloutFactory func(*rand.Rand) hearts.Player
	workers        int
}

// NewPIMC constructs a PIMC player.
//
//   - rng: parent RNG; PIMC consumes six Uint64 values from it to
//     populate three immutable [2]uint64 seed slots (sample,
//     tiebreak, pass) used to derive per-call RNGs. The caller
//     controls seeding for reproducible play; note that the parent
//     RNG state is advanced by this call.
//   - samples: number of determinized deals per decision (must be > 0).
//   - rolloutFactory: produces a fresh rollout policy per (sample,
//     candidate) pair, seeded with the per-pair derived RNG. Must
//     not be nil. The factory is called per-pair (not per-sample) so
//     stochastic policies like Heuristic do not share RNG state
//     across candidates and break common random numbers.
//   - workers: number of goroutines used to run samples in parallel
//     (must be > 0).
//
// Panics if rng is nil, samples <= 0, rolloutFactory is nil, or
// workers <= 0.
func NewPIMC(
	rng *rand.Rand,
	samples int,
	rolloutFactory func(*rand.Rand) hearts.Player,
	workers int,
) *PIMC {
	if rng == nil {
		panic("ai: NewPIMC requires non-nil rng")
	}
	if samples <= 0 {
		panic("ai: NewPIMC requires samples > 0")
	}
	if rolloutFactory == nil {
		panic("ai: NewPIMC requires non-nil rolloutFactory")
	}
	if workers <= 0 {
		panic("ai: NewPIMC requires workers > 0")
	}
	return &PIMC{
		sampleSeed:     [2]uint64{rng.Uint64(), rng.Uint64()},
		tiebreakSeed:   [2]uint64{rng.Uint64(), rng.Uint64()},
		passSeed:       [2]uint64{rng.Uint64(), rng.Uint64()},
		samples:        samples,
		rolloutFactory: rolloutFactory,
		workers:        workers,
	}
}

// ChoosePass delegates to a per-call [Heuristic] seeded from a
// fingerprint-derived RNG. Pass-phase PIMC is out of scope for this
// implementation.
func (p *PIMC) ChoosePass(g *hearts.Game, seat hearts.Seat) [hearts.PassCount]cardcore.Card {
	r := deriveRNG(p.passSeed, fingerprint(g, seat))
	return NewHeuristic(r).ChoosePass(g, seat)
}

// ChoosePlay selects a card to play from the hand at seat using PIMC.
// It enumerates legal moves, samples N constraint-satisfying deals,
// runs a rollout per (sample, candidate) pair, and returns the
// candidate with the lowest total leaf score. Ties are broken
// uniformly at random using a tiebreak RNG derived independently
// from the sampling RNG.
//
// Panics if g is not in PhasePlay or if seat has no legal moves
// (caller programming error).
func (p *PIMC) ChoosePlay(g *hearts.Game, seat hearts.Seat) cardcore.Card {
	fp := fingerprint(g, seat)

	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic("ai: PIMC.ChoosePlay called when seat has no legal moves: " + err.Error())
	}
	if len(legal) == 1 {
		return legal[0]
	}

	c := buildConstraints(g, seat)
	dp := newSampleDealDP(&c)

	results := make([][]sampleResult, p.samples)

	work := make(chan int, p.samples)
	for i := range p.samples {
		work <- i
	}
	close(work)

	var wg sync.WaitGroup
	for range p.workers {
		wg.Go(func() {
			for i := range work {
				sampleRNG := deriveRNG(p.sampleSeed, fp, uint64(i))
				deal := dp.sample(g, seat, sampleRNG)

				results[i] = make([]sampleResult, len(legal))
				for j, card := range legal {
					rolloutRNG := deriveRNG(p.sampleSeed, fp, uint64(i), uint64(j))
					policy := p.rolloutFactory(rolloutRNG)
					score := rollout(g, seat, card, deal, policy)
					results[i][j] = sampleResult{candidate: card, leafScore: score}
				}
			}
		})
	}
	wg.Wait()

	tiebreakRNG := deriveRNG(p.tiebreakSeed, fp)
	return aggregate(results, tiebreakRNG)
}

// fingerprint returns a deterministic 64-bit key that uniquely
// identifies a decision point within a single game. It is consumed
// by deriveRNG as the fp argument: passing the same fingerprint
// produces correlated RNG streams (intentional, e.g. across the
// many rollouts at one decision); passing different fingerprints
// produces uncorrelated streams (intentional, across decisions).
//
// Identity contract:
//
//   - Same decision point ⇒ same fingerprint. This is what makes
//     PIMC reproducible: replaying the same game from the same
//     parent seed reaches each decision with the same fp, so
//     deriveRNG hands out the same sub-streams, so the algorithm
//     makes the same choices.
//
//   - Different decision point ⇒ different fingerprint. This is
//     what keeps the RNGs used at different decisions from being
//     accidentally correlated, which would couple AI choices
//     across decisions and bias the algorithm.
//
// The decision point is identified by the tuple (seat, Phase,
// Round, TrickNum, Trick.Count). Within one game's normal
// progression these five scalars uniquely identify every point at
// which a player must act. Phase is included to disambiguate the
// pass decision from the play decision at the start of a round,
// where the other four scalars are equal.
//
// This is a key, not a state digest. Opponent hands, scores, and
// trick history are intentionally excluded — within a single game
// they are determined by the decisions already made, so they add
// no distinguishing information here.
//
// Lifetime: the identity contract holds within a single game. A
// PIMC instance is intended to play one game; reusing one instance
// across multiple games will produce fingerprint collisions at
// matching decisions, breaking decorrelation.
func fingerprint(g *hearts.Game, seat hearts.Seat) uint64 {
	h := fnv.New64a()
	writeUint64(h, uint64(seat))
	writeUint64(h, uint64(g.Phase))
	writeUint64(h, uint64(g.Round))
	writeUint64(h, uint64(g.TrickNum))
	writeUint64(h, uint64(g.Trick.Count))
	return h.Sum64()
}

// deriveRNG returns a fresh *rand.Rand whose stream is a pure
// function of (slot, fp, indices). Same inputs ⇒ identical sequence
// of values; different inputs in any position ⇒ uncorrelated streams.
// This lets PIMC fan out a small amount of captured seed material
// into the many independent RNGs the algorithm needs, with no
// dependence on goroutine ordering or shared mutable state.
//
// Parameters:
//
//   - slot: one of the [2]uint64 seed slots captured by NewPIMC
//     (sampleSeed, tiebreakSeed, or passSeed). The slot partitions
//     PIMC's RNGs into three non-overlapping families so that, e.g.,
//     a rollout RNG can never accidentally match a tiebreak RNG.
//
//   - fp: a decision-point fingerprint (see fingerprint). Different
//     decision points within one game produce different fp values,
//     ensuring the RNGs used at different decisions are uncorrelated.
//     Within one decision point, fp is constant across all
//     sub-streams.
//
//   - indices: a tuple identifying a sub-stream within one decision
//     point. Order and length are significant: (3, 2) and (2, 3) and
//     (3, 2, 0) all produce distinct streams. For rollouts:
//     (sampleIdx, candidateIdx). For per-decision singleton RNGs
//     (tiebreak, pass), pass nothing.
//
// Seed-pair derivation: the inputs are hashed to a single uint64 s0;
// the second PCG seed word is s1 = ^s0 (bitwise NOT). This is a
// deterministic injective mapping from the 64-bit hash to a 128-bit
// initial PCG state. PCG-DXSM in math/rand/v2 treats both seed words
// as the initial 128-bit state, and its internal mixer diffuses any
// non-trivial input — no further pre-mixing of the seed words is
// needed for non-cryptographic game-AI sampling.
func deriveRNG(slot [2]uint64, fp uint64, indices ...uint64) *rand.Rand {
	h := fnv.New64a()
	writeUint64(h, slot[0])
	writeUint64(h, slot[1])
	writeUint64(h, fp)
	for _, idx := range indices {
		writeUint64(h, idx)
	}
	s0 := h.Sum64()
	s1 := ^s0
	return rand.New(rand.NewPCG(s0, s1))
}

// writeUint64 feeds a uint64 to h in little-endian byte order. The
// byte-order choice is part of the determinism contract: changing it
// changes every reproducible run. Errors from hash.Hash64.Write are
// impossible (documented contract) and so are dropped.
func writeUint64(h interface{ Write([]byte) (int, error) }, v uint64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], v)
	_, _ = h.Write(buf[:])
}
