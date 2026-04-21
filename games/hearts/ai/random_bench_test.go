package ai

import (
	"math/rand/v2"
	"testing"
)

// BenchmarkRandomPlay measures Random.ChoosePlay per-call cost across
// the six benchmark fixtures. Cost reflects LegalMoves computation plus
// uniform random selection.
//
// ReportAllocs ON — per-call allocs/op feeds the PIMC sizing
// budget. RNG seeded once outside the loop; state evolution across
// iterations is part of what we measure, not a confound.
func BenchmarkRandomPlay(b *testing.B) {
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			r := NewRandom(rand.New(rand.NewPCG(1, 2)))
			b.ReportAllocs()
			for b.Loop() {
				_ = r.ChoosePlay(g, seat)
			}
		})
	}
}

// BenchmarkRandomPlayWithClone measures Random.ChoosePlay per-call cost
// when each iteration clones the game first. The diff against
// BenchmarkRandomPlay quantifies Game.Clone() cost — informative for
// PIMC sizing where every rollout starts from a clone.
func BenchmarkRandomPlayWithClone(b *testing.B) {
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			r := NewRandom(rand.New(rand.NewPCG(1, 2)))
			b.ReportAllocs()
			for b.Loop() {
				_ = r.ChoosePlay(g.Clone(), seat)
			}
		})
	}
}
