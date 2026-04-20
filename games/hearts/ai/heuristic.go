package ai

import (
	"math/rand/v2"
	"slices"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// Heuristic is a Hearts player that uses rule-based priority scoring
// to make decisions. On each call it analyzes the game state —
// counting cards, tracking opponent voids, locating dangerous cards,
// and detecting moon threats — without simulation or lookahead.
//
// Strategy in brief: shed high cards on the pass, duck tricks when
// safe, dump the Queen of Spades on opponents, and attempt to shoot
// the moon when the hand supports it.
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

	a := analyze(g, seat)

	var scoreFunc func(cardcore.Card) int
	if a.considerShoot {
		scoreFunc = func(card cardcore.Card) int {
			return shootPassScore(card, hand)
		}
	} else {
		scoreFunc = func(card cardcore.Card) int {
			return passScore(card, hand)
		}
	}

	// Copy and shuffle for random tie-breaking.
	candidates := make([]cardcore.Card, hand.Len())
	copy(candidates, hand.Cards)
	h.rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	// Stable sort by descending pass score.
	slices.SortStableFunc(candidates, func(a, b cardcore.Card) int {
		return scoreFunc(b) - scoreFunc(a)
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
	if a.shootActive {
		return shootFollowScore(card, g, a)
	}

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

// shootFollowScore returns how desirable it is to play this card when
// following suit and actively shooting the moon. Higher scores are preferred.
func shootFollowScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	_, winnerRank := currentWinner(g)
	wouldWin := card.Rank > winnerRank
	hand := g.Hands[a.seat]

	// Q♠ has three outcomes when shooting:
	//   would lose (-50): an opponent gets a penalty card, killing the shoot.
	//   would win, K♠/A♠ in hand (10): defer Q♠, play the higher spade instead.
	//   would win, no K♠/A♠ (100): play it now — we win the trick.
	if card == queenOfSpades {
		if !wouldWin {
			return -50
		}
		if hand.Contains(kingOfSpades) || hand.Contains(aceOfSpades) {
			return 10
		}
		return 100
	}

	if wouldWin {
		return int(card.Rank) + 30
	}

	return int(cardcore.Ace) - int(card.Rank)
}

// voidScore returns how desirable it is to slough this card when void
// in the led suit. Higher scores are preferred.
func voidScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	if a.shootActive {
		return shootVoidScore(card, g, a)
	}

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

// shootVoidScore returns how desirable it is to slough this card when
// void in the led suit and actively shooting the moon. Higher scores
// are preferred.
func shootVoidScore(card cardcore.Card, _ *hearts.Game, _ analysis) int {
	if card.Suit == cardcore.Hearts || card == queenOfSpades {
		return -100
	}

	return int(cardcore.Ace) - int(card.Rank)
}

// spadeFlushScore returns the spade-flush adjustment when leading
// spades below an unseen Q♠. Returns 0 when the rule does not apply.
func spadeFlushScore(card cardcore.Card, hand *cardcore.Hand, a analysis) int {
	if card.Suit != cardcore.Spades || a.queen == queenPlayed || a.queen == queenInHand {
		return 0
	}
	if hand.Contains(aceOfSpades) || hand.Contains(kingOfSpades) {
		// High spades at risk — avoid leading spades.
		return -20
	}
	// No high spades — safe to flush. Prefer highest below Q♠
	// to maximize chance of winning the trick and leading again.
	return 10 + int(card.Rank)
}

// heartLeadScore returns the hearts-lead adjustment. Generally avoid
// leading hearts; under moon threat, lead them to block (high hearts
// win outright; low hearts open the contest). Returns 0 for non-hearts.
func heartLeadScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	if card.Suit != cardcore.Hearts {
		return 0
	}
	moonBlock := a.moonThreat >= 0 && a.moonThreat != int(a.seat) && g.TrickNum >= 6
	if !moonBlock {
		return -15
	}
	if card.Rank >= cardcore.King {
		return 30
	}
	return 15
}

// leadScore returns how desirable it is to lead this card.
// Higher scores are preferred.
func leadScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	if a.shootActive {
		return shootLeadScore(card, g, a)
	}

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

	score += spadeFlushScore(card, hand, a)
	score += heartLeadScore(card, g, a)

	return score
}

// shootLeadScore returns how desirable it is to lead this card when
// actively shooting the moon. Higher scores are preferred.
func shootLeadScore(card cardcore.Card, g *hearts.Game, a analysis) int {
	suit := card.Suit
	hand := g.Hands[a.seat]

	if card == queenOfSpades {
		return -50
	}

	score := 0

	if suit != cardcore.Hearts {
		// Side suits: aces and kings win clean tricks and maintain lead
		// control before running hearts later. Hearts get plain rank
		// because leading them early signals moonshot intent to opponents.
		switch card.Rank {
		case cardcore.Ace:
			score += 40
		case cardcore.King:
			score += 35
		default:
			score += int(card.Rank)
		}
	} else {
		score += int(card.Rank)
	}

	suitLen := len(hand.CardsOfSuit(suit))
	if suitLen == 1 && suit != cardcore.Hearts {
		score += 15
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

// shootPassScore returns how much the heuristic wants to pass this card
// when shooting the moon. The strategy inverts: keep hearts, Q♠, and
// aces; pass low non-heart cards to shed weak suits.
func shootPassScore(card cardcore.Card, hand *cardcore.Hand) int {
	if card.Suit == cardcore.Hearts {
		return -100
	}

	if card == queenOfSpades {
		return -100
	}

	if card.Rank == cardcore.Ace {
		return -50
	}

	score := int(cardcore.Ace) - int(card.Rank)

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
