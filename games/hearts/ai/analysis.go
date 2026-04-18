package ai

import (
	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// queenLocation describes where the queen of spades is from a player's
// perspective.
type queenLocation uint8

const (
	queenUnknown queenLocation = iota // Q♠ location is unknown (held by an opponent).
	queenInHand                       // The player holds Q♠.
	queenPlayed                       // Q♠ has been played in a completed trick.
	queenPassed                       // The player passed Q♠ this round.
)

// analysis captures what a single seat can legally deduce from the
// visible game state: their own hand, their own pass history, and the
// completed trick history. It is computed fresh on every decision.
type analysis struct {
	seat hearts.Seat

	// played tracks which cards have appeared in completed tricks.
	played [cardcore.NumSuits][cardcore.NumRanks]bool

	// voids tracks which seats are known to be void in a suit,
	// derived from failing to follow suit in trick history.
	voids [hearts.NumPlayers][cardcore.NumSuits]bool

	// queen is the Q♠ location from this seat's perspective.
	queen queenLocation

	// queenHolder is the seat known to hold Q♠ when queen == queenPassed.
	// Only meaningful when queen == queenPassed; undefined otherwise.
	queenHolder hearts.Seat

	// heartsPlayed is the number of hearts seen in completed tricks.
	heartsPlayed int

	// pointsTaken tracks penalty points collected by each seat so far.
	pointsTaken [hearts.NumPlayers]int

	// moonThreat is the seat collecting all distributed points, or -1
	// if no single player has all of them (or no points distributed yet).
	moonThreat int
}

var (
	twoOfClubs    = cardcore.Card{Rank: cardcore.Two, Suit: cardcore.Clubs}
	aceOfSpades   = cardcore.Card{Rank: cardcore.Ace, Suit: cardcore.Spades}
	kingOfSpades  = cardcore.Card{Rank: cardcore.King, Suit: cardcore.Spades}
	queenOfSpades = cardcore.Card{Rank: cardcore.Queen, Suit: cardcore.Spades}
)

// scanTrickHistory records played cards, opponent voids, hearts count, and points from completed tricks.
func (a *analysis) scanTrickHistory(g *hearts.Game) {
	for _, trick := range g.TrickHistory {
		ledSuit := trick.LedSuit()
		winner := trickWinner(trick)

		for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
			card := trick.Cards[s]
			a.played[card.Suit][card.Rank] = true

			if card.Suit != ledSuit {
				a.voids[s][ledSuit] = true
			}

			if card.Suit == cardcore.Hearts {
				a.heartsPlayed++
			}
		}

		a.pointsTaken[winner] += trickPoints(trick)
	}
}

// scanHandVoids marks suits missing from the seat's own hand as void.
func (a *analysis) scanHandVoids(g *hearts.Game, seat hearts.Seat) {
	for suit := cardcore.Suit(0); suit < cardcore.NumSuits; suit++ {
		if !g.Hands[seat].HasSuit(suit) {
			a.voids[seat][suit] = true
		}
	}
}

// locateQueen determines the Q♠ location from the seat's perspective.
func (a *analysis) locateQueen(g *hearts.Game, seat hearts.Seat) {
	if a.played[cardcore.Spades][cardcore.Queen] {
		a.queen = queenPlayed
		return
	}

	if g.Hands[seat].Contains(queenOfSpades) {
		a.queen = queenInHand
		return
	}

	for _, card := range g.PassHistory[seat] {
		if card == queenOfSpades {
			a.queen = queenPassed
			a.queenHolder = passTarget(seat, g.PassDir)
			return
		}
	}
}

// detectMoonThreat checks whether a single player holds all distributed penalty points.
func (a *analysis) detectMoonThreat() {
	totalDistributed := 0
	for i := range hearts.NumPlayers {
		totalDistributed += a.pointsTaken[i]
	}

	if totalDistributed == 0 {
		return
	}

	for i := range hearts.NumPlayers {
		if a.pointsTaken[i] == totalDistributed {
			a.moonThreat = i
			return
		}
	}
}

