package tienlen

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// MinPlayers is the minimum number of players in a Tiến Lên game.
const MinPlayers = 2

// MaxPlayers is the maximum number of players in a Tiến Lên game.
const MaxPlayers = 4

// HandSize is the number of cards dealt to each player.
const HandSize = 13

// DefaultMatchLength is the number of hands in a match when
// Config.MatchLength is zero.
const DefaultMatchLength = 20

// Variant identifies a Tiến Lên ruleset (ADR-010).
type Variant uint8

// Tiến Lên variants.
const (
	// Killer is the San Diego "Killer" ruleset.
	Killer Variant = iota
)

// Phase represents the current phase of a Tiến Lên match.
type Phase uint8

// Phases of a Tiến Lên match, in the order they occur.
const (
	// PhaseDeal is the phase in which players are waiting to be dealt cards.
	PhaseDeal Phase = iota
	// PhasePlay is the phase in which a hand is being played.
	PhasePlay
	// PhaseScore is the phase in which the hand is complete and settled.
	PhaseScore
	// PhaseEnd is the phase in which the match is over.
	PhaseEnd
)

// Seat identifies a player position at the table. Seats are numbered
// 0 through NumPlayers-1; play proceeds in seat order (clockwise under
// Killer rules).
type Seat int

// Config holds the construction parameters of a Tiến Lên game (ADR-010).
type Config struct {
	// Variant selects the fixed ruleset to play.
	Variant Variant
	// NumPlayers is the number of players, from MinPlayers to MaxPlayers.
	NumPlayers int
	// Stake is one whole stake in the caller's smallest integral units.
	// It must be positive and even so that half-stake payments stay
	// integral.
	Stake int
	// MatchLength is the number of hands in the match. Zero selects
	// DefaultMatchLength.
	MatchLength int
}

// AutoWinKind identifies the type of an automatic win.
type AutoWinKind uint8

// Automatic-win kinds. Killer's set is a Killer of 2s and any six pairs.
const (
	// AutoWinQuadTwos is a four-of-a-kind of 2s. It outranks every other
	// automatic win.
	AutoWinQuadTwos AutoWinKind = iota
	// AutoWinSixPairs is any hand that partitions into six pairs.
	AutoWinSixPairs
)

// AutoWin is an automatic win detected in a dealt hand.
type AutoWin struct {
	// Seat is the player holding the automatic win.
	Seat Seat
	// Kind is the type of the automatic win.
	Kind AutoWinKind
	// Top is the tie-break card: the highest card of the highest pair
	// (rank first, then suit).
	Top cardcore.Card
}

// FinishReason records how a seat's place in a hand was decided.
type FinishReason uint8

// Ways a seat's place is decided.
const (
	// FinishPlayed means the seat shed all of its cards in play.
	FinishPlayed FinishReason = iota
	// FinishAutoWin means the seat declared an automatic win.
	FinishAutoWin
	// FinishLast means the seat was the last one still holding cards.
	FinishLast
)

// PilePlay records one combination played on a pile.
type PilePlay struct {
	// Index is the 0-based position of the play within the pile.
	Index int
	// Seat is the player who made the play.
	Seat Seat
	// Combo is the combination played.
	Combo Combo
	// Chop reports whether the play chopped the previous top.
	Chop bool
	// Interrupt reports whether the play was made out of turn (a free
	// chop). It is never true under Killer rules.
	Interrupt bool
	// PaymentFree reports whether the chop targeted a finisher's final
	// play and therefore earns nothing.
	PaymentFree bool
	// Chain is the 1-based number of the chop chain the play belongs
	// to within the pile, or 0 if the play is not part of a payable
	// chain.
	Chain int
	// Depth is the 1-based layer of the play within its chain, or 0
	// when Chain is 0.
	Depth int
	// Finished reports whether the play emptied the seat's hand.
	Finished bool
}

