package ai

import (
	"math/rand/v2"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// BenchmarkRoundHeuristic measures the cost of constructing a Hearts game
// and playing one full round (deal, pass, play 13 tricks, score) with four
// Heuristic players. The measurement includes hearts.New since each
// iteration needs a fresh game.
func BenchmarkRoundHeuristic(b *testing.B) {
	players := newHeuristics()
	for b.Loop() {
		g := hearts.New()
		playRoundBench(g, players)
	}
}

// BenchmarkFullGameHeuristic measures the cost of constructing a Hearts
// game and playing it to completion (rounds played until any player reaches
// MaxScore) with four Heuristic players. The measurement includes
// hearts.New since each iteration needs a fresh game.
func BenchmarkFullGameHeuristic(b *testing.B) {
	const maxRounds = 20
	players := newHeuristics()
	for b.Loop() {
		g := hearts.New()
		for range maxRounds {
			playRoundBench(g, players)
			if g.Phase == hearts.PhaseEnd {
				break
			}
		}
		if g.Phase != hearts.PhaseEnd {
			panic("game did not end within maxRounds")
		}
	}
}

// BenchmarkRoundPIMC measures the cost of playing one full round with
// four PIMC players (N=30, W=4, Heuristic rollout policy).
func BenchmarkRoundPIMC(b *testing.B) {
	players := newPIMCs(30, 4)
	for b.Loop() {
		g := hearts.New()
		playRoundBench(g, players)
	}
}

// BenchmarkFullGamePIMC measures the cost of playing a complete game
// with four PIMC players (N=30, W=4, Heuristic rollout policy).
func BenchmarkFullGamePIMC(b *testing.B) {
	const maxRounds = 20
	players := newPIMCs(30, 4)
	for b.Loop() {
		g := hearts.New()
		for range maxRounds {
			playRoundBench(g, players)
			if g.Phase == hearts.PhaseEnd {
				break
			}
		}
		if g.Phase != hearts.PhaseEnd {
			panic("game did not end within maxRounds")
		}
	}
}

// newHeuristics returns four Heuristic players, one per seat, each
// with its own deterministically seeded RNG.
func newHeuristics() [hearts.NumPlayers]hearts.Player {
	var players [hearts.NumPlayers]hearts.Player
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		rng := rand.New(rand.NewPCG(1, 1+uint64(seat)))
		players[seat] = NewHeuristic(rng)
	}
	return players
}

// newPIMCs returns four PIMC players, one per seat, each with its own
// deterministically seeded RNG, the given sample count and worker count,
// and a Heuristic rollout policy.
func newPIMCs(samples, workers int) [hearts.NumPlayers]hearts.Player {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }
	var players [hearts.NumPlayers]hearts.Player
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		rng := rand.New(rand.NewPCG(1, 1+uint64(seat)))
		players[seat] = NewPIMC(rng, samples, factory, workers)
	}
	return players
}

// playRoundBench plays one complete round (deal, pass, play, score)
// using a distinct Player per seat. Panics on any engine error.
func playRoundBench(g *hearts.Game, players [hearts.NumPlayers]hearts.Player) {
	if err := g.Deal(); err != nil {
		panic("Deal: " + err.Error())
	}

	if g.Phase == hearts.PhasePass {
		for i := hearts.Seat(0); i < hearts.NumPlayers; i++ {
			cards := players[i].ChoosePass(g.Clone(), i)
			if err := g.SetPass(i, cards); err != nil {
				panic("SetPass: " + err.Error())
			}
		}
	}

	for g.Phase == hearts.PhasePlay {
		seat := g.Turn
		card := players[seat].ChoosePlay(g.Clone(), seat)
		if err := g.PlayCard(seat, card); err != nil {
			panic("PlayCard: " + err.Error())
		}
	}

	if err := g.EndRound(); err != nil {
		panic("EndRound: " + err.Error())
	}
}