// guaranteedLowest reports whether card is the lowest remaining card
// of its suit — all lower ranks have already been played.
func (a *analysis) guaranteedLowest(card cardcore.Card) bool {
	for r := cardcore.Rank(0); r < card.Rank; r++ {
		if !a.played[card.Suit][r] {
			return false
		}
	}
	return true
}

// opponentVoidInSuit reports whether any opponent of the asking seat
// is known to be void in the given suit.
func (a *analysis) opponentVoidInSuit(suit cardcore.Suit) bool {
	for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
		if s != a.seat && a.voids[s][suit] {
			return true
		}
	}
	return false
}

// analyze builds a fresh analysis of the visible game state from the given seat's perspective.
func analyze(g *hearts.Game, seat hearts.Seat) analysis {
	a := analysis{
		seat:       seat,
		moonThreat: -1,
	}

	a.scanTrickHistory(g)
	a.scanHandVoids(g, seat)
	a.locateQueen(g, seat)
	a.detectMoonThreat()

	return a
}

// trickWinner returns the seat that won the given completed trick.
func trickWinner(trick hearts.Trick) hearts.Seat {
	ledSuit := trick.LedSuit()
	winner := trick.Leader
	highRank := trick.Cards[winner].Rank

	seat := nextSeat(trick.Leader)
	for range hearts.NumPlayers - 1 {
		c := trick.Cards[seat]
		if c.Suit == ledSuit && c.Rank > highRank {
			winner = seat
			highRank = c.Rank
		}
		seat = nextSeat(seat)
	}

	return winner
}

// trickPoints returns the total penalty points in the given trick.
func trickPoints(trick hearts.Trick) int {
	pts := 0
	for _, c := range trick.Cards {
		if c.Suit == cardcore.Hearts {
			pts++
		}
		if c == queenOfSpades {
			pts += 13
		}
	}
	return pts
}

// nextSeat returns the next seat in clockwise order.
func nextSeat(s hearts.Seat) hearts.Seat {
	return (s + 1) % hearts.NumPlayers
}

// currentTrickPoints returns the penalty points in the in-progress trick.
func currentTrickPoints(g *hearts.Game) int {
	pts := 0
	seat := g.Trick.Leader
	for range g.Trick.Count {
		c := g.Trick.Cards[seat]
		if c.Suit == cardcore.Hearts {
			pts++
		}
		if c == queenOfSpades {
			pts += 13
		}
		seat = nextSeat(seat)
	}
	return pts
}

// currentWinner returns the seat and rank currently winning the
// in-progress trick. Only cards matching the led suit compete.
func currentWinner(g *hearts.Game) (hearts.Seat, cardcore.Rank) {
	ledSuit := g.Trick.LedSuit()
	best := g.Trick.Cards[g.Trick.Leader].Rank
	winner := g.Trick.Leader
	seat := nextSeat(g.Trick.Leader)
	for range g.Trick.Count - 1 {
		c := g.Trick.Cards[seat]
		if c.Suit == ledSuit && c.Rank > best {
			best = c.Rank
			winner = seat
		}
		seat = nextSeat(seat)
	}
	return winner, best
}

// highCardRatio returns the proportion of cards rank Ten or higher in the
// hand, scaled to 0–10. It panics if the hand is empty.
func highCardRatio(hand *cardcore.Hand) int {
	if hand.Len() == 0 {
		panic("ai: highCardRatio called with empty hand")
	}
	count := 0
	for _, c := range hand.Cards {
		if c.Rank >= cardcore.Ten {
			count++
		}
	}
	return count * 10 / hand.Len()
}

// passTarget returns the seat that receives cards from the given seat for the given pass direction.
func passTarget(from hearts.Seat, dir hearts.PassDirection) hearts.Seat {
	switch dir {
	case hearts.PassLeft:
		return (from + 1) % hearts.NumPlayers
	case hearts.PassRight:
		return (from + 3) % hearts.NumPlayers
	case hearts.PassAcross:
		return (from + 2) % hearts.NumPlayers
	default:
		panic("ai: passTarget called for hold round")
	}
}
