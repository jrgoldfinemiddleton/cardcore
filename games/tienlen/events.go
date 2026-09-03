package tienlen

import (
	"slices"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// Event is a single entry in a game's ordered event log. Every accepted
// action appends events; rejected actions append nothing. The set of
// event types is closed: only this package can implement it. The log is
// the input to settlement (doc/games/tienlen/rules.md): chop chains,
// places, and cards played are all recoverable from it without replaying
// the hand.
type Event interface {
	isEvent()
}

// HandStartedEvent records the start of a hand.
type HandStartedEvent struct {
	// Hand is the 0-based hand index.
	Hand int
}

// AutoWinDeclaredEvent records a seat's declaration of an automatic win.
type AutoWinDeclaredEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Seat is the declaring player.
	Seat Seat
	// Kind is the declared automatic win.
	Kind AutoWinKind
	// Top is the tie-break card: the top of the highest pair.
	Top cardcore.Card
}

// PileOpenedEvent records the opening of a pile.
type PileOpenedEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Pile is the 1-based pile number within the hand.
	Pile int
	// Leader is the seat that opened the pile.
	Leader Seat
}

// PlayEvent records a combination played on a pile.
type PlayEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Pile is the 1-based pile number within the hand.
	Pile int
	// Play is the 0-based position of the play within the pile.
	Play int
	// Seat is the player who made the play.
	Seat Seat
	// Combo is the combination played.
	Combo Combo
	// Chop reports whether the play chopped the previous top.
	Chop bool
	// Interrupt reports whether the play was made out of turn (a free
	// chop). It is never true under Killer rules.
	Interrupt bool
	// PaymentFree reports whether the chop targeted a finisher's last
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

// PassEvent records a seat's pass on a pile.
type PassEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Pile is the 1-based pile number within the hand.
	Pile int
	// Seat is the passing player.
	Seat Seat
}

// FinishEvent records a seat taking its place in the finishing order.
type FinishEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Seat is the finishing player.
	Seat Seat
	// Place is the 1-based finishing position.
	Place int
	// Reason is how the place was decided.
	Reason FinishReason
}

// PileClosedEvent records the closing of a pile: its cards are set
// aside.
type PileClosedEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Pile is the 1-based pile number within the hand.
	Pile int
	// NextLeader is the seat that leads the next pile. It is the zero
	// value (seat 0) and meaningless when HasNextLeader is false,
	// because the hand is over.
	NextLeader Seat
	// HasNextLeader reports whether a next pile will be led (false when
	// the hand is over).
	HasNextLeader bool
}

// HandEndedEvent records the end of a hand with its final state.
type HandEndedEvent struct {
	// Hand is the 0-based hand index.
	Hand int
	// Places is the finishing order, first place first.
	Places []Seat
	// CardsPlayed counts the cards each seat played during the hand.
	CardsPlayed []int
}

// Events returns a deep copy of the game's ordered event log: every
// accepted action in order. Mutating the result does not affect the
// game.
func (g *Game) Events() []Event {
	return cloneEvents(g.events)
}

// isEvent seals the Event interface to this package.
func (HandStartedEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (AutoWinDeclaredEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (PileOpenedEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (PlayEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (PassEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (FinishEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (PileClosedEvent) isEvent() {}

// isEvent seals the Event interface to this package.
func (HandEndedEvent) isEvent() {}

// cloneEvents returns a deep copy of the event log.
func cloneEvents(events []Event) []Event {
	if events == nil {
		return nil
	}
	out := make([]Event, len(events))
	for i, e := range events {
		switch ev := e.(type) {
		case PlayEvent:
			ev.Combo.Cards = slices.Clone(ev.Combo.Cards)
			out[i] = ev
		case HandEndedEvent:
			ev.Places = slices.Clone(ev.Places)
			ev.CardsPlayed = slices.Clone(ev.CardsPlayed)
			out[i] = ev
		default:
			out[i] = e
		}
	}
	return out
}
