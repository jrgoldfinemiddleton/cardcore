package hearts

import (
	"fmt"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// NumPlayers is the number of players in a Hearts game.
const NumPlayers = 4

// HandSize is the number of cards dealt to each player.
const HandSize = 13

// PassCount is the number of cards each player passes per round.
const PassCount = 3

// MaxScore is the score threshold that ends the game.
const MaxScore = 100

// MoonPoints is the total number of penalty points available in a round
// (thirteen hearts plus the queen of spades).
const MoonPoints = 26

// Phase represents the current phase of a Hearts round.
type Phase uint8

const (
	PhaseDeal  Phase = iota // Waiting to deal cards.
	PhasePass               // Players selecting cards to pass.
	PhasePlay               // Trick-taking play in progress.
	PhaseScore              // Round complete, scoring.
	PhaseEnd                // Game over (someone hit MaxScore).
)

// PassDirection determines which direction cards are passed each round.
type PassDirection uint8

const (
	PassLeft   PassDirection = iota // Pass to the player on your left.
	PassRight                       // Pass to the player on your right.
	PassAcross                      // Pass to the player across from you.
	PassHold                        // No passing this round.
)

// NumPassDirections is the number of distinct pass directions in the rotation.
const NumPassDirections = 4

var queenOfSpades = cardcore.Card{Rank: cardcore.Queen, Suit: cardcore.Spades}
var twoOfClubs = cardcore.Card{Rank: cardcore.Two, Suit: cardcore.Clubs}

// Player positions at the table.
type Seat uint8

const (
	South Seat = iota // The human player (in a typical setup).
	West              // The player to South's left.
	North             // The player across from South.
	East              // The player to South's right.
)

// Trick records the cards played in a single trick.
type Trick struct {
	Cards  [NumPlayers]cardcore.Card // The card each player contributed.
	Leader Seat                      // The seat that leads this trick.
	Count  int                       // The number of cards played so far.
}

// LedSuit returns the suit that was led.
func (tr *Trick) LedSuit() cardcore.Suit {
	return tr.Cards[tr.Leader].Suit
}

// Game holds the complete state of a Hearts game.
type Game struct {
	Phase        Phase                      // Current phase of the round.
	Round        int                        // Zero-indexed round number.
	PassDir      PassDirection              // Pass direction for the current round.
	Hands        [NumPlayers]*cardcore.Hand // Each player's current hand.
	Scores       [NumPlayers]int            // Cumulative scores across all rounds.
	RoundPts     [NumPlayers]int            // Penalty points accumulated this round.
	Trick        Trick                      // The trick currently in progress.
	TrickNum     int                        // Zero-indexed trick number within the round.
	Turn         Seat                       // The seat whose turn it is to play.
	HeartsBroken bool                       // Whether hearts have been played this round.

	// Pending passes: passCards[from] = cards to pass.
	passCards [NumPlayers][PassCount]cardcore.Card
	passReady [NumPlayers]bool
}

// New creates a new Hearts game ready to deal the first round.
func New() *Game {
	return &Game{
		Phase: PhaseDeal,
	}
}

// Deal shuffles and deals 13 cards to each player, advancing to the pass
// or play phase.
func (g *Game) Deal() error {
	if g.Phase != PhaseDeal {
		return fmt.Errorf("cannot deal in phase %d", g.Phase)
	}

	deck := cardcore.NewStandardDeck()
	deck.Shuffle()

	for i := range NumPlayers {
		cards, err := deck.Deal(HandSize)
		if err != nil {
			return fmt.Errorf("deal failed: %w", err)
		}
		g.Hands[i] = cardcore.NewHand(cards)
		g.Hands[i].Sort()
	}

	g.RoundPts = [NumPlayers]int{}
	g.HeartsBroken = false
	g.TrickNum = 0
	g.passCards = [NumPlayers][PassCount]cardcore.Card{}
	g.passReady = [NumPlayers]bool{}

	if g.PassDir == PassHold {
		g.Phase = PhasePlay
		g.startFirstTrick()
	} else {
		g.Phase = PhasePass
	}

	return nil
}

// SetPass records a player's chosen cards to pass.
// Once all four players have set their passes, cards are exchanged and
// play begins.
func (g *Game) SetPass(seat Seat, cards [PassCount]cardcore.Card) error {
	if g.Phase != PhasePass {
		return fmt.Errorf("cannot pass in phase %d", g.Phase)
	}
	for i := range cards {
		if !g.Hands[seat].Contains(cards[i]) {
			return fmt.Errorf("player %d does not have %v", seat, cards[i])
		}
		for j := i + 1; j < len(cards); j++ {
			if cards[i].Equal(cards[j]) {
				return fmt.Errorf("duplicate card %v", cards[i])
			}
		}
	}

	g.passCards[seat] = cards
	g.passReady[seat] = true

	if g.allPassesReady() {
		g.executePass()
		g.Phase = PhasePlay
		g.startFirstTrick()
	}

	return nil
}

// PlayCard plays a card from the given seat into the current trick.
func (g *Game) PlayCard(seat Seat, card cardcore.Card) error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot play in phase %d", g.Phase)
	}
	if seat != g.Turn {
		return fmt.Errorf("not player %d's turn (current: %d)", seat, g.Turn)
	}
	if err := g.validatePlay(seat, card); err != nil {
		return err
	}

	if !g.Hands[seat].Remove(card) {
		panic(fmt.Sprintf("validated card %v not in hand for seat %d", card, seat))
	}
	g.Trick.Cards[seat] = card
	g.Trick.Count++

	if card.Suit == cardcore.Hearts {
		g.HeartsBroken = true
	}

	if g.Trick.Count == NumPlayers {
		g.resolveTrick()
	} else {
		g.Turn = nextSeat(g.Turn)
	}

	return nil
}