// Pile holds the state of the current pile: one contested combination
// plus the plays made to beat it.
type Pile struct {
	// Open reports whether the pile has a live combination.
	Open bool
	// ID is the 1-based number of the pile within the hand.
	ID int
	// Top is the current live combination.
	Top Combo
	// Holder is the seat whose play is Top, even after going out.
	Holder Seat
	// Locked records which seats have passed on this pile. Under a
	// variant with free chops a locked seat may still chop its way
	// back in (see rulesFor).
	Locked []bool
	// Plays records every play on this pile in order.
	Plays []PilePlay

	// chainSerial is the number of payable chop chains started so far
	// in this pile.
	chainSerial int
}

// variantRules holds the play parameters derived from the variant. These
// are internal mechanics, not caller-configurable options (ADR-010).
type variantRules struct {
	// direction is the seat step per turn: +1 clockwise, -1
	// counterclockwise.
	direction int
	// interruptChops reports whether out-of-turn chops are possible.
	interruptChops bool
	// lockedMayChop reports whether a locked-out player may still chop.
	lockedMayChop bool
}

// Game holds the complete state of a Tiến Lên match.
type Game struct {
	// Config is the construction configuration. Do not mutate it.
	Config Config
	// Phase is the current phase of the match.
	Phase Phase
	// Hand is the 0-based index of the current hand within the match.
	Hand int
	// Hands holds each player's current cards.
	Hands []*cardcore.Hand
	// Undealt holds the cards left out of play in two- and three-player
	// games. It is empty in four-player games.
	Undealt []cardcore.Card
	// DeclaredHands holds the hands of automatic winners, removed from
	// play when they declared. Entries are nil for seats that did not
	// declare.
	DeclaredHands []*cardcore.Hand
	// Active reports which seats are still playing in the current hand.
	Active []bool
	// Turn is the seat whose turn it is to act. It is meaningful only
	// after the declaration window has closed (see StartPlay).
	Turn Seat
	// Places records the seats in finishing order, first place first.
	Places []Seat
	// CardsPlayed counts the cards each seat has played this hand.
	CardsPlayed []int
	// AutoWins holds the automatic wins detected in the deal, in
	// declaration priority order (a Killer of 2s first, then six-pair
	// hands by highest top pair).
	AutoWins []AutoWin
	// AutoWinClaims records which seats have declared during the
	// declaration window.
	AutoWinClaims []bool
	// AutoWinWindowOpen reports whether automatic wins may still be
	// declared. The window opens at Deal and closes at StartPlay; no
	// combination may be played while it is open.
	AutoWinWindowOpen bool
	// OpeningCard is the lowest card still in play. In Killer, the
	// first play of the hand must include it.
	OpeningCard cardcore.Card
	// OpeningLeadRequired reports whether the opening-card rule still
	// applies (no play has been made yet this hand).
	OpeningLeadRequired bool
	// Pile is the pile currently being contested.
	Pile Pile
	// PileHistory holds the closed piles of the current hand, in order.
	PileHistory []Pile
	// PilePendingResolution is true when the current pile is finished
	// but has not yet been resolved, so that clients can observe the
	// final state. Call ResolvePile to advance.
	PilePendingResolution bool
	// SelfBeatContinuation is true when every other player has passed
	// and the holder of the pile must keep playing. The holder's next
	// play either beats their own previous play (the pile continues) or
	// does not (the play opens a new pile and everyone is unlocked).
	SelfBeatContinuation bool

	// events is the match's ordered log of accepted actions.
	events []Event
	// rng is the random number generator used for shuffling. Shared with
	// clones via Game.Clone; not safe for concurrent use. The caller
	// controls seeding for reproducible games.
	rng *rand.Rand
	// rules is the play parameter set derived from Config.Variant.
	rules variantRules
	// nextPileID is the number of piles opened so far this hand.
	nextPileID int
}

