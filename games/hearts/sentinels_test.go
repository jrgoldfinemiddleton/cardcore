package hearts

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// TestSentinelsLegalMovesWrongPhase verifies that LegalMoves wraps
// ErrWrongPhase when called outside PhasePlay.
func TestSentinelsLegalMovesWrongPhase(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := New(rng)
	_, err := g.LegalMoves(South)
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsLegalMovesTrickPending verifies that LegalMoves wraps
// ErrIllegalMove while a trick is pending resolution.
func TestSentinelsLegalMovesTrickPending(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupFixedHands(rng)
	// Fill the trick without resolving it.
	for range NumPlayers {
		if err := g.PlayCard(g.Turn, g.Hands[g.Turn].Cards[0]); err != nil {
			t.Fatalf("PlayCard: %v", err)
		}
	}
	if !g.TrickPendingResolution {
		t.Fatal("expected trick pending resolution")
	}
	_, err := g.LegalMoves(g.Turn)
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsLegalMovesOutOfTurn verifies that LegalMoves wraps
// ErrOutOfTurn when called for the wrong seat.
func TestSentinelsLegalMovesOutOfTurn(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	wrongSeat := (g.Turn + 1) % NumPlayers
	_, err := g.LegalMoves(wrongSeat)
	if !errors.Is(err, ErrOutOfTurn) {
		t.Fatalf("errors.Is(err, ErrOutOfTurn) = false, got %v", err)
	}
}

// TestSentinelsDealWrongPhase verifies that Deal wraps ErrWrongPhase
// when called outside PhaseDeal.
func TestSentinelsDealWrongPhase(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newPassGame(t, rng)
	err := g.Deal()
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsSetPassWrongPhase verifies that SetPass wraps ErrWrongPhase
// when called outside PhasePass.
func TestSentinelsSetPassWrongPhase(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	var cards [PassCount]cardcore.Card
	err := g.SetPass(g.Turn, cards)
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsSetPassCardNotInHand verifies that SetPass wraps
// ErrIllegalMove when asked to pass a card the seat does not hold.
func TestSentinelsSetPassCardNotInHand(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newPassGame(t, rng)
	var cards [PassCount]cardcore.Card
	cards[0] = c(rAce, sSpades)
	err := g.SetPass(South, cards)
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsSetPassDuplicate verifies that SetPass wraps ErrIllegalMove
// when the same card is passed twice.
func TestSentinelsSetPassDuplicate(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newPassGame(t, rng)
	card := g.Hands[South].Cards[0]
	var cards [PassCount]cardcore.Card
	cards[0] = card
	cards[1] = card
	err := g.SetPass(South, cards)
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsPlayCardWrongPhase verifies that PlayCard wraps ErrWrongPhase
// when called outside PhasePlay.
func TestSentinelsPlayCardWrongPhase(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newPassGame(t, rng)
	err := g.PlayCard(South, g.Hands[South].Cards[0])
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsPlayCardTrickPending verifies that PlayCard wraps
// ErrIllegalMove when a trick is pending resolution.
func TestSentinelsPlayCardTrickPending(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupFixedHands(rng)
	for range NumPlayers {
		if err := g.PlayCard(g.Turn, g.Hands[g.Turn].Cards[0]); err != nil {
			t.Fatalf("PlayCard: %v", err)
		}
	}
	err := g.PlayCard(g.Turn, g.Hands[g.Turn].Cards[0])
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsPlayCardOutOfTurn verifies that PlayCard wraps ErrOutOfTurn
// when called for the wrong seat.
func TestSentinelsPlayCardOutOfTurn(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	wrongSeat := (g.Turn + 1) % NumPlayers
	err := g.PlayCard(wrongSeat, g.Hands[wrongSeat].Cards[0])
	if !errors.Is(err, ErrOutOfTurn) {
		t.Fatalf("errors.Is(err, ErrOutOfTurn) = false, got %v", err)
	}
}

// TestSentinelsPlayCardMustFollowSuit verifies that PlayCard wraps
// ErrIllegalMove when the player can follow suit but does not.
func TestSentinelsPlayCardMustFollowSuit(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupFixedHands(rng)
	// South leads 2♣; West must follow clubs but will try a diamond.
	if err := g.PlayCard(South, twoOfClubs); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	err := g.PlayCard(West, c(rTwo, sDiamonds))
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsPlayCardFirstTrickLead verifies that PlayCard wraps
// ErrIllegalMove when the first trick is not led with 2♣.
func TestSentinelsPlayCardFirstTrickLead(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	leader := g.Turn
	// The leader must lead 2♣; try leading another card from their hand.
	err := g.PlayCard(leader, g.Hands[leader].Cards[1])
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsPlayCardHeartsNotBroken verifies that PlayCard wraps
// ErrIllegalMove when leading hearts before hearts are broken.
func TestSentinelsPlayCardHeartsNotBroken(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupFixedHands(rng)
	// Play first trick through to advance TrickNum and break nothing.
	if err := g.PlayCard(South, twoOfClubs); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	for i := 1; i < NumPlayers; i++ {
		if err := g.PlayCard(g.Turn, g.Hands[g.Turn].Cards[0]); err != nil {
			t.Fatalf("PlayCard: %v", err)
		}
	}
	if err := g.ResolveTrick(); err != nil {
		t.Fatalf("ResolveTrick: %v", err)
	}
	// TrickNum is now 1 (second trick of the round). South attempts to
	// lead hearts before any hearts have been broken.
	g.Turn = South
	err := g.PlayCard(South, c(rFour, sHearts))
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsPlayCardHeartsFirstTrick verifies that PlayCard wraps
// ErrIllegalMove when hearts are played on the first trick.
func TestSentinelsPlayCardHeartsFirstTrick(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupVoidClubs(rng)
	// South leads 2♣; West is void and tries to play a heart.
	if err := g.PlayCard(South, twoOfClubs); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	err := g.PlayCard(West, c(rTwo, sHearts))
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsPlayCardQueenSpadesFirstTrick verifies that PlayCard wraps
// ErrIllegalMove when Q♠ is played on the first trick.
func TestSentinelsPlayCardQueenSpadesFirstTrick(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupVoidClubs(rng)
	// South leads 2♣; West is void and tries to play Q♠.
	if err := g.PlayCard(South, twoOfClubs); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	err := g.PlayCard(West, queenOfSpades)
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsEndRoundWrongPhase verifies that EndRound wraps
// ErrWrongPhase when called outside PhaseScore.
func TestSentinelsEndRoundWrongPhase(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	err := g.EndRound()
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsWinnerWrongPhase verifies that Winner wraps ErrWrongPhase
// when called before the game has ended.
func TestSentinelsWinnerWrongPhase(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	_, err := g.Winner()
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsResolveTrickNoTrickPending verifies that ResolveTrick
// wraps ErrWrongPhase when no trick is pending.
func TestSentinelsResolveTrickNoTrickPending(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newHoldGame(t, rng)
	err := g.ResolveTrick()
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
}

// TestSentinelsResolveTrickIncomplete verifies that ResolveTrick wraps
// ErrIllegalMove when the trick is incomplete.
func TestSentinelsResolveTrickIncomplete(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := setupFixedHands(rng)
	g.TrickPendingResolution = true
	g.Trick.Count = 2
	err := g.ResolveTrick()
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("errors.Is(err, ErrIllegalMove) = false, got %v", err)
	}
}

// TestSentinelsMessagePreserved verifies that wrapping a sentinel
// preserves the descriptive error-message prefix.
func TestSentinelsMessagePreserved(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	g := newPassGame(t, rng)
	err := g.PlayCard(South, g.Hands[South].Cards[0])
	if !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("errors.Is(err, ErrWrongPhase) = false, got %v", err)
	}
	if !strings.Contains(err.Error(), "cannot play in phase") {
		t.Fatalf("error message %q does not contain expected prefix", err.Error())
	}
}