func (g *Game) validatePlay(seat Seat, card cardcore.Card) error {
	if !g.Hands[seat].Contains(card) {
		return fmt.Errorf("player %d does not have %v", seat, card)
	}

	isLeading := g.Trick.Count == 0

	if isLeading {
		// First trick of the round: must lead 2♣.
		if g.TrickNum == 0 {
			if card != twoOfClubs {
				return fmt.Errorf("first trick must be led with 2♣")
			}
			return nil
		}
		// Cannot lead hearts until broken (unless only hearts remain).
		if card.Suit == cardcore.Hearts && !g.HeartsBroken {
			if !g.onlyHasHearts(seat) {
				return fmt.Errorf("cannot lead hearts until hearts are broken")
			}
		}
		return nil
	}

	// Must follow suit if possible.
	ledSuit := g.Trick.LedSuit()
	if card.Suit != ledSuit && g.Hands[seat].HasSuit(ledSuit) {
		return fmt.Errorf("must follow suit (%v)", ledSuit)
	}

	// First trick: cannot play hearts or Q♠ unless the player has no
	// non-penalty cards (outside the led suit).
	if g.TrickNum == 0 {
		if card.Suit == cardcore.Hearts {
			if g.hasNonPointCards(seat) {
				return fmt.Errorf("cannot play hearts on the first trick")
			}
		}
		if card == queenOfSpades {
			if g.hasNonPointCards(seat) {
				return fmt.Errorf("cannot play Q♠ on the first trick")
			}
		}
	}

	return nil
}

func (g *Game) resolveTrick() {
	winner := g.trickWinner()
	pts := g.trickPoints()
	g.RoundPts[winner] += pts

	g.TrickNum++
	if g.TrickNum == HandSize {
		g.scoreRound()
	} else {
		g.startTrick(winner)
	}
}

func (g *Game) trickWinner() Seat {
	ledSuit := g.Trick.LedSuit()
	winner := g.Trick.Leader
	highRank := g.Trick.Cards[winner].Rank

	seat := nextSeat(g.Trick.Leader)
	for range NumPlayers - 1 {
		c := g.Trick.Cards[seat]
		if c.Suit == ledSuit && c.Rank > highRank {
			winner = seat
			highRank = c.Rank
		}
		seat = nextSeat(seat)
	}

	return winner
}