// New creates a new Tiến Lên game ready to deal the first hand. The rng
// parameter controls shuffling for reproducible games. It panics if rng
// is nil, if cfg.Variant is unknown, if cfg.NumPlayers is out of range,
// if cfg.Stake is not positive and even, or if cfg.MatchLength is
// negative.
func New(rng *rand.Rand, cfg Config) *Game {
	if rng == nil {
		panic("tienlen: New requires a non-nil *rand.Rand")
	}
	rules := rulesFor(cfg.Variant)
	if cfg.NumPlayers < MinPlayers || cfg.NumPlayers > MaxPlayers {
		panic(fmt.Sprintf(
			"tienlen: NumPlayers %d outside [%d, %d]",
			cfg.NumPlayers, MinPlayers, MaxPlayers,
		))
	}
	if cfg.Stake <= 0 || cfg.Stake%2 != 0 {
		panic(fmt.Sprintf("tienlen: Stake must be positive and even, got %d", cfg.Stake))
	}
	if cfg.MatchLength < 0 {
		panic(fmt.Sprintf("tienlen: MatchLength must be non-negative, got %d", cfg.MatchLength))
	}
	if cfg.MatchLength == 0 {
		cfg.MatchLength = DefaultMatchLength
	}
	return &Game{
		Config: cfg,
		Phase:  PhaseDeal,
		Hands:  make([]*cardcore.Hand, cfg.NumPlayers),
		rules:  rules,
		rng:    rng,
	}
}

// CanDeclareAutoWin reports whether seat may declare an automatic win
// right now. It returns an error when the game is not in the play phase
// or the seat is invalid. It returns false when the declaration window
// has closed, the seat is no longer in the hand, the seat has already
// declared, or the seat holds no automatic win.
func (g *Game) CanDeclareAutoWin(seat Seat) (bool, error) {
	if g.Phase != PhasePlay {
		return false, fmt.Errorf(
			"cannot query auto-win declaration in phase %d: %w", g.Phase, ErrWrongPhase,
		)
	}
	if !g.validSeat(seat) {
		return false, fmt.Errorf("invalid seat %d: %w", seat, ErrIllegalMove)
	}
	if !g.AutoWinWindowOpen || !g.Active[seat] || g.AutoWinClaims[seat] {
		return false, nil
	}
	for _, aw := range g.AutoWins {
		if aw.Seat == seat {
			return true, nil
		}
	}
	return false, nil
}

// Clone returns a deep copy of the game state. The returned Game is
// fully independent — mutating it does not affect the original. The rng
// field is shared (shallow copy); callers must ensure sequential access.
func (g *Game) Clone() *Game {
	clone := *g
	clone.Hands = make([]*cardcore.Hand, len(g.Hands))
	for i, h := range g.Hands {
		if h != nil {
			clone.Hands[i] = cardcore.NewHand(h.Cards)
		}
	}
	clone.DeclaredHands = make([]*cardcore.Hand, len(g.DeclaredHands))
	for i, h := range g.DeclaredHands {
		if h != nil {
			clone.DeclaredHands[i] = cardcore.NewHand(h.Cards)
		}
	}
	clone.Undealt = slices.Clone(g.Undealt)
	clone.Active = slices.Clone(g.Active)
	clone.Places = slices.Clone(g.Places)
	clone.CardsPlayed = slices.Clone(g.CardsPlayed)
	clone.AutoWins = slices.Clone(g.AutoWins)
	clone.AutoWinClaims = slices.Clone(g.AutoWinClaims)
	clone.Pile = clonePile(g.Pile)
	clone.PileHistory = make([]Pile, len(g.PileHistory))
	for i, p := range g.PileHistory {
		clone.PileHistory[i] = clonePile(p)
	}
	clone.events = cloneEvents(g.events)
	return &clone
}

// Deal shuffles and deals 13 cards to each player, then opens the
// automatic-win declaration window. Cards are dealt in batch per seat in
// ascending seat order; this fixed distribution order is part of the
// seeded-reproducibility contract and must not change. In two- and
// three-player games the remaining cards are kept in Undealt, out of
// play.
func (g *Game) Deal() error {
	if g.Phase != PhaseDeal {
		return fmt.Errorf("cannot deal in phase %d: %w", g.Phase, ErrWrongPhase)
	}

	deck := cardcore.NewStandardDeck()
	deck.Shuffle(g.rng)

	for i := range g.Hands {
		cards, err := deck.Deal(HandSize)
		if err != nil {
			return fmt.Errorf("deal failed: %w", err)
		}
		g.Hands[i] = cardcore.NewHand(cards)
		slices.SortFunc(g.Hands[i].Cards, compareCards)
	}
	g.Undealt = slices.Clone(deck.Cards)

	g.Places = nil
	g.CardsPlayed = make([]int, len(g.Hands))
	g.Active = make([]bool, len(g.Hands))
	for i := range g.Active {
		g.Active[i] = true
	}
	g.DeclaredHands = make([]*cardcore.Hand, len(g.Hands))
	g.Pile = Pile{}
	g.PileHistory = nil
	g.nextPileID = 0
	g.PilePendingResolution = false
	g.SelfBeatContinuation = false
	g.AutoWinClaims = make([]bool, len(g.Hands))
	g.OpeningLeadRequired = false
	g.detectAutoWins()
	g.AutoWinWindowOpen = true
	g.Phase = PhasePlay

	g.events = append(g.events, HandStartedEvent{Hand: g.Hand})
	return nil
}

