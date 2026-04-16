package ai

import (
	"math/rand/v2"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// Heuristic is a Hearts player that uses rule-based priority scoring
// to make decisions. It analyzes the game state to count cards, track
// voids, locate dangerous cards, and detect moon threats.
type Heuristic struct {
	rng *rand.Rand
}

// NewHeuristic creates a Heuristic player using the given random number
// generator for tie-breaking. The caller controls seeding for
// reproducible play.
func NewHeuristic(rng *rand.Rand) *Heuristic {
	return &Heuristic{rng: rng}
}

// ChoosePass selects three cards to pass from the hand at seat.
func (h *Heuristic) ChoosePass(g *hearts.Game, seat hearts.Seat) [hearts.PassCount]cardcore.Card {
	if g.Phase != hearts.PhasePass {
		panic("ai: ChoosePass called outside pass phase")
	}
	hand := g.Hands[seat]
	if hand.Len() < hearts.PassCount {
		panic("ai: ChoosePass called with fewer than 3 cards in hand")
	}
	var cards [hearts.PassCount]cardcore.Card
	copy(cards[:], hand.Cards[:hearts.PassCount])
	return cards
}

// ChoosePlay selects a card to play from the hand at seat.
func (h *Heuristic) ChoosePlay(g *hearts.Game, seat hearts.Seat) cardcore.Card {
	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic("ai: ChoosePlay called in invalid state: " + err.Error())
	}
	return legal[0]
}
