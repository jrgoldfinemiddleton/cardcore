package tienlen

import (
	"errors"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// TestSentinelsDealWrongPhase verifies that Deal wraps ErrWrongPhase
// when the game is not waiting to deal.
func TestSentinelsDealWrongPhase(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal: %v", err)
	}
	if err := g.Deal(); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("second Deal: got %v, want ErrWrongPhase", err)
	}
}

// TestSentinelsPlayWrongPhase verifies that Play wraps ErrWrongPhase
// when no hand is being played.
func TestSentinelsPlayWrongPhase(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.Play(0, []cardcore.Card{c(rThree, sSpades)}); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("Play in PhaseDeal: got %v, want ErrWrongPhase", err)
	}
}

// TestSentinelsPlayOutOfTurn verifies that Play wraps ErrOutOfTurn for
// the wrong seat.
func TestSentinelsPlayOutOfTurn(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	if err := g.Play(1, []cardcore.Card{c(rFour, sSpades)}); !errors.Is(err, ErrOutOfTurn) {
		t.Fatalf("Play out of turn: got %v, want ErrOutOfTurn", err)
	}
}

// TestSentinelsPlayWindowOpen verifies that Play wraps ErrIllegalMove
// while the declaration window is open.
func TestSentinelsPlayWindowOpen(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{c(rThree, sSpades)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	if err := g.Play(0, []cardcore.Card{c(rThree, sSpades)}); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("Play with window open: got %v, want ErrIllegalMove", err)
	}
}

// TestSentinelsPlayIllegalBeat verifies that Play wraps ErrIllegalMove
// for a combination that cannot beat the current top.
func TestSentinelsPlayIllegalBeat(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sClubs)},
		[]cardcore.Card{c(rFour, sSpades), c(rFour, sClubs)},
		[]cardcore.Card{c(rThree, sHearts), c(rFive, sSpades)},
	)
	if err := g.Play(0, []cardcore.Card{c(rThree, sSpades)}); err != nil {
		t.Fatalf("Play(lead): %v", err)
	}
	if err := g.Play(1, []cardcore.Card{c(rFour, sSpades)}); err != nil {
		t.Fatalf("Play(beat): %v", err)
	}
	if err := g.Play(2, []cardcore.Card{c(rThree, sHearts)}); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("Play(weaker): got %v, want ErrIllegalMove", err)
	}
}

// TestSentinelsPassWhenLeading verifies that passing wraps
// ErrIllegalMove when there is no pile to respond to.
func TestSentinelsPassWhenLeading(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	if err := g.Play(0, nil); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("pass while leading: got %v, want ErrIllegalMove", err)
	}
}

// TestSentinelsResolvePile verifies ResolvePile's phase and pending
// gates.
func TestSentinelsResolvePile(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.ResolvePile(); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("ResolvePile in PhaseDeal: got %v, want ErrWrongPhase", err)
	}

	g = newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	if err := g.ResolvePile(); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("ResolvePile with nothing pending: got %v, want ErrWrongPhase", err)
	}
}

// TestSentinelsDeclareAutoWin verifies DeclareAutoWin's failure paths.
func TestSentinelsDeclareAutoWin(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.DeclareAutoWin(0); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("DeclareAutoWin in PhaseDeal: got %v, want ErrWrongPhase", err)
	}

	g = newKillerGameWindowOpen(t,
		[]cardcore.Card{c(rThree, sSpades)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	if err := g.DeclareAutoWin(0); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("DeclareAutoWin with no win: got %v, want ErrIllegalMove", err)
	}
	if err := g.DeclareAutoWin(9); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("DeclareAutoWin with invalid seat: got %v, want ErrIllegalMove", err)
	}
}

// TestSentinelsStartPlayWrongPhase verifies that StartPlay wraps
// ErrWrongPhase when no hand is being played.
func TestSentinelsStartPlayWrongPhase(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.StartPlay(); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("StartPlay in PhaseDeal: got %v, want ErrWrongPhase", err)
	}
}

// TestSentinelsEndHandWrongPhase verifies that EndHand wraps
// ErrWrongPhase when the hand is not being scored.
func TestSentinelsEndHandWrongPhase(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.EndHand(); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("EndHand in PhaseDeal: got %v, want ErrWrongPhase", err)
	}
}

// TestSentinelsKillerInterruptChop verifies that out-of-turn chops are
// rejected under Killer rules and outside the play phase.
func TestSentinelsKillerInterruptChop(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if err := g.InterruptChop(0, nil); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("InterruptChop in PhaseDeal: got %v, want ErrWrongPhase", err)
	}

	g = newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rSeven, sSpades), c(rSeven, sClubs),
			c(rSeven, sDiamonds), c(rSeven, sHearts)},
	)
	// The quad of 7s is a legal chop shape held by seat 1; only the
	// variant forbids playing it out of turn.
	err := g.InterruptChop(1, []cardcore.Card{
		c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
	})
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("InterruptChop under Killer: got %v, want ErrIllegalMove", err)
	}
}

// TestSentinelsCanDeclareAutoWinWrongPhase verifies that
// CanDeclareAutoWin wraps ErrWrongPhase when no hand is being played.
func TestSentinelsCanDeclareAutoWinWrongPhase(t *testing.T) {
	g := New(testRNG(), testKillerConfig(2))
	if _, err := g.CanDeclareAutoWin(0); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("CanDeclareAutoWin in PhaseDeal: got %v, want ErrWrongPhase", err)
	}
}

// TestSentinelsInvalidSeat verifies that Play wraps ErrIllegalMove for
// a seat that does not exist.
func TestSentinelsInvalidSeat(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	if err := g.Play(9, []cardcore.Card{c(rThree, sSpades)}); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("Play with invalid seat: got %v, want ErrIllegalMove", err)
	}
}