// DeclareAutoWin records seat's declaration of an automatic win.
// Declarations are collected while the declaration window is open and
// resolved in priority order when StartPlay closes the window, so the
// order of DeclareAutoWin calls does not affect places. It returns an
// error when no hand is being played, the declaration window is closed,
// the seat is invalid or no longer in the hand, the seat has already
// declared, or the seat holds no automatic win.
func (g *Game) DeclareAutoWin(seat Seat) error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot declare an auto-win in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	if !g.AutoWinWindowOpen {
		return fmt.Errorf("the declaration window is closed: %w", ErrIllegalMove)
	}
	if !g.validSeat(seat) {
		return fmt.Errorf("invalid seat %d: %w", seat, ErrIllegalMove)
	}
	if !g.Active[seat] {
		return fmt.Errorf("player %d is not in the hand: %w", seat, ErrIllegalMove)
	}
	for _, aw := range g.AutoWins {
		if aw.Seat == seat {
			if g.AutoWinClaims[seat] {
				return fmt.Errorf("player %d has already declared: %w", seat, ErrIllegalMove)
			}
			g.AutoWinClaims[seat] = true
			g.events = append(g.events, AutoWinDeclaredEvent{
				Hand: g.Hand,
				Seat: seat,
				Kind: aw.Kind,
				Top:  aw.Top,
			})
			return nil
		}
	}
	return fmt.Errorf("player %d has no automatic win to declare: %w", seat, ErrIllegalMove)
}

