package tienlen

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// TestRecordPlayChainAssignment verifies chop-chain bookkeeping: chains
// escalate within a pile, a finisher's last play is payment-free, a
// chop of a payment-free chop starts a new chain, and chains restart
// with each pile.
func TestRecordPlayChainAssignment(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFive, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rSix, sHearts)},
	)

	// Round 1: a single 2 is chopped by a quad, an eight-card Bomb, and
	// another quad; the third chopper goes out; the next chop is
	// payment-free; the chop after that starts a new chain.
	lead := mustClassify(t, c(rTwo, sSpades))
	quad7 := mustClassify(t,
		c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts))
	bomb8low := mustClassify(t,
		c(rThree, sClubs), c(rThree, sDiamonds),
		c(rFour, sClubs), c(rFour, sDiamonds),
		c(rFive, sClubs), c(rFive, sDiamonds),
		c(rSix, sClubs), c(rSix, sDiamonds))
	quad8 := mustClassify(t,
		c(rEight, sSpades), c(rEight, sClubs), c(rEight, sDiamonds), c(rEight, sHearts))
	bomb8high := mustClassify(t,
		c(rNine, sSpades), c(rNine, sClubs),
		c(rTen, sSpades), c(rTen, sClubs),
		c(rJack, sSpades), c(rJack, sClubs),
		c(rQueen, sSpades), c(rQueen, sClubs))
	quadK := mustClassify(t,
		c(rKing, sSpades), c(rKing, sClubs), c(rKing, sDiamonds), c(rKing, sHearts))

	g.openPile(0)
	g.recordPlay(0, lead, false)
	g.recordPlay(1, quad7, false)
	g.recordPlay(0, bomb8low, false)
	g.recordPlay(1, quad8, true)
	g.recordPlay(0, bomb8high, false)
	g.recordPlay(1, quadK, false)

	type want struct {
		chop        bool
		paymentFree bool
		chain       int
		depth       int
	}
	wants := []want{
		{false, false, 0, 0}, // the lead cannot be a chop
		{true, false, 1, 1},  // the first chop starts chain 1 at depth 1
		{true, false, 1, 2},  // a chop of a chop continues the chain
		{true, false, 1, 3},  // and escalates it
		{true, true, 0, 0},   // chopping a finisher's play earns nothing
		{true, false, 2, 1},  // a chop of a payment-free chop starts chain 2
	}
	if len(g.Pile.Plays) != len(wants) {
		t.Fatalf("got %d plays, want %d", len(g.Pile.Plays), len(wants))
	}
	for i, w := range wants {
		play := g.Pile.Plays[i]
		if play.Chop != w.chop || play.PaymentFree != w.paymentFree ||
			play.Chain != w.chain || play.Depth != w.depth {
			t.Errorf("play %d: got (chop %v, payment-free %v, chain %d, depth %d), "+
				"want (%v, %v, %d, %d)",
				i, play.Chop, play.PaymentFree, play.Chain, play.Depth,
				w.chop, w.paymentFree, w.chain, w.depth)
		}
		if play.Chain > 0 != play.Chop && !play.PaymentFree {
			t.Errorf("play %d: chain %d without a chop", i, play.Chain)
		}
	}

	// The PlayEvent mirror carries the same bookkeeping.
	for i, e := range eventsOfType[PlayEvent](g.Events()) {
		if e.Chop != g.Pile.Plays[i].Chop || e.Chain != g.Pile.Plays[i].Chain ||
			e.Depth != g.Pile.Plays[i].Depth || e.PaymentFree != g.Pile.Plays[i].PaymentFree {
			t.Errorf("event %d does not mirror its pile play", i)
		}
	}

	// Round 2: a new pile restarts chain numbering.
	g.closePile(0, true)
	g.openPile(0)
	g.recordPlay(0, mustClassify(t, c(rTwo, sHearts)), false)
	g.recordPlay(1, mustClassify(t,
		c(rAce, sSpades), c(rAce, sClubs), c(rAce, sDiamonds), c(rAce, sHearts)), false)
	last := g.Pile.Plays[1]
	if !last.Chop || last.Chain != 1 || last.Depth != 1 {
		t.Errorf("new pile: got chop %v chain %d depth %d, want chain 1 depth 1",
			last.Chop, last.Chain, last.Depth)
	}
}

