package ai

import (
	"math/rand/v2"
	"testing"
)

// BenchmarkHeuristicPlay measures Heuristic.ChoosePlay per-call cost
// across the six benchmark fixtures. The opponent_moon_threat fixture
// exercises moonBlock branches in followScore/voidScore/heartLeadScore.
func BenchmarkHeuristicPlay(b *testing.B) {
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			h := NewHeuristic(rand.New(rand.NewPCG(3, 4)))
			b.ReportAllocs()
			for b.Loop() {
				_ = h.ChoosePlay(g, seat)
			}
		})
	}
}

// BenchmarkHeuristicPlayWithClone measures Heuristic.ChoosePlay per-call
// cost when each iteration clones the game first. The diff against
// BenchmarkHeuristicPlay quantifies Game.Clone() cost in the heuristic
// hot path.
func BenchmarkHeuristicPlayWithClone(b *testing.B) {
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			h := NewHeuristic(rand.New(rand.NewPCG(3, 4)))
			b.ReportAllocs()
			for b.Loop() {
				_ = h.ChoosePlay(g.Clone(), seat)
			}
		})
	}
}