// EndHand advances past the scoring phase. If the match length has been
// reached the match ends; otherwise a new hand may be dealt.
func (g *Game) EndHand() error {
	if g.Phase != PhaseScore {
		return fmt.Errorf("cannot end hand in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	g.Hand++
	if g.Hand >= g.Config.MatchLength {
		g.Phase = PhaseEnd
		return nil
	}
	g.Phase = PhaseDeal
	return nil
}

// InterruptChop plays a chop out of turn (a free chop). No shipped
// variant allows this yet: under Killer rules every call fails with
// ErrIllegalMove. The method exists so that the Standard variant's free
// chops (doc/games/tienlen/rules.md) can be added without changing the
// turn model.
func (g *Game) InterruptChop(seat Seat, cards []cardcore.Card) error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot interrupt in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	if g.AutoWinWindowOpen || g.PilePendingResolution {
		return fmt.Errorf("no pile is accepting plays: %w", ErrIllegalMove)
	}
	if !g.validSeat(seat) {
		return fmt.Errorf("invalid seat %d: %w", seat, ErrIllegalMove)
	}
	if _, err := Classify(cards); err != nil {
		return err
	}
	if !g.rules.interruptChops {
		return fmt.Errorf(
			"the variant does not allow out-of-turn chops: %w", ErrIllegalMove,
		)
	}
	return fmt.Errorf("out-of-turn chops are not implemented: %w", ErrIllegalMove)
}

// Play plays the given cards from seat's hand as one combination, or
// passes when cards is empty. The cards may be given in any order; they
// are classified by the engine. Passing is legal only when responding to
// a live pile.
func (g *Game) Play(seat Seat, cards []cardcore.Card) error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot play in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	if g.AutoWinWindowOpen {
		return fmt.Errorf(
			"the declaration window is open; call StartPlay first: %w", ErrIllegalMove,
		)
	}
	if g.PilePendingResolution {
		return fmt.Errorf(
			"pile pending resolution; call ResolvePile to advance: %w", ErrIllegalMove,
		)
	}
	if !g.validSeat(seat) {
		return fmt.Errorf("invalid seat %d: %w", seat, ErrIllegalMove)
	}
	if seat != g.Turn {
		return fmt.Errorf("not player %d's turn (current: %d): %w", seat, g.Turn, ErrOutOfTurn)
	}
	if !g.Active[seat] {
		return fmt.Errorf("player %d is no longer in the hand: %w", seat, ErrIllegalMove)
	}
	if len(cards) == 0 {
		return g.pass(seat)
	}

	combo, err := Classify(cards)
	if err != nil {
		return err
	}
	for _, c := range combo.Cards {
		if !g.Hands[seat].Contains(c) {
			return fmt.Errorf("player %d does not have %v: %w", seat, c, ErrIllegalMove)
		}
	}
	if g.Pile.Open && g.Pile.Locked[seat] {
		if !g.rules.lockedMayChop {
			return fmt.Errorf(
				"player %d passed and is locked out of this pile: %w", seat, ErrIllegalMove,
			)
		}
		if !combo.Chops(g.Pile.Top, g.Config.Variant) {
			return fmt.Errorf("a locked-out player may only chop: %w", ErrIllegalMove)
		}
	}

	// A play opening a pile is always legal, except that the first play
	// of the hand must include the lowest card in play. A response must
	// beat the current top. During a self-beat continuation every
	// combination is legal: whether it could have beaten the previous
	// play decides if the pile continues (in Killer).
	if !g.Pile.Open {
		if g.OpeningLeadRequired && !slices.ContainsFunc(combo.Cards, func(c cardcore.Card) bool {
			return c.Equal(g.OpeningCard)
		}) {
			return fmt.Errorf(
				"the opening play must include the %v: %w", g.OpeningCard, ErrIllegalMove,
			)
		}
	} else if !g.SelfBeatContinuation && !combo.CanBeat(g.Pile.Top, g.Config.Variant) {
		return fmt.Errorf("%v does not beat %v: %w", combo.Cards, g.Pile.Top.Cards, ErrIllegalMove)
	}

	for _, c := range combo.Cards {
		if !g.Hands[seat].Remove(c) {
			panic(fmt.Sprintf("tienlen: validated card %v not in hand for seat %d", c, seat))
		}
	}
	g.CardsPlayed[seat] += len(combo.Cards)
	finished := g.Hands[seat].Len() == 0

	if !g.Pile.Open {
		g.openPile(seat)
		g.OpeningLeadRequired = false
		g.recordPlay(seat, combo, finished)
	} else if g.SelfBeatContinuation && !combo.CanBeat(g.Pile.Top, g.Config.Variant) {
		// A self-beat play that could not have beaten the previous play
		// closes the old pile and opens a new one in the same action;
		// everyone is unlocked.
		g.closePile(seat, true)
		g.SelfBeatContinuation = false
		g.openPile(seat)
		g.recordPlay(seat, combo, finished)
	} else {
		g.recordPlay(seat, combo, finished)
	}

	if finished {
		g.Active[seat] = false
		g.Places = append(g.Places, seat)
		g.events = append(g.events, FinishEvent{
			Hand:   g.Hand,
			Seat:   seat,
			Place:  len(g.Places),
			Reason: FinishPlayed,
		})
	}
	g.advanceTurn(seat)
	return nil
}

