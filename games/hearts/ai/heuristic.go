package ai

import (
	"math/rand/v2"
	"slices"

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

	// Copy and shuffle for random tie-breaking.
	candidates := make([]cardcore.Card, hand.Len())
	copy(candidates, hand.Cards)
	h.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Stable sort by descending pass score.
	slices.SortStableFunc(candidates, func(a, b cardcore.Card) int {
		return passScore(b, hand) - passScore(a, hand)
	})

	var cards [hearts.PassCount]cardcore.Card
	copy(cards[:], candidates[:hearts.PassCount])
	return cards
}

// ChoosePlay selects a card to play from the hand at seat.
func (h *Heuristic) ChoosePlay(g *hearts.Game, seat hearts.Seat) cardcore.Card {
	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic("ai: ChoosePlay called in invalid state: " + err.Error())
	}
	if len(legal) == 1 {
		return legal[0]
	}

	a := analyze(g, seat)

	if g.Trick.Count == 0 {
		return h.chooseLead(g, legal, a)
	}

	// TODO: following strategy (B5).
	return legal[0]
}

// chooseLead picks a card to lead a new trick.
func (h *Heuristic) chooseLead(g *hearts.Game, legal []cardcore.Card, a analysis) cardcore.Card {
	// Shuffle for random tie-breaking, then stable sort by descending score.
	candidates := make([]cardcore.Card, len(legal))
	copy(candidates, legal)
	h.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	slices.SortStableFunc(candidates, func(x, y cardcore.Card) int {
		return leadScore(y, g, a) - leadScore(x, g, a)
	})
	return candidates[0]
}

// leadScore returns how desirable it is to lead this card.
// Higher scores are preferred.
func leadScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	score := 0
	suit := card.Suit
	hand := g.Hands[a.seat]
	suitLen := len(hand.CardsOfSuit(suit))

	if suitLen <= 3 && suit != cardcore.Hearts {
		// Short non-heart suit: bonus for voiding.
		score += (4 - suitLen) * 10
		// Spade rank preference is handled by the flush section below.
		if suit != cardcore.Spades {
			if g.TrickNum <= 3 {
				// Early tricks: lead high to maintain control while voiding.
				score += int(card.Rank)
			} else {
				// Later tricks: lead low to avoid eating sloughed penalties.
				score += int(cardcore.Ace) - int(card.Rank)
			}
		}
	} else {
		// Long suit: prefer low cards (safer leads).
		score += int(cardcore.Ace) - int(card.Rank)
	}

	// Unprotected Q♠: extra urgency to void a non-spade suit.
	if a.queen == queenInHand && !queenProtected(hand) && suit != cardcore.Spades && suitLen <= 3 {
		score += 15
	}

	// Opponent void penalty — they may slough penalties on us.
	// Safe if our card is guaranteed lowest (we can't win the trick).
	if a.opponentVoidInSuit(suit) && !a.guaranteedLowest(card) {
		score -= 40
	}

	// Spade flush: lead spades below Q♠ to draw it out.
	if suit == cardcore.Spades && a.queen != queenPlayed && a.queen != queenInHand {
		if hand.Contains(aceOfSpades) || hand.Contains(kingOfSpades) {
			// High spades at risk — avoid leading spades.
			score -= 20
		} else {
			// No high spades — safe to flush. Prefer highest below Q♠
			// to maximize chance of winning the trick and leading again.
			score += 10
			score += int(card.Rank)
		}
	}

	// Hearts: generally avoid leading.
	if suit == cardcore.Hearts {
		score -= 15
	}

	return score
}

// passScore returns how much the heuristic wants to pass this card.
// Higher scores are passed first.
func passScore(card cardcore.Card, hand *cardcore.Hand) int {
	score := int(card.Rank) // 0 (Two) to 12 (Ace).

	// Q♠ is very dangerous unless well-protected by low spades.
	if card == queenOfSpades {
		if !queenProtected(hand) {
			return 100
		}
		// Well-protected Q♠ is an asset — low priority to pass.
		return 2
	}

	// A♠ and K♠ attract the queen — unless Q♠ is protected in hand.
	if card.Suit == cardcore.Spades && card.Rank > cardcore.Queen {
		if !queenProtected(hand) {
			score += 20
		}
	}

	// Hearts carry penalty points.
	if card.Suit == cardcore.Hearts {
		score += 10
	}

	// Short suit bonus: passing from short suits creates voids.
	suitLen := len(hand.CardsOfSuit(card.Suit))
	if suitLen <= 3 {
		score += (4 - suitLen) * 5
	}

	return score
}

// queenProtected reports whether the hand holds Q♠ with enough low
// spades to avoid winning unwanted spade tricks.
func queenProtected(hand *cardcore.Hand) bool {
	if !hand.Contains(queenOfSpades) {
		return false
	}
	belowQueen := 0
	for _, c := range hand.CardsOfSuit(cardcore.Spades) {
		if c.Rank < cardcore.Queen {
			belowQueen++
		}
	}
	return belowQueen >= 4
}
