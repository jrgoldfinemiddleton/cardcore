package ai

import (
	"math/rand/v2"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// aggregate selects the best candidate across all sampled rollouts.
//
// Input shape: results[i][j] is the leaf score for sample i, candidate j.
// Every inner slice must have the same length and present candidates in
// the same order (the caller's enumeration order); results[0][j].candidate
// is taken as the canonical identity for candidate j.
//
// Hearts is a point-minimization game, so the candidate with the lowest
// total leaf score wins. (Lowest total is equivalent to lowest mean for
// ranking, since every candidate shares the same denominator N. Summing
// avoids floating-point arithmetic.)
//
// Ties at the minimum are broken uniformly at random using tiebreakRNG,
// which the caller derives from the captured tiebreakSeed and the
// game-state fingerprint (see deriveRNG).
//
// The function is order-independent across the outer (sample) dimension:
// permuting results yields the same return value. This is what makes
// PIMC deterministic despite parallel sample collection — goroutine
// scheduling cannot affect the outcome.
//
// Panics if results is empty, if any inner slice is empty, or if inner
// slice lengths disagree (caller programming error).
func aggregate(results [][]sampleResult, tiebreakRNG *rand.Rand) cardcore.Card {
	if len(results) == 0 {
		panic("ai: aggregate called with empty results")
	}
	numCandidates := len(results[0])
	if numCandidates == 0 {
		panic("ai: aggregate called with empty candidate slice")
	}
	for _, row := range results {
		if len(row) != numCandidates {
			panic("ai: aggregate called with ragged results (inner slices disagree on length)")
		}
	}

	totals := make([]int, numCandidates)
	for _, row := range results {
		for j, r := range row {
			totals[j] += r.leafScore
		}
	}

	best := totals[0]
	for _, t := range totals[1:] {
		if t < best {
			best = t
		}
	}

	// tiebreak draw is deterministic given tiebreakRNG state.
	tied := make([]int, 0, numCandidates)
	for j, t := range totals {
		if t == best {
			tied = append(tied, j)
		}
	}

	if len(tied) == 1 {
		return results[0][tied[0]].candidate
	}
	return results[0][tied[tiebreakRNG.IntN(len(tied))]].candidate
}