func (g *Game) trickPoints() int {
	pts := 0
	for _, c := range g.Trick.Cards {
		if c.Suit == cardcore.Hearts {
			pts++
		}
		if c == queenOfSpades {
			pts += 13
		}
	}
	return pts
}

func (g *Game) scoreRound() {
	g.Phase = PhaseScore

	moonShooter := -1
	for i := range NumPlayers {
		if g.RoundPts[i] == MoonPoints {
			moonShooter = i
			break
		}
	}

	if moonShooter >= 0 {
		for i := range NumPlayers {
			if i != moonShooter {
				g.Scores[i] += MoonPoints
			}
		}
	} else {
		for i := range NumPlayers {
			g.Scores[i] += g.RoundPts[i]
		}
	}
}

// EndRound advances past the scoring phase. If any player has reached
// MaxScore the game ends; otherwise a new round begins.
func (g *Game) EndRound() error {
	if g.Phase != PhaseScore {
		return fmt.Errorf("cannot end round in phase %d", g.Phase)
	}

	for i := range NumPlayers {
		if g.Scores[i] >= MaxScore {
			g.Phase = PhaseEnd
			return nil
		}
	}

	g.Round++
	g.PassDir = PassDirection(g.Round % NumPassDirections)
	g.Phase = PhaseDeal

	return nil
}

func (g *Game) startFirstTrick() {
	holder := g.findTwoOfClubs()
	g.startTrick(holder)
}

func (g *Game) startTrick(lead Seat) {
	g.Trick = Trick{Leader: lead}
	g.Turn = lead
}

func (g *Game) findTwoOfClubs() Seat {
	for i := range NumPlayers {
		if g.Hands[i].Contains(twoOfClubs) {
			return Seat(i)
		}
	}
	panic("no player has 2♣")
}

func (g *Game) allPassesReady() bool {
	for i := range NumPlayers {
		if !g.passReady[i] {
			return false
		}
	}
	return true
}

func (g *Game) executePass() {
	received := [NumPlayers][PassCount]cardcore.Card{}

	for from := Seat(0); from < NumPlayers; from++ {
		to := g.passTarget(from)
		received[to] = g.passCards[from]
		for _, c := range g.passCards[from] {
			if !g.Hands[from].Remove(c) {
				panic(fmt.Sprintf("pass card %v not in hand for seat %d", c, from))
			}
		}
	}

	for i := Seat(0); i < NumPlayers; i++ {
		for _, c := range received[i] {
			g.Hands[i].Add(c)
		}
		g.Hands[i].Sort()
	}
}

func (g *Game) passTarget(from Seat) Seat {
	switch g.PassDir {
	case PassLeft:
		return (from + 1) % NumPlayers
	case PassRight:
		return (from + 3) % NumPlayers
	case PassAcross:
		return (from + 2) % NumPlayers
	default:
		panic("no pass target for hold round")
	}
}

func (g *Game) onlyHasHearts(seat Seat) bool {
	for _, c := range g.Hands[seat].Cards {
		if c.Suit != cardcore.Hearts {
			return false
		}
	}
	return true
}

func (g *Game) hasNonPointCards(seat Seat) bool {
	for _, c := range g.Hands[seat].Cards {
		if c.Suit != cardcore.Hearts && c != queenOfSpades {
			return true
		}
	}
	return false
}

func nextSeat(s Seat) Seat {
	return (s + 1) % NumPlayers
}

// Winner returns the seat with the lowest score. Only valid when Phase == PhaseEnd.
func (g *Game) Winner() (Seat, error) {
	if g.Phase != PhaseEnd {
		return 0, fmt.Errorf("game not over")
	}
	best := Seat(0)
	for i := Seat(1); i < NumPlayers; i++ {
		if g.Scores[i] < g.Scores[best] {
			best = i
		}
	}
	return best, nil
}
