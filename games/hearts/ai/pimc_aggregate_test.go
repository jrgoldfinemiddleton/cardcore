package ai

import (
	"math/rand/v2"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// panicSource is a fake rand.Source whose Uint64 panics. Used via
// panicRNG to assert the tiebreak path is not consulted.
type panicSource struct{}

// TestAggregateSingleCandidateSingleSample verifies the trivial case:
// one sample, one candidate returns that candidate without consulting RNG.
func TestAggregateSingleCandidateSingleSample(t *testing.T) {
	cand := twoOfClubs
	results := [][]sampleResult{
		{{candidate: cand, leafScore: 5}},
	}
	got := aggregate(results, panicRNG())
	if got != cand {
		t.Errorf("got %v, want %v", got, cand)
	}
}

// TestAggregateClearWinner verifies the lowest total wins and the RNG
// is not consulted when a single candidate is strictly best.
func TestAggregateClearWinner(t *testing.T) {
	low := twoOfClubs
	mid := c(rThree, sClubs)
	high := c(rFour, sClubs)
	results := [][]sampleResult{
		{
			{candidate: low, leafScore: 1},
			{candidate: mid, leafScore: 5},
			{candidate: high, leafScore: 9},
		},
		{
			{candidate: low, leafScore: 2},
			{candidate: mid, leafScore: 5},
			{candidate: high, leafScore: 9},
		},
		{
			{candidate: low, leafScore: 3},
			{candidate: mid, leafScore: 5},
			{candidate: high, leafScore: 9},
		},
	}
	got := aggregate(results, panicRNG())
	if got != low {
		t.Errorf("got %v, want %v", got, low)
	}
}

// TestAggregateLowestMeanWins verifies the direction is min, not max:
// candidate A (sum 10) beats candidate B (sum 100).
func TestAggregateLowestMeanWins(t *testing.T) {
	a := twoOfClubs
	b := c(rThree, sClubs)
	results := [][]sampleResult{
		{{candidate: a, leafScore: 5}, {candidate: b, leafScore: 50}},
		{{candidate: a, leafScore: 5}, {candidate: b, leafScore: 50}},
	}
	got := aggregate(results, panicRNG())
	if got != a {
		t.Errorf("got %v (sum 10), want %v (sum 100 should lose)", got, a)
	}
}

// TestAggregateTiebreakUsesRNG verifies the tiebreak RNG can produce
// different choices for different seeds when candidates are tied. Uses
// the tries=N loop pattern: try several seed pairs and require at least
// one to disagree with the baseline (defends against RNG-luck collapse).
func TestAggregateTiebreakUsesRNG(t *testing.T) {
	a := twoOfClubs
	b := c(rThree, sClubs)
	results := [][]sampleResult{
		{{candidate: a, leafScore: 7}, {candidate: b, leafScore: 7}},
	}
	baseline := aggregate(results, rngWithSeed(1, 2))

	const tries = 8
	for i := uint64(0); i < tries; i++ {
		seed0 := 100 + i
		alt := aggregate(results, rngWithSeed(seed0, seed0+1))
		if alt != baseline {
			return
		}
	}
	t.Errorf("tiebreak appears insensitive to RNG: %d seeds all returned %v", tries, baseline)
}

// TestAggregateTiebreakDeterministicGivenSeed verifies that the same RNG
// state produces the same choice across two calls.
func TestAggregateTiebreakDeterministicGivenSeed(t *testing.T) {
	a := twoOfClubs
	b := c(rThree, sClubs)
	results := [][]sampleResult{
		{{candidate: a, leafScore: 7}, {candidate: b, leafScore: 7}},
	}
	got1 := aggregate(results, rngWithSeed(42, 43))
	got2 := aggregate(results, rngWithSeed(42, 43))
	if got1 != got2 {
		t.Errorf("tiebreak not deterministic for fixed seed: got %v then %v", got1, got2)
	}
}

// TestAggregateAllTied verifies the tiebreak handles the degenerate case
// where every candidate ties at the same total.
func TestAggregateAllTied(t *testing.T) {
	cands := []cardcore.Card{twoOfClubs, c(rThree, sClubs), c(rFour, sClubs)}
	results := [][]sampleResult{
		{
			{candidate: cands[0], leafScore: 5},
			{candidate: cands[1], leafScore: 5},
			{candidate: cands[2], leafScore: 5},
		},
	}
	got := aggregate(results, rngWithSeed(1, 2))
	found := false
	for _, candidate := range cands {
		if got == candidate {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("got %v, want one of %v", got, cands)
	}
}

// TestAggregateOrderIndependence verifies permuting the outer (sample)
// dimension yields identical output. This is what makes PIMC
// deterministic despite parallel sample collection — index-keyed
// aggregation makes worker completion order irrelevant.
func TestAggregateOrderIndependence(t *testing.T) {
	a := twoOfClubs
	b := c(rThree, sClubs)
	original := [][]sampleResult{
		{{candidate: a, leafScore: 1}, {candidate: b, leafScore: 4}},
		{{candidate: a, leafScore: 2}, {candidate: b, leafScore: 5}},
		{{candidate: a, leafScore: 3}, {candidate: b, leafScore: 6}},
	}
	want := aggregate(original, panicRNG())

	permutations := [][][]sampleResult{
		{original[2], original[0], original[1]},
		{original[1], original[2], original[0]},
		{original[2], original[1], original[0]},
	}
	for i, perm := range permutations {
		got := aggregate(perm, panicRNG())
		if got != want {
			t.Errorf("permutation %d: got %v, want %v", i, got, want)
		}
	}
}

// TestAggregatePanicsOnDegenerateInput verifies aggregate panics on
// caller programming errors: nil results, empty outer slice (no
// samples), or empty inner slice (a sample with no candidates).
func TestAggregatePanicsOnDegenerateInput(t *testing.T) {
	tests := []struct {
		name    string
		results [][]sampleResult
	}{
		{"nil", nil},
		{"empty outer", [][]sampleResult{}},
		{"empty inner", [][]sampleResult{{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("aggregate did not panic on %s", tt.name)
				}
			}()
			aggregate(tt.results, panicRNG())
		})
	}
}

// TestAggregatePanicsOnRaggedInput verifies inner slices of differing
// lengths trigger a panic.
func TestAggregatePanicsOnRaggedInput(t *testing.T) {
	a := twoOfClubs
	b := c(rThree, sClubs)
	results := [][]sampleResult{
		{{candidate: a, leafScore: 1}, {candidate: b, leafScore: 2}},
		{{candidate: a, leafScore: 1}},
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("aggregate did not panic on ragged input")
		}
	}()
	aggregate(results, panicRNG())
}

// Uint64 panics to flag any unexpected RNG consultation.
func (panicSource) Uint64() uint64 {
	panic("panicRNG: tiebreak path consulted RNG when it should not have")
}

// panicRNG returns a *rand.Rand that panics if the tiebreak path consults
// it. Used by tests that assert RNG is NOT touched on a clear winner.
func panicRNG() *rand.Rand {
	return rand.New(panicSource{})
}

// rngWithSeed returns a *rand.Rand seeded by (s0, s1) for deterministic tests.
func rngWithSeed(s0, s1 uint64) *rand.Rand {
	return rand.New(rand.NewPCG(s0, s1))
}
