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

	ledSuit := g.Trick.LedSuit()
	if g.Hands[seat].HasSuit(ledSuit) {
		return h.chooseFollow(g, legal, a)
	}
	return h.chooseVoid(g, legal, a)
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

// chooseFollow picks a card when we must follow the led suit.
func (h *Heuristic) chooseFollow(g *hearts.Game, legal []cardcore.Card, a analysis) cardcore.Card {
	candidates := make([]cardcore.Card, len(legal))
	copy(candidates, legal)
	h.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	slices.SortStableFunc(candidates, func(x, y cardcore.Card) int {
		return followScore(y, g, a) - followScore(x, g, a)
	})
	return candidates[0]
}

// chooseVoid picks a card when void in the led suit — free to slough anything.
func (h *Heuristic) chooseVoid(g *hearts.Game, legal []cardcore.Card, a analysis) cardcore.Card {
	candidates := make([]cardcore.Card, len(legal))
	copy(candidates, legal)
	h.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	slices.SortStableFunc(candidates, func(x, y cardcore.Card) int {
		return voidScore(y, g, a) - voidScore(x, g, a)
	})
	return candidates[0]
}

// followScore returns how desirable it is to play this card when
// following suit. Higher scores are preferred.
func followScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	_, winnerRank := currentWinner(g)
	wouldWin := card.Rank > winnerRank
	isLast := g.Trick.Count == 3
	trickPts := currentTrickPoints(g)

	// Q♠ when following spades: dump only if a higher spade already
	// guarantees we lose the trick.
	if card == queenOfSpades {
		for i := g.Trick.Leader; i != a.seat; i = nextSeat(i) {
			tc := g.Trick.Cards[i]
			if tc.Suit == cardcore.Spades && tc.Rank > cardcore.Queen {
				return 100
			}
		}
		return -100
	}

	// Moon blocking — prefer winning tricks to deny the shooter
	// lead control. Winning cards get a bonus; losing cards are neutral.
	moonBlock := a.moonThreat >= 0 && a.moonThreat != int(a.seat) && g.TrickNum >= 6
	if moonBlock && wouldWin {
		return int(card.Rank) + 30
	}

	// Last to play, trick has points — strongly prefer losing cards;
	// if forced to win, shed the highest card.
	if isLast && trickPts > 0 {
		if !wouldWin {
			return int(card.Rank) + 50
		}
		return int(card.Rank)
	}

	// Playing under the current winner — safe to shed high cards.
	if !wouldWin {
		return int(card.Rank)
	}

	// Card would win the trick — decide whether that's desirable.
	if isLast {
		// Last to play, trick is clean — win it, but discount if
		// hand is dangerous (winning means leading next).
		danger := highCardRatio(g.Hands[a.seat])
		return int(card.Rank) - danger*2
	}

	// Not last to play — remaining opponents may dump points.
	if trickPts > 0 {
		// Points already in trick — duck.
		return int(cardcore.Ace) - int(card.Rank)
	}

	// Clean trick, not last — small win bonus scaled down by remaining
	// players and hand danger.
	playersLeft := hearts.NumPlayers - 1 - g.Trick.Count
	danger := highCardRatio(g.Hands[a.seat])
	bonus := int(card.Rank) - playersLeft*3 - danger*2
	return bonus
}

// voidScore returns how desirable it is to slough this card when void
// in the led suit. Higher scores are preferred.
func voidScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	if card == queenOfSpades {
		return 100
	}

	if card.Suit == cardcore.Spades && card.Rank > cardcore.Queen {
		return 50
	}

	score := int(card.Rank)

	// Under moon threat, avoid dumping hearts when the threat
	// seat is winning — that feeds their moon attempt.
	moonBlock := a.moonThreat >= 0 && a.moonThreat != int(a.seat) && g.TrickNum >= 6
	if card.Suit == cardcore.Hearts {
		winnerSeat, _ := currentWinner(g)
		if moonBlock && winnerSeat == hearts.Seat(a.moonThreat) {
			score -= 10
		} else {
			score += 10
		}
	}

	suitLen := len(g.Hands[a.seat].CardsOfSuit(card.Suit))
	if suitLen <= 3 {
		score += (4 - suitLen) * 5
	}

	return score
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
	// Under moon threat, lead hearts to block. High hearts
	// (A♥, K♥) win the trick outright; low hearts open the contest.
	moonBlock := a.moonThreat >= 0 && a.moonThreat != int(a.seat) && g.TrickNum >= 6
	if suit == cardcore.Hearts {
		if moonBlock {
			if card.Rank >= cardcore.King {
				score += 30
			} else {
				score += 15
			}
		} else {
			score -= 15
		}
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