// ResolvePile advances past a finished pile. It must be called only when
// PilePendingResolution is true. If the finisher's last play stood, the
// pile is archived and the next active player after the finisher takes
// the lead. If the hand is over, the last remaining player takes last
// place and the game moves to PhaseScore.
func (g *Game) ResolvePile() error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot resolve a pile in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	if !g.PilePendingResolution {
		return fmt.Errorf("no pile pending resolution: %w", ErrWrongPhase)
	}
	g.PilePendingResolution = false

	if g.activeCount() == 1 {
		last := g.soleActiveSeat()
		g.closePile(0, false)
		g.Places = append(g.Places, last)
		g.events = append(g.events, FinishEvent{
			Hand:   g.Hand,
			Seat:   last,
			Place:  len(g.Places),
			Reason: FinishLast,
		})
		g.events = append(g.events, HandEndedEvent{
			Hand:        g.Hand,
			Places:      slices.Clone(g.Places),
			CardsPlayed: slices.Clone(g.CardsPlayed),
		})
		g.Phase = PhaseScore
		return nil
	}

	leader := g.nextActiveSeat(g.Pile.Holder)
	g.closePile(leader, true)
	g.Turn = leader
	return nil
}

// StartPlay closes the declaration window and starts the first pile.
// Declared automatic wins are resolved in priority order: each declarer
// takes the next available place, and their hand is removed from play.
// If declarations leave more than one player, the holder of the lowest
// remaining card leads and must include it in the first play.
func (g *Game) StartPlay() error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot start play in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	if !g.AutoWinWindowOpen {
		return fmt.Errorf("the declaration window is already closed: %w", ErrIllegalMove)
	}
	g.AutoWinWindowOpen = false

	for _, aw := range g.AutoWins {
		if !g.AutoWinClaims[aw.Seat] {
			continue
		}
		g.DeclaredHands[aw.Seat] = g.Hands[aw.Seat]
		g.Hands[aw.Seat] = cardcore.NewHand(nil)
		g.Active[aw.Seat] = false
		g.Places = append(g.Places, aw.Seat)
		g.events = append(g.events, FinishEvent{
			Hand:   g.Hand,
			Seat:   aw.Seat,
			Place:  len(g.Places),
			Reason: FinishAutoWin,
		})
	}

	switch g.activeCount() {
	case 0:
		g.events = append(g.events, HandEndedEvent{
			Hand:        g.Hand,
			Places:      slices.Clone(g.Places),
			CardsPlayed: slices.Clone(g.CardsPlayed),
		})
		g.Phase = PhaseScore
	case 1:
		last := g.soleActiveSeat()
		g.Places = append(g.Places, last)
		g.events = append(g.events, FinishEvent{
			Hand:   g.Hand,
			Seat:   last,
			Place:  len(g.Places),
			Reason: FinishLast,
		})
		g.events = append(g.events, HandEndedEvent{
			Hand:        g.Hand,
			Places:      slices.Clone(g.Places),
			CardsPlayed: slices.Clone(g.CardsPlayed),
		})
		g.Phase = PhaseScore
	default:
		card, holder := g.lowestActiveCard()
		g.OpeningCard = card
		g.OpeningLeadRequired = true
		g.Turn = holder
	}
	return nil
}

// activeCount returns the number of seats still playing in the hand.
func (g *Game) activeCount() int {
	n := 0
	for _, active := range g.Active {
		if active {
			n++
		}
	}
	return n
}

// advanceTurn moves the turn after an action by seat. A finish that
// leaves one card-holder ends the hand: the pile pauses for resolution.
// Otherwise the turn goes to the next active, unlocked player other
// than the pile holder. If no such player remains, the pile can no
// longer be contested: an active holder must keep playing (self-beat),
// and a finished holder leaves a live last play to be resolved.
func (g *Game) advanceTurn(from Seat) {
	if g.activeCount() == 1 {
		g.SelfBeatContinuation = false
		g.PilePendingResolution = true
		return
	}
	if next, ok := g.nextEligibleResponder(from); ok {
		g.Turn = next
		return
	}
	if !g.Active[g.Pile.Holder] {
		g.SelfBeatContinuation = false
		g.PilePendingResolution = true
		return
	}
	g.SelfBeatContinuation = true
	g.Turn = g.Pile.Holder
}

// closePile archives the current pile to the pile history and records
// its closing. nextLeader and hasNextLeader say who leads next, if
// anyone.
func (g *Game) closePile(nextLeader Seat, hasNextLeader bool) {
	g.events = append(g.events, PileClosedEvent{
		Hand:          g.Hand,
		Pile:          g.Pile.ID,
		NextLeader:    nextLeader,
		HasNextLeader: hasNextLeader,
	})
	g.PileHistory = append(g.PileHistory, g.Pile)
	g.Pile = Pile{}
}

