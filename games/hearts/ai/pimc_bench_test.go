package ai

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// BenchmarkPIMCPlay measures the end-to-end cost of PIMC.ChoosePlay
// (N=30, W=4) across the six benchmark fixtures.
func BenchmarkPIMCPlay(b *testing.B) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			p := NewPIMC(rand.New(rand.NewPCG(1, 2)), 30, factory, 4)
			b.ReportAllocs()
			for b.Loop() {
				_ = p.ChoosePlay(g, seat)
			}
		})
	}
}

// BenchmarkPIMCPlayWithClone measures PIMC.ChoosePlay per-call cost
// when each iteration clones the game first. The diff against
// BenchmarkPIMCPlay quantifies Game.Clone() cost in the PIMC hot path.
func BenchmarkPIMCPlayWithClone(b *testing.B) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }
	for _, tc := range benchFixtures() {
		b.Run(tc.name, func(b *testing.B) {
			g, seat := tc.build()
			p := NewPIMC(rand.New(rand.NewPCG(1, 2)), 30, factory, 4)
			b.ReportAllocs()
			for b.Loop() {
				_ = p.ChoosePlay(g.Clone(), seat)
			}
		})
	}
}

// BenchmarkPIMCPlaySamples sweeps sample count N ∈ {10, 30, 100} with
// W=4 on the follow_with_points fixture (the most complex of the six:
// mid-game, hearts broken, points on the table, multiple legal moves).
func BenchmarkPIMCPlaySamples(b *testing.B) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }
	for _, n := range []int{10, 30, 100} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			g, seat := buildFollowWithPoints()
			p := NewPIMC(rand.New(rand.NewPCG(1, 2)), n, factory, 4)
			b.ReportAllocs()
			for b.Loop() {
				_ = p.ChoosePlay(g, seat)
			}
		})
	}
}

// BenchmarkPIMCPlayWorkers sweeps worker count W ∈ {1, 2, 4, 8} with
// N=30 on the follow_with_points fixture. Shows parallelism scaling.
func BenchmarkPIMCPlayWorkers(b *testing.B) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }
	for _, w := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("W=%d", w), func(b *testing.B) {
			g, seat := buildFollowWithPoints()
			p := NewPIMC(rand.New(rand.NewPCG(1, 2)), 30, factory, w)
			b.ReportAllocs()
			for b.Loop() {
				_ = p.ChoosePlay(g, seat)
			}
		})
	}
}
