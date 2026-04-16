package ai

import (
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

func c(r cardcore.Rank, s cardcore.Suit) cardcore.Card {
	return cardcore.Card{Rank: r, Suit: s}
}

func playRoundWithPlayer(t *testing.T, g *hearts.Game, p hearts.Player, seed uint64) {
	t.Helper()

	if err := g.Deal(); err != nil {
		t.Fatalf("seed %d: Deal error: %v", seed, err)
	}

	if g.Phase == hearts.PhasePass {
		for i := hearts.Seat(0); i < hearts.NumPlayers; i++ {
			cards := p.ChoosePass(g.Clone(), i)
			if err := g.SetPass(i, cards); err != nil {
				t.Fatalf("seed %d: SetPass(%d) error: %v", seed, i, err)
			}
		}
	}

	for g.Phase == hearts.PhasePlay {
		seat := g.Turn
		card := p.ChoosePlay(g.Clone(), seat)
		if err := g.PlayCard(seat, card); err != nil {
			t.Fatalf("seed %d: PlayCard(%d, %v) error: %v", seed, seat, card, err)
		}
	}

	roundTotal := 0
	for i := range hearts.NumPlayers {
		roundTotal += g.RoundPts[i]
	}
	if roundTotal != hearts.MoonPoints {
		t.Fatalf("seed %d: sum(RoundPts) = %d, want %d", seed, roundTotal, hearts.MoonPoints)
	}

	if err := g.EndRound(); err != nil {
		t.Fatalf("seed %d: EndRound error: %v", seed, err)
	}
}