// detectAutoWins finds every seat's automatic win, if any, and stores
// them in declaration priority order: a Killer of 2s first, then
// six-pair hands by highest top pair (rank first, then suit).
func (g *Game) detectAutoWins() {
	g.AutoWins = nil
	for i := range g.Hands {
		if aw, ok := detectAutoWin(Seat(i), g.Hands[i].Cards); ok {
			g.AutoWins = append(g.AutoWins, aw)
		}
	}
	slices.SortStableFunc(g.AutoWins, func(a, b AutoWin) int {
		if a.Kind != b.Kind {
			if a.Kind == AutoWinQuadTwos {
				return -1
			}
			return 1
		}
		return compareCards(b.Top, a.Top)
	})
}

// lowestActiveCard returns the lowest card held by any active player
// under Tiến Lên ordering, and the seat holding it.
func (g *Game) lowestActiveCard() (cardcore.Card, Seat) {
	var lowest cardcore.Card
	holder := Seat(0)
	found := false
	for seat, hand := range g.Hands {
		if !g.Active[seat] {
			continue
		}
		for _, c := range hand.Cards {
			if !found || compareCards(c, lowest) < 0 {
				lowest = c
				holder = Seat(seat)
				found = true
			}
		}
	}
	if !found {
		panic("tienlen: no active player holds a card")
	}
	return lowest, holder
}

// nextActiveSeat returns the next active seat in turn order after from.
// It panics if no seat is active.
func (g *Game) nextActiveSeat(from Seat) Seat {
	s := g.stepSeat(from)
	for !g.Active[s] {
		s = g.stepSeat(s)
	}
	return s
}

// nextEligibleResponder returns the next seat in turn order after from
// that may still respond to the current top: active, not locked out, and
// not the pile holder. The bool reports whether such a seat exists; when
// it is false the returned seat is meaningless.
func (g *Game) nextEligibleResponder(from Seat) (Seat, bool) {
	s := g.stepSeat(from)
	for s != from {
		if g.Active[s] && !g.Pile.Locked[s] && s != g.Pile.Holder {
			return s, true
		}
		s = g.stepSeat(s)
	}
	return 0, false
}

// openPile opens a new pile led by seat. The pile's first play is
// recorded by recordPlay.
func (g *Game) openPile(seat Seat) {
	g.nextPileID++
	g.Pile = Pile{
		Open:   true,
		ID:     g.nextPileID,
		Locked: make([]bool, len(g.Hands)),
	}
	g.events = append(g.events, PileOpenedEvent{
		Hand:   g.Hand,
		Pile:   g.Pile.ID,
		Leader: seat,
	})
}

// pass records seat's pass on the current pile: the seat is locked out
// of the pile and the turn advances.
func (g *Game) pass(seat Seat) error {
	if !g.Pile.Open {
		return fmt.Errorf("cannot pass when opening a pile: %w", ErrIllegalMove)
	}
	if g.SelfBeatContinuation {
		return fmt.Errorf(
			"the sole remaining player in the pile must keep playing: %w", ErrIllegalMove,
		)
	}
	if g.Pile.Locked[seat] {
		return fmt.Errorf("player %d has already passed on this pile: %w", seat, ErrIllegalMove)
	}
	g.Pile.Locked[seat] = true
	g.events = append(g.events, PassEvent{Hand: g.Hand, Pile: g.Pile.ID, Seat: seat})
	g.advanceTurn(seat)
	return nil
}

