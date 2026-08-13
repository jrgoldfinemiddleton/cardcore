package ai

import (
	"math/rand/v2"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// Random is a Hearts player that makes uniformly random legal moves.
// It serves as a baseline opponent and as a control in statistical
// tests. The caller controls seeding for reproducible play.
type Random struct {
	// rng is the random number generator used for all move selection.
	// The caller controls seeding for reproducible play.
	rng *rand.Rand
}

// NewRandom creates a Random player using the given random number
// generator. The caller controls seeding for reproducible play.
func NewRandom(rng *rand.Rand) *Random {
	return &Random{rng: rng}
}

// ChoosePass selects three cards at random from the hand at seat via
// rng.Perm, giving every card an equal chance of being passed; Random
// applies no pass strategy. It panics outside the pass phase or when
// the hand holds fewer than PassCount cards.
func (r *Random) ChoosePass(g *hearts.Game, seat hearts.Seat) [hearts.PassCount]cardcore.Card {
	if g.Phase != hearts.PhasePass {
		panic("ai: ChoosePass called outside pass phase")
	}
	hand := g.Hands[seat]
	if hand.Len() < hearts.PassCount {
		panic("ai: ChoosePass called with fewer than 3 cards in hand")
	}
	perm := r.rng.Perm(hand.Len())
	var cards [hearts.PassCount]cardcore.Card
	for i := range hearts.PassCount {
		cards[i] = hand.Cards[perm[i]]
	}
	return cards
}

// ChoosePlay selects a uniformly random card from the legal moves at
// seat via rng.IntN; Random applies no play strategy beyond legality.
// It panics if LegalMoves reports an invalid state.
func (r *Random) ChoosePlay(g *hearts.Game, seat hearts.Seat) cardcore.Card {
	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic("ai: ChoosePlay called in invalid state: " + err.Error())
	}
	return legal[r.rng.IntN(len(legal))]
}
