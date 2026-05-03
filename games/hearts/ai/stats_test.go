//go:build stats

package ai

import (
	"math/rand/v2"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// TestShootTheMoonFrequency runs 100 complete games per AI type and logs
// how often any seat shoots the moon. Run with -v to see the stats.
func TestShootTheMoonFrequency(t *testing.T) {
	const (
		numGames  = 100
		maxRounds = 20
	)

	type aiSetup struct {
		name    string
		players func(seed uint64) [hearts.NumPlayers]hearts.Player
	}

	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }

	setups := []aiSetup{
		{
			name: "Random",
			players: func(seed uint64) [hearts.NumPlayers]hearts.Player {
				var p [hearts.NumPlayers]hearts.Player
				for i := range hearts.NumPlayers {
					s := seed*uint64(hearts.NumPlayers) + uint64(i)
					p[i] = newSeededRandom(s)
				}
				return p
			},
		},
		{
			name: "Heuristic",
			players: func(seed uint64) [hearts.NumPlayers]hearts.Player {
				var p [hearts.NumPlayers]hearts.Player
				for i := range hearts.NumPlayers {
					s := seed*uint64(hearts.NumPlayers) + uint64(i)
					p[i] = newSeededHeuristic(s)
				}
				return p
			},
		},
		{
			name: "PIMC",
			players: func(seed uint64) [hearts.NumPlayers]hearts.Player {
				var p [hearts.NumPlayers]hearts.Player
				for i := range hearts.NumPlayers {
					s := seed*uint64(hearts.NumPlayers) + uint64(i)
					p[i] = NewPIMC(
						rand.New(rand.NewPCG(s, s+1)),
						10,
						factory,
						1,
					)
				}
				return p
			},
		},
	}

	for _, setup := range setups {
		t.Run(setup.name, func(t *testing.T) {
			var totalRounds, moonShoots int

			for game := range numGames {
				seed := uint64(game) + 1
				players := setup.players(seed)
				g := hearts.New()

				for range maxRounds {
					playRoundWithPlayers(t, g, players, seed)
					totalRounds++

					for i := range hearts.NumPlayers {
						if g.RoundPts[i] == hearts.MoonPoints {
							moonShoots++
							break
						}
					}

					if g.Phase == hearts.PhaseEnd {
						break
					}
				}
			}

			t.Logf("%d games, %d rounds, %d moon shoots (%.1f%%)",
				numGames, totalRounds, moonShoots,
				100*float64(moonShoots)/float64(totalRounds))
		})
	}
}