// TestKillerEventOrderAcrossPileLifecycle verifies the ordered event
// stream of a short two-player Killer hand from opening lead to hand
// end.
func TestKillerEventOrderAcrossPileLifecycle(t *testing.T) {
	// Round 1: seat 0 leads; seat 1 takes the pile, self-beats after one
	// pass, and goes out.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFive, sSpades)},
		[]cardcore.Card{c(rFour, sSpades), c(rSix, sSpades)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 0)
	mustPlay(t, g, 1, c(rSix, sSpades))
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}

	want := []string{
		"PileOpenedEvent",
		"PlayEvent",   // seat 0 leads the 3♠
		"PlayEvent",   // seat 1 beats it with the 4♠
		"PassEvent",   // seat 0 passes
		"PlayEvent",   // seat 1 self-beats with the 6♠ and goes out
		"FinishEvent", // seat 1 finishes in first place
		"PileClosedEvent",
		"FinishEvent", // seat 0 is last
		"HandEndedEvent",
	}
	events := g.Events()
	if len(events) != len(want) {
		t.Fatalf("got %d events (%v), want %d", len(events), eventNames(events), len(want))
	}
	for i, name := range eventNames(events) {
		if name != want[i] {
			t.Errorf("event %d: got %s, want %s", i, name, want[i])
		}
	}

	// Spot-check the fields that settlement will rely on.
	plays := eventsOfType[PlayEvent](events)
	if plays[2].Seat != 1 || !plays[2].Finished {
		t.Errorf("final play event: got %+v", plays[2])
	}
	closed := eventsOfType[PileClosedEvent](events)
	if len(closed) != 1 || closed[0].HasNextLeader {
		t.Errorf("got closed events %v, want one with no next leader", closed)
	}
	finishes := eventsOfType[FinishEvent](events)
	if len(finishes) != 2 || finishes[0].Reason != FinishPlayed ||
		finishes[1].Reason != FinishLast {
		t.Errorf("got finish events %v, want played then last", finishes)
	}
	ended := eventsOfType[HandEndedEvent](events)
	if !slices.Equal(ended[0].Places, []Seat{1, 0}) ||
		!slices.Equal(ended[0].CardsPlayed, []int{1, 2}) {
		t.Errorf("got hand-ended event %+v, want places [1 0] and cards [1 2]", ended[0])
	}
}

// TestEventsReturnsDeepCopy verifies that mutating a returned event log
// does not affect the game.
func TestEventsReturnsDeepCopy(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFive, sSpades)},
		[]cardcore.Card{c(rFour, sSpades), c(rSix, sSpades)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 0)
	mustPlay(t, g, 1, c(rSix, sSpades))
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}

	events := g.Events()
	for _, e := range events {
		if play, ok := e.(PlayEvent); ok {
			play.Combo.Cards[0] = c(rTwo, sHearts)
		}
		if ended, ok := e.(HandEndedEvent); ok {
			ended.Places[0] = 9
			ended.CardsPlayed[0] = 99
		}
	}

	fresh := g.Events()
	for _, e := range fresh {
		if play, ok := e.(PlayEvent); ok {
			if play.Combo.Cards[0].Equal(c(rTwo, sHearts)) && play.Seat == 0 {
				t.Error("mutating a returned event changed the game log")
			}
		}
		if ended, ok := e.(HandEndedEvent); ok {
			if ended.Places[0] == 9 || ended.CardsPlayed[0] == 99 {
				t.Error("mutating a returned event changed the game log")
			}
		}
	}
}

// TestRejectedActionsAppendNoEvents verifies that rejected actions do
// not grow the event log.
func TestRejectedActionsAppendNoEvents(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	before := len(g.Events())
	mustFailPlay(t, g, 1, ErrOutOfTurn, c(rFour, sSpades))
	mustFailPass(t, g, 0, ErrIllegalMove) // passing the opening lead
	if got := len(g.Events()); got != before {
		t.Errorf("got %d events, want %d (rejected actions append nothing)", got, before)
	}
}

// mustClassify classifies cards and fails the test on error.
func mustClassify(t *testing.T, cards ...cardcore.Card) Combo {
	t.Helper()
	combo, err := Classify(cards)
	if err != nil {
		t.Fatalf("Classify(%v): %v", cards, err)
	}
	return combo
}

// eventNames returns the type names of the events in order.
func eventNames(events []Event) []string {
	names := make([]string, len(events))
	for i, e := range events {
		name := fmt.Sprintf("%T", e)
		for j, r := range name {
			if r == '.' {
				name = name[j+1:]
				break
			}
		}
		names[i] = name
	}
	return names
}