// recordPlay appends a play of combo by seat to the current pile,
// updates the pile's top and holder, and appends a PlayEvent. A play
// either opens the pile or beats the pile's current top — the previous
// play, by definition. Chop-chain numbers run per pile: a chop of an
// ordinary play or of a payment-free chop starts a new chain at depth 1,
// a chop of a payable chop continues its chain, and a chop of a
// finisher's last play is payment-free and belongs to no chain.
func (g *Game) recordPlay(seat Seat, combo Combo, finished bool) {
	play := PilePlay{
		Index:    len(g.Pile.Plays),
		Seat:     seat,
		Combo:    combo,
		Finished: finished,
	}
	if play.Index > 0 {
		target := g.Pile.Plays[play.Index-1]
		play.Chop = combo.Chops(target.Combo, g.Config.Variant)
		if play.Chop {
			switch {
			case target.Finished:
				play.PaymentFree = true
			case !target.Chop || target.PaymentFree:
				g.Pile.chainSerial++
				play.Chain = g.Pile.chainSerial
				play.Depth = 1
			default:
				play.Chain = target.Chain
				play.Depth = target.Depth + 1
			}
		}
	}
	g.Pile.Plays = append(g.Pile.Plays, play)
	g.Pile.Top = combo
	g.Pile.Holder = seat
	g.events = append(g.events, PlayEvent{
		Hand:        g.Hand,
		Pile:        g.Pile.ID,
		Play:        play.Index,
		Seat:        seat,
		Combo:       combo,
		Chop:        play.Chop,
		PaymentFree: play.PaymentFree,
		Chain:       play.Chain,
		Depth:       play.Depth,
		Finished:    finished,
	})
}

// soleActiveSeat returns the only active seat. It panics unless exactly
// one seat is active.
func (g *Game) soleActiveSeat() Seat {
	seat := Seat(-1)
	for i, active := range g.Active {
		if active {
			if seat >= 0 {
				panic("tienlen: more than one active seat")
			}
			seat = Seat(i)
		}
	}
	if seat < 0 {
		panic("tienlen: no active seat")
	}
	return seat
}

// stepSeat returns the next seat in the variant's direction of play,
// wrapping around the table.
func (g *Game) stepSeat(s Seat) Seat {
	n := Seat(len(g.Hands))
	return (s + Seat(g.rules.direction) + n) % n
}

// validSeat reports whether s is an occupied seat.
func (g *Game) validSeat(s Seat) bool {
	return s >= 0 && s < Seat(len(g.Hands))
}

// clonePile returns a deep copy of a pile.
func clonePile(p Pile) Pile {
	out := p
	out.Locked = slices.Clone(p.Locked)
	out.Top.Cards = slices.Clone(p.Top.Cards)
	if p.Plays != nil {
		out.Plays = make([]PilePlay, len(p.Plays))
		for i, play := range p.Plays {
			out.Plays[i] = play
			out.Plays[i].Combo.Cards = slices.Clone(play.Combo.Cards)
		}
	}
	return out
}

// detectAutoWin reports the automatic win held in cards, if any.
// Killer's set is a Killer of 2s and any six pairs. Pairs are counted by
// dividing each rank's multiplicity by two, so a quad counts as two
// pairs and a triple as one (five pairs plus a triple qualifies).
func detectAutoWin(seat Seat, cards []cardcore.Card) (AutoWin, bool) {
	var byRank [cardcore.NumRanks]int
	twos := 0
	for _, c := range cards {
		byRank[c.Rank]++
		if c.Rank == cardcore.Two {
			twos++
		}
	}
	if twos == 4 {
		return AutoWin{
			Seat: seat,
			Kind: AutoWinQuadTwos,
			Top:  cardcore.Card{Rank: cardcore.Two, Suit: cardcore.Hearts},
		}, true
	}
	pairs := 0
	for _, n := range byRank {
		pairs += n / 2
	}
	if pairs < 6 {
		return AutoWin{}, false
	}
	var top cardcore.Card
	found := false
	for _, c := range cards {
		if byRank[c.Rank] >= 2 && (!found || compareCards(c, top) > 0) {
			top = c
			found = true
		}
	}
	return AutoWin{Seat: seat, Kind: AutoWinSixPairs, Top: top}, true
}

// rulesFor returns the play parameters for a variant. It panics on an
// unknown variant. The switch deliberately has no default case: adding a
// variant to the enum fails the build until its rules are wired here.
func rulesFor(v Variant) variantRules {
	switch v {
	case Killer:
		// Killer: clockwise, no free chops, and a locked-out player may
		// not chop.
		return variantRules{direction: 1}
	}
	panic(fmt.Sprintf("tienlen: unknown variant %d", v))
}
