package tienlen

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// TestFoldKillerHand verifies the Killer settlement fold over synthetic
// event logs: chain escalation and cancellation, payment-free barriers,
// the finisher freeze, the self-chop freeze, cross-pile independence,
// transfer order, and the place payment.
func TestFoldKillerHand(t *testing.T) {
	tests := []struct {
		name   string
		stake  int
		events []Event
		want   []Transfer
	}{
		{
			name:  "single chop pays one stake",
			stake: 2,
			events: []Event{
				HandStartedEvent{},
				PileOpenedEvent{Pile: 1}, // non-play events are ignored
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				handEnd(1, 0),
			},
			want: []Transfer{
				chopTransfer(0, 1, 2),
				placeTransfer(0, 1, 2),
			},
		},
		{
			name:  "chain escalation cancels earlier debts",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				chopPlay(1, 2, 2, 1, 2),
				chopPlay(1, 3, 0, 1, 3),
				handEnd(0, 2, 1),
			},
			want: []Transfer{
				// Only the last chopped player pays, settling with
				// the last chopper: three stakes from seat 2 to seat 0.
				chopTransfer(2, 0, 6),
				placeTransfer(1, 0, 2),
			},
		},
		{
			name:  "chains separated by a payment-free chop settle independently",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				chopPlay(1, 2, 2, 1, 2),
				freePlay(1, 3, 0),
				chopPlay(1, 4, 1, 2, 1),
				handEnd(2, 1, 0, 3),
			},
			want: []Transfer{
				chopTransfer(1, 2, 4),
				chopTransfer(0, 1, 2),
				placeTransfer(3, 2, 2),
			},
		},
		{
			name:  "same chain numbers in different piles settle independently",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				leadPlay(2, 0, 2),
				chopPlay(2, 1, 0, 1, 1),
				handEnd(1, 0, 2),
			},
			want: []Transfer{
				chopTransfer(0, 1, 2),
				chopTransfer(2, 0, 2),
				placeTransfer(2, 1, 2),
			},
		},
		{
			name:  "out of the blue bomb-class lead earns nothing",
			stake: 2,
			events: []Event{
				// The lead's kind is invisible to the fold: Chain == 0
				// is what keeps it unpaid.
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				handEnd(1, 0),
			},
			want: []Transfer{
				chopTransfer(0, 1, 2),
				placeTransfer(0, 1, 2),
			},
		},
		{
			name:  "payment-free chops never settle",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				freePlay(1, 2, 2),
				// A finisher's payment-free chop stays protected:
				// no one ever earns by chopping a final play.
				freePlay(1, 3, 0),
				handEnd(1, 2, 0),
			},
			want: []Transfer{
				chopTransfer(0, 1, 2),
				placeTransfer(0, 1, 2),
			},
		},
		{
			name:  "self-chop freezes the chain",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				chopPlay(1, 2, 2, 1, 2),
				// Seat 2's self-chop: escalation stops at depth 2
				// and seat 1's debt stands.
				chopPlay(1, 3, 2, 1, 3),
				handEnd(2, 1, 0),
			},
			want: []Transfer{
				chopTransfer(1, 2, 4),
				placeTransfer(0, 2, 2),
			},
		},
		{
			name:  "a chain of nothing but self-chops settles nothing",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 0, 1, 1),
				chopPlay(1, 2, 0, 1, 2),
				handEnd(0, 1),
			},
			want: []Transfer{
				placeTransfer(1, 0, 2),
			},
		},
		{
			name:  "self-chop suffix length is irrelevant",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				chopPlay(1, 2, 1, 1, 2),
				chopPlay(1, 3, 1, 1, 3),
				handEnd(1, 0),
			},
			want: []Transfer{
				chopTransfer(0, 1, 2),
				placeTransfer(0, 1, 2),
			},
		},
		{
			name:  "transfers are never netted",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				leadPlay(2, 0, 0),
				chopPlay(2, 1, 1, 1, 1),
				handEnd(1, 0),
			},
			want: []Transfer{
				chopTransfer(0, 1, 2),
				chopTransfer(0, 1, 2),
				placeTransfer(0, 1, 2),
			},
		},
		{
			name:  "hand without chops settles the place payment only",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				leadPlay(1, 1, 1),
				handEnd(1, 0),
			},
			want: []Transfer{
				placeTransfer(0, 1, 2),
			},
		},
		{
			name:  "declaration-only hand settles the place payment only",
			stake: 2,
			events: []Event{
				AutoWinDeclaredEvent{Seat: 0, Kind: AutoWinQuadTwos},
				HandEndedEvent{Places: []Seat{0, 1}},
			},
			want: []Transfer{
				placeTransfer(1, 0, 2),
			},
		},
		{
			name:  "passes between plays do not disturb the payer lookup",
			stake: 2,
			events: []Event{
				leadPlay(1, 0, 0),
				PassEvent{Pile: 1, Seat: 1},
				chopPlay(1, 1, 2, 1, 1),
				PassEvent{Pile: 1, Seat: 0},
				handEnd(2, 0, 1),
			},
			want: []Transfer{
				chopTransfer(0, 2, 2),
				placeTransfer(1, 2, 2),
			},
		},
		{
			name:  "amounts scale with the stake",
			stake: 10,
			events: []Event{
				leadPlay(1, 0, 0),
				chopPlay(1, 1, 1, 1, 1),
				chopPlay(1, 2, 2, 1, 2),
				chopPlay(1, 3, 0, 1, 3),
				handEnd(0, 2, 1),
			},
			want: []Transfer{
				chopTransfer(2, 0, 30),
				placeTransfer(1, 0, 10),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := foldKillerHand(tt.events, tt.stake, 0)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFoldKillerHandIncompleteSettlesNothing verifies that a hand whose
// events contain no hand-ended event produces no transfers: an
// incomplete hand never settles.
func TestFoldKillerHandIncompleteSettlesNothing(t *testing.T) {
	events := []Event{
		leadPlay(1, 0, 0),
		chopPlay(1, 1, 1, 1, 1),
	}
	if got := foldKillerHand(events, 2, 0); got != nil {
		t.Errorf("got %v, want nil (an incomplete hand never settles)", got)
	}
}

// TestFoldKillerHandIgnoresOtherHands verifies that the fold of one hand
// ignores every other hand's events, including their unfinished chains.
func TestFoldKillerHandIgnoresOtherHands(t *testing.T) {
	events := []Event{
		leadPlay(1, 0, 0),
		chopPlay(1, 1, 1, 1, 1),
		handEnd(1, 0),
		// Hand 1 begins but never ends.
		PlayEvent{Hand: 1, Pile: 1, Play: 0, Seat: 2},
		PlayEvent{Hand: 1, Pile: 1, Play: 1, Seat: 3, Chop: true, Chain: 1, Depth: 1},
	}
	want := []Transfer{
		chopTransfer(0, 1, 2),
		placeTransfer(0, 1, 2),
	}
	if got := foldKillerHand(events, 2, 0); !slices.Equal(got, want) {
		t.Errorf("hand 0: got %v, want %v", got, want)
	}
	if got := foldKillerHand(events, 2, 1); got != nil {
		t.Errorf("hand 1: got %v, want nil (the hand never ended)", got)
	}
}

// TestFoldHandDispatchesVariant verifies that foldHand routes to the
// variant's fold and panics on an unknown variant.
func TestFoldHandDispatchesVariant(t *testing.T) {
	events := []Event{leadPlay(1, 0, 0), handEnd(1, 0)}
	cfg := testKillerConfig(2)
	got := foldHand(events, cfg, 0)
	want := foldKillerHand(events, cfg.Stake, 0)
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	bad := cfg
	bad.Variant = Variant(42)
	assertPanics(t, func() { foldHand(events, bad, 0) })
}

// TestNewGameLedgerIsZero verifies that a fresh match starts with zero
// balances and an empty transfer record, and that the accessors return
// copies.
func TestNewGameLedgerIsZero(t *testing.T) {
	g := New(testRNG(), testKillerConfig(3))
	balances := g.Balances()
	if want := []int{0, 0, 0}; !slices.Equal(balances, want) {
		t.Errorf("got balances %v, want %v", balances, want)
	}
	if got := g.Transfers(); len(got) != 0 {
		t.Errorf("got transfers %v, want none before the first hand settles", got)
	}
	balances[0] = 99
	if got := g.Balances()[0]; got != 0 {
		t.Errorf("got balance %d, want 0 (mutating the copy changed the game)", got)
	}
}

// TestWinner verifies the match outcome: an error before the match ends,
// the unique highest balance wins outright, and tied highest balances
// draw.
func TestWinner(t *testing.T) {
	t.Run("wrong phase before the match ends", func(t *testing.T) {
		g := New(testRNG(), testKillerConfig(2))
		for _, phase := range []Phase{PhaseDeal, PhasePlay, PhaseScore} {
			g.Phase = phase
			if _, _, err := g.Winner(); !errors.Is(err, ErrWrongPhase) {
				t.Errorf("phase %d: got error %v, want ErrWrongPhase", phase, err)
			}
		}
	})
	t.Run("unique highest balance wins", func(t *testing.T) {
		g := New(testRNG(), testKillerConfig(3))
		g.Phase = PhaseEnd
		g.balances = []int{-4, 6, -2}
		seat, ok, err := g.Winner()
		if seat != 1 || !ok || err != nil {
			t.Errorf("got (%d, %v, %v), want (1, true, nil)", seat, ok, err)
		}
	})
	t.Run("tied highest balance draws", func(t *testing.T) {
		g := New(testRNG(), testKillerConfig(3))
		g.Phase = PhaseEnd
		g.balances = []int{6, -12, 6}
		if _, ok, err := g.Winner(); ok || err != nil {
			t.Errorf("got (_, %v, %v), want (false, nil)", ok, err)
		}
	})
	t.Run("three-way tie draws", func(t *testing.T) {
		g := New(testRNG(), testKillerConfig(4))
		g.Phase = PhaseEnd
		g.balances = []int{4, 4, 4, -12}
		if _, ok, err := g.Winner(); ok || err != nil {
			t.Errorf("got (_, %v, %v), want (false, nil)", ok, err)
		}
	})
	t.Run("all-zero balances draw", func(t *testing.T) {
		g := New(testRNG(), testKillerConfig(2))
		g.Phase = PhaseEnd
		if _, ok, err := g.Winner(); ok || err != nil {
			t.Errorf("got (_, %v, %v), want (false, nil)", ok, err)
		}
	})
	t.Run("highest balance wins among mixed signs", func(t *testing.T) {
		g := New(testRNG(), testKillerConfig(3))
		g.Phase = PhaseEnd
		g.balances = []int{-6, 8, -2}
		seat, ok, err := g.Winner()
		if seat != 1 || !ok || err != nil {
			t.Errorf("got (%d, %v, %v), want (1, true, nil)", seat, ok, err)
		}
	})
}

// TestKillerChainSettlementIntegration settles a scripted hand whose
// pile carries a four-deep chop chain: the last chop collects four
// stakes from the player it chopped, every earlier chain debt is
// forgiven, and the last-place payment is one stake.
func TestKillerChainSettlementIntegration(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{ // seat 0
			c(rThree, sSpades), c(rThree, sHearts),
			c(rEight, sSpades), c(rEight, sClubs), c(rEight, sDiamonds), c(rEight, sHearts),
			c(rTwo, sSpades),
		},
		[]cardcore.Card{ // seat 1
			c(rFour, sSpades),
			c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
			c(rNine, sSpades), c(rNine, sClubs), c(rNine, sHearts),
			c(rTen, sSpades), c(rTen, sClubs),
			c(rJack, sSpades), c(rJack, sClubs),
			c(rQueen, sSpades), c(rQueen, sClubs),
		},
		[]cardcore.Card{ // seat 2
			c(rThree, sClubs), c(rThree, sDiamonds),
			c(rFour, sClubs), c(rFour, sDiamonds),
			c(rFive, sClubs), c(rFive, sDiamonds), c(rFive, sHearts),
			c(rSix, sClubs), c(rSix, sDiamonds),
		},
		[]cardcore.Card{ // seat 3
			c(rFour, sHearts),
		},
	)

	// Round 1: seat 0 opens with the 3♠ and seat 3 goes out on the 4♥;
	// the finisher's play stands, and seat 0 inherits the lead. (The
	// chain pile needs everyone unlocked: a pass locks a seat out of the
	// pile for good, so the 2♠ can only be chopped on a fresh pile.)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 2)
	mustPlay(t, g, 3, c(rFour, sHearts))
	mustPass(t, g, 0)
	mustPass(t, g, 1)
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}

	// Round 2: seat 0 leads the 2♠; a four-chop chain follows (quad 7s,
	// low Bomb, quad 8s, high Bomb).
	mustPlay(t, g, 0, c(rTwo, sSpades))
	mustPlay(t, g, 1, c(rSeven, sSpades), c(rSeven, sClubs),
		c(rSeven, sDiamonds), c(rSeven, sHearts))
	mustPlay(t, g, 2, c(rThree, sClubs), c(rThree, sDiamonds),
		c(rFour, sClubs), c(rFour, sDiamonds),
		c(rFive, sClubs), c(rFive, sDiamonds),
		c(rSix, sClubs), c(rSix, sDiamonds))
	mustPlay(t, g, 0, c(rEight, sSpades), c(rEight, sClubs),
		c(rEight, sDiamonds), c(rEight, sHearts))
	mustPlay(t, g, 1, c(rNine, sSpades), c(rNine, sClubs),
		c(rTen, sSpades), c(rTen, sClubs),
		c(rJack, sSpades), c(rJack, sClubs),
		c(rQueen, sSpades), c(rQueen, sClubs))
	mustPass(t, g, 2)
	mustPass(t, g, 0)

	// Round 3: seat 1 cannot beat its own Bomb, opens a new pile with
	// the 9♥, and goes out; the play stands.
	mustPlay(t, g, 1, c(rNine, sHearts))
	mustPass(t, g, 2)
	mustPass(t, g, 0)
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}

	// Round 4: seat 2 leads its last card and goes out; seat 0 is last.
	mustPlay(t, g, 2, c(rFive, sHearts))
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}
	if g.Phase != PhaseScore {
		t.Fatalf("got phase %d, want PhaseScore", g.Phase)
	}

	want := []Transfer{
		{Hand: 0, From: 0, To: 1, Amount: 8, Reason: TransferChop},
		{Hand: 0, From: 0, To: 3, Amount: 2, Reason: TransferPlace},
	}
	if got := g.Transfers(); !slices.Equal(got, want) {
		t.Errorf("got transfers %v, want %v", got, want)
	}
	wantBalances := []int{-10, 8, 0, 2}
	if got := g.Balances(); !slices.Equal(got, wantBalances) {
		t.Errorf("got balances %v, want %v", got, wantBalances)
	}
	assertLedgerInvariants(t, g)

	// The accessors return copies.
	transfers := g.Transfers()
	transfers[0].Amount = 99
	if got := g.Transfers()[0].Amount; got != 8 {
		t.Errorf("got amount %d, want 8 (mutating the copy changed the game)", got)
	}
}

// TestKillerSelfChopFreezeSettlementIntegration verifies the ruled
// self-chop settlement through the state machine: a self-chop stops the
// escalation but never cancels what the chopper already earned — the
// chain settles at the last chop of another player's play, at that
// chop's depth.
func TestKillerSelfChopFreezeSettlementIntegration(t *testing.T) {
	g := selfChopFixture(t)

	want := []Transfer{
		{Hand: 0, From: 1, To: 2, Amount: 4, Reason: TransferChop},
		{Hand: 0, From: 1, To: 3, Amount: 2, Reason: TransferPlace},
	}
	if got := g.Transfers(); !slices.Equal(got, want) {
		t.Errorf("got transfers %v, want %v", got, want)
	}
	wantBalances := []int{0, -6, 4, 2}
	if got := g.Balances(); !slices.Equal(got, wantBalances) {
		t.Errorf("got balances %v, want %v", got, wantBalances)
	}
	assertLedgerInvariants(t, g)
}

// TestKillerDeclarationSettlementIntegration verifies settlement of a
// hand that ends without a card being played: declarations take places
// by priority, the remaining player is last, and the only transfer is
// the last-place payment of one stake.
func TestKillerDeclarationSettlementIntegration(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{ // seat 0: quad of 2s
			c(rThree, sSpades), c(rFive, sSpades), c(rEight, sDiamonds),
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
		},
		[]cardcore.Card{ // seat 1: six pairs, top pair aces
			c(rFour, sSpades), c(rFour, sHearts),
			c(rSix, sSpades), c(rSix, sHearts),
			c(rNine, sSpades), c(rNine, sHearts),
			c(rTen, sHearts),
			c(rJack, sSpades), c(rJack, sHearts),
			c(rKing, sSpades), c(rKing, sHearts),
			c(rAce, sSpades), c(rAce, sHearts),
		},
		[]cardcore.Card{ // seat 2: no automatic win
			c(rThree, sHearts), c(rFour, sDiamonds), c(rSeven, sSpades), c(rEight, sSpades),
		},
	)

	// Claims arrive in reverse priority order.
	for _, seat := range []Seat{1, 0} {
		if err := g.DeclareAutoWin(seat); err != nil {
			t.Fatalf("DeclareAutoWin(%d): %v", seat, err)
		}
	}
	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}

	if g.Phase != PhaseScore {
		t.Fatalf("got phase %d, want PhaseScore (one player left)", g.Phase)
	}
	if plays := eventsOfType[PlayEvent](g.Events()); len(plays) != 0 {
		t.Fatalf("got %d play events, want none (the hand ended at the deal)", len(plays))
	}
	want := []Transfer{
		{Hand: 0, From: 2, To: 0, Amount: 2, Reason: TransferPlace},
	}
	if got := g.Transfers(); !slices.Equal(got, want) {
		t.Errorf("got transfers %v, want %v", got, want)
	}
	wantBalances := []int{2, 0, -2}
	if got := g.Balances(); !slices.Equal(got, wantBalances) {
		t.Errorf("got balances %v, want %v", got, wantBalances)
	}
	assertLedgerInvariants(t, g)
}

// TestLedgerCloneIndependenceIntegration verifies that cloning a settled
// game copies the ledger and that the copies are independent in both
// directions.
func TestLedgerCloneIndependenceIntegration(t *testing.T) {
	g := selfChopFixture(t)

	clone := g.Clone()
	if !slices.Equal(clone.Balances(), g.Balances()) {
		t.Errorf("got clone balances %v, want %v", clone.Balances(), g.Balances())
	}
	if !slices.Equal(clone.Transfers(), g.Transfers()) {
		t.Errorf("got clone transfers %v, want %v", clone.Transfers(), g.Transfers())
	}

	clone.balances[0] += 100
	clone.transfers = append(clone.transfers, Transfer{Hand: 9, From: 0, To: 1, Amount: 1})
	if got := g.Balances()[0]; got != 0 {
		t.Errorf("got balance %d, want 0 (the original changed through the clone)", got)
	}
	if got := len(g.Transfers()); got != 2 {
		t.Errorf("got %d transfers, want 2 (the original changed through the clone)", got)
	}

	// The clone proceeds independently to the match end.
	if err := clone.EndHand(); err != nil {
		t.Fatalf("clone EndHand: %v", err)
	}
	if clone.Phase != PhaseEnd {
		t.Errorf("got clone phase %d, want PhaseEnd", clone.Phase)
	}
	if g.Phase != PhaseScore {
		t.Errorf("got phase %d, want PhaseScore (the original changed through the clone)", g.Phase)
	}
}

// TestLedgerFullMatchIntegration drives a seeded three-hand match and
// verifies the ledger across hand boundaries: balances persist through
// Deal, every settled hand passes the invariant suite, the transfer
// record grows hand by hand, and Winner agrees with the balances.
func TestLedgerFullMatchIntegration(t *testing.T) {
	cfg := testKillerConfig(3)
	cfg.MatchLength = 3
	g := New(rand.New(rand.NewPCG(3003, 4)), cfg)
	var previous []int
	for hand := 0; hand < cfg.MatchLength; hand++ {
		if err := g.Deal(); err != nil {
			t.Fatalf("hand %d: Deal: %v", hand, err)
		}
		if previous != nil && !slices.Equal(g.Balances(), previous) {
			t.Errorf("hand %d: got balances %v after Deal, want %v (a deal never settles)",
				hand, g.Balances(), previous)
		}
		driveHand(t, g, true)
		if g.Phase != PhaseScore {
			t.Fatalf("hand %d: got phase %d, want PhaseScore", hand, g.Phase)
		}
		assertLedgerInvariants(t, g)
		previous = g.Balances()
		if err := g.EndHand(); err != nil {
			t.Fatalf("hand %d: EndHand: %v", hand, err)
		}
	}
	if g.Phase != PhaseEnd {
		t.Fatalf("got phase %d, want PhaseEnd after %d hands", g.Phase, cfg.MatchLength)
	}
	assertWinnerConsistent(t, g)
	if err := g.EndHand(); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("got error %v, want ErrWrongPhase (the match is over)", err)
	}
}

// TestLedgerRandomMatchIntegration plays random full matches and
// verifies the ledger invariants after every hand: zero-sum, well-formed
// transfers, replay consistency, and a consistent Winner.
func TestLedgerRandomMatchIntegration(t *testing.T) {
	for try := range 200 {
		t.Run(fmt.Sprintf("seed-%d", try), func(t *testing.T) {
			rng := rand.New(rand.NewPCG(uint64(try), 5))
			cfg := testKillerConfig(2 + try%3)
			cfg.MatchLength = 2
			g := New(rng, cfg)
			for g.Phase != PhaseEnd {
				switch g.Phase {
				case PhaseDeal:
					if err := g.Deal(); err != nil {
						t.Fatalf("Deal: %v", err)
					}
				case PhasePlay:
					driveRandomHand(t, g, rng)
				case PhaseScore:
					assertLedgerInvariants(t, g)
					if err := g.EndHand(); err != nil {
						t.Fatalf("EndHand: %v", err)
					}
				case PhaseEnd:
					t.Fatal("unreachable: the loop condition excludes PhaseEnd")
				}
			}
			assertWinnerConsistent(t, g)
		})
	}
}

// leadPlay returns the PlayEvent of an ordinary play: never a chop,
// never payable.
func leadPlay(pile, play int, seat Seat) PlayEvent {
	return PlayEvent{Pile: pile, Play: play, Seat: seat}
}

// chopPlay returns the PlayEvent of a payable chop at the given chain
// and depth.
func chopPlay(pile, play int, seat Seat, chain, depth int) PlayEvent {
	return PlayEvent{Pile: pile, Play: play, Seat: seat, Chop: true, Chain: chain, Depth: depth}
}

// freePlay returns the PlayEvent of a payment-free chop (the target was
// a finisher's final play).
func freePlay(pile, play int, seat Seat) PlayEvent {
	return PlayEvent{Pile: pile, Play: play, Seat: seat, Chop: true, PaymentFree: true}
}

// handEnd returns the HandEndedEvent closing hand 0 with the given
// finishing order, first place first.
func handEnd(places ...Seat) HandEndedEvent {
	return HandEndedEvent{Places: places}
}

// chopTransfer returns the want Transfer settling a chop chain in hand 0.
func chopTransfer(from, to Seat, amount int) Transfer {
	return Transfer{From: from, To: to, Amount: amount, Reason: TransferChop}
}

// placeTransfer returns the want Transfer settling the place payment in
// hand 0.
func placeTransfer(from, to Seat, amount int) Transfer {
	return Transfer{From: from, To: to, Amount: amount, Reason: TransferPlace}
}

// selfChopFixture drives the ruled self-chop scenario to PhaseScore:
// seat 0's 2♠ is chopped by seat 1's Killer, chopped by seat 2's Bomb,
// and then self-chopped by seat 2's Killer after everyone passes.
func selfChopFixture(t *testing.T) *Game {
	t.Helper()
	g := newKillerGame(t,
		[]cardcore.Card{ // seat 0
			c(rThree, sSpades), c(rThree, sHearts), c(rTwo, sSpades),
		},
		[]cardcore.Card{ // seat 1
			c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
			c(rNine, sHearts),
		},
		[]cardcore.Card{ // seat 2
			c(rThree, sClubs), c(rThree, sDiamonds),
			c(rFour, sClubs), c(rFour, sDiamonds),
			c(rFive, sClubs), c(rFive, sDiamonds),
			c(rSix, sClubs), c(rSix, sDiamonds),
			c(rKing, sSpades), c(rKing, sClubs), c(rKing, sDiamonds), c(rKing, sHearts),
		},
		[]cardcore.Card{ // seat 3
			c(rFour, sHearts),
		},
	)

	// Round 1: seat 0 opens with the 3♠ and seat 3 goes out on the 4♥;
	// the finisher's play stands, and seat 0 inherits the lead. (The
	// chain pile needs everyone unlocked: a pass locks a seat out of the
	// pile for good, so the 2♠ can only be chopped on a fresh pile.)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPass(t, g, 1)
	mustPass(t, g, 2)
	mustPlay(t, g, 3, c(rFour, sHearts))
	mustPass(t, g, 0)
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}

	// Round 2: seat 0 leads the 2♠; seat 1 kills it, seat 2 bombs seat
	// 1, and seat 2 self-chops with a Killer after everyone passes,
	// going out with it.
	mustPlay(t, g, 0, c(rTwo, sSpades))
	mustPlay(t, g, 1, c(rSeven, sSpades), c(rSeven, sClubs),
		c(rSeven, sDiamonds), c(rSeven, sHearts))
	mustPlay(t, g, 2, c(rThree, sClubs), c(rThree, sDiamonds),
		c(rFour, sClubs), c(rFour, sDiamonds),
		c(rFive, sClubs), c(rFive, sDiamonds),
		c(rSix, sClubs), c(rSix, sDiamonds))
	mustPass(t, g, 0)
	mustPass(t, g, 1)
	mustPlay(t, g, 2, c(rKing, sSpades), c(rKing, sClubs), c(rKing, sDiamonds), c(rKing, sHearts))
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}

	// Round 3: seat 0 leads its last card and goes out; seat 1 is last.
	mustPlay(t, g, 0, c(rThree, sHearts))
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}
	if g.Phase != PhaseScore {
		t.Fatalf("got phase %d, want PhaseScore", g.Phase)
	}
	return g
}

// assertLedgerInvariants verifies the ledger invariants that must hold
// when a hand has just settled: balances sum to zero, transfers are
// well-formed (positive amounts, distinct in-range seats), applying the
// transfer record from zero reproduces the balances, folding the event
// log reproduces every settled hand's transfers exactly (replay
// consistency), and the just-settled hand's place transfer matches its
// finishing order.
func assertLedgerInvariants(t *testing.T, g *Game) {
	t.Helper()
	balances := g.Balances()
	sum := 0
	for _, b := range balances {
		sum += b
	}
	if sum != 0 {
		t.Errorf("got balance sum %d, want 0", sum)
	}
	transfers := g.Transfers()
	applied := make([]int, len(balances))
	for _, tr := range transfers {
		if tr.Amount <= 0 {
			t.Errorf("transfer %+v has a non-positive amount", tr)
		}
		if tr.From == tr.To {
			t.Errorf("transfer %+v is a self-transfer", tr)
		}
		if n := Seat(len(balances)); tr.From < 0 || tr.From >= n || tr.To < 0 || tr.To >= n {
			t.Errorf("transfer %+v has a seat out of range", tr)
		}
		applied[tr.From] -= tr.Amount
		applied[tr.To] += tr.Amount
	}
	if !slices.Equal(applied, balances) {
		t.Errorf("got balances %v, want %v from applying the transfer record", balances, applied)
	}
	// Replay consistency: folding the event log reproduces every settled hand's transfers exactly.
	for hand := 0; hand <= g.Hand; hand++ {
		var got []Transfer
		for _, tr := range transfers {
			if tr.Hand == hand {
				got = append(got, tr)
			}
		}
		if want := foldHand(g.Events(), g.Config, hand); !slices.Equal(got, want) {
			t.Errorf("hand %d: got transfers %v, want %v from folding the event log",
				hand, got, want)
		}
	}
	assertPileHistoryChopsMatch(t, g)
	var places []Transfer
	for _, tr := range transfers {
		if tr.Hand == g.Hand && tr.Reason == TransferPlace {
			places = append(places, tr)
		}
	}
	if len(places) != 1 {
		t.Fatalf("got %d place transfers for hand %d, want 1", len(places), g.Hand)
	}
	place := places[0]
	last := g.Places[len(g.Places)-1]
	if place.Amount != g.Config.Stake || place.From != last || place.To != g.Places[0] {
		t.Errorf("got place transfer %+v, want %d from %d to %d",
			place, g.Config.Stake, last, g.Places[0])
	}
}

// assertPileHistoryChopsMatch recomputes every closed pile's chop
// chains from scratch — chops via Combo.Chops, payment-free marking via
// the target's Finished flag, chain and depth per recordPlay's
// assignment — and verifies that the current hand's chop transfers match
// the independent recomputation. The fold's replay consistency cannot
// catch a recordPlay regression that corrupts PilePlay and PlayEvent
// identically; this cross-check can.
func assertPileHistoryChopsMatch(t *testing.T, g *Game) {
	t.Helper()
	var want []Transfer
	for _, pile := range g.PileHistory {
		// chops[i] is true if play i chops the previous play.
		chop := make([]bool, len(pile.Plays))
		// free[i] is true if the chop is payment-free.
		free := make([]bool, len(pile.Plays))
		// chain[i] is the index of the chain in settling.
		chain := make([]int, len(pile.Plays))
		// depth[i] is the depth of the chain at play i.
		depth := make([]int, len(pile.Plays))
		// settling[i] is the index of the last chop in the chain that
		// settles at play i, or -1 if the chain has not yet settled.
		var settling []int
		for i := 1; i < len(pile.Plays); i++ {
			prev := pile.Plays[i-1]
			if !pile.Plays[i].Combo.Chops(prev.Combo, g.Config.Variant) {
				continue
			}
			chop[i] = true
			switch {
			// A payment-free chop is one that chops a finisher's last play.
			case prev.Finished:
				free[i] = true
			// Catch the first chop in a chain or a chop after a payment-free
			// chop: the chain starts here, depth 1.
			case !chop[i-1] || free[i-1]:
				// The chain settles at the last chop of another player's play, so
				// the first chop in a chain does not settle yet.
				settling = append(settling, -1)
				chain[i] = len(settling)
				depth[i] = 1
			// Otherwise, the chain continues and the depth increments.
			default:
				chain[i] = chain[i-1]
				depth[i] = depth[i-1] + 1
			}
			// A chain settles at the last chop of another player's play: the
			// last chop in a chain is the one that pays, and it pays the
			// depth of the chain times the stake.
			if chain[i] > 0 && prev.Seat != pile.Plays[i].Seat {
				settling[chain[i]-1] = i
			}
		}
		for _, i := range settling {
			// The settling index is the last chop in a chain that settles at
			// play i.
			if i < 0 {
				continue
			}
			want = append(want, Transfer{
				Hand:   g.Hand,
				From:   pile.Plays[i-1].Seat,
				To:     pile.Plays[i].Seat,
				Amount: depth[i] * g.Config.Stake,
				Reason: TransferChop,
			})
		}
	}
	var got []Transfer
	for _, tr := range g.Transfers() {
		if tr.Hand == g.Hand && tr.Reason == TransferChop {
			got = append(got, tr)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("got chop transfers %v, want %v recomputed from the pile history",
			got, want)
	}
}

// assertWinnerConsistent verifies that Winner agrees with the balances
// of a finished match: the unique highest balance wins outright, and a
// tied highest balance draws.
func assertWinnerConsistent(t *testing.T, g *Game) {
	t.Helper()
	balances := g.Balances()
	seat, ok, err := g.Winner()
	if err != nil {
		t.Fatalf("Winner: %v", err)
	}
	best := slices.Max(balances)
	ties := 0
	for _, b := range balances {
		if b == best {
			ties++
		}
	}
	if ties > 1 {
		if ok {
			t.Errorf("got outright winner %d, want a draw (balances %v)", seat, balances)
		}
		return
	}
	if !ok || balances[seat] != best {
		t.Errorf("got (%d, %v), want the unique highest balance of %v", seat, ok, balances)
	}
}

// driveRandomHand plays out a hand with a random policy: each eligible
// seat declares its automatic win with probability ½, each turn plays a
// uniformly random legal combination or, when legal, passes with
// probability ¼, and every pile pause is resolved immediately. It drives
// g to PhaseScore and fails if the hand does not end within a generous
// action cap.
func driveRandomHand(t *testing.T, g *Game, rng *rand.Rand) {
	t.Helper()
	balances := g.Balances()
	transfers := g.Transfers()
	// The ledger must not move while the hand is in play: settlement
	// happens only at the transition into PhaseScore.
	assertLedgerFrozen := func() {
		t.Helper()
		if g.Phase != PhasePlay {
			return
		}
		if got := g.Balances(); !slices.Equal(got, balances) {
			t.Errorf("got balances %v during play, want %v (settlement waits for PhaseScore)",
				got, balances)
		}
		if got := g.Transfers(); !slices.Equal(got, transfers) {
			t.Errorf("got transfers %v during play, want %v (settlement waits for PhaseScore)",
				got, transfers)
		}
	}
	for actions := 0; ; actions++ {
		if actions > 1000 {
			t.Fatal("hand did not end within 1000 actions")
		}
		switch {
		case g.Phase == PhaseScore:
			return
		case g.Phase != PhasePlay:
			t.Fatalf("driver reached phase %d before scoring", g.Phase)
		case g.AutoWinWindowOpen:
			for seat := range g.Hands {
				ok, err := g.CanDeclareAutoWin(Seat(seat))
				if err != nil {
					t.Fatalf("CanDeclareAutoWin(%d): %v", seat, err)
				}
				if ok && rng.IntN(2) == 0 {
					if err := g.DeclareAutoWin(Seat(seat)); err != nil {
						t.Fatalf("DeclareAutoWin(%d): %v", seat, err)
					}
				}
			}
			if err := g.StartPlay(); err != nil {
				t.Fatalf("StartPlay: %v", err)
			}
			assertLedgerFrozen()
		case g.PilePendingResolution:
			if err := g.ResolvePile(); err != nil {
				t.Fatalf("ResolvePile: %v", err)
			}
			assertLedgerFrozen()
		default:
			seat := g.Turn
			moves, err := g.LegalMoves(seat)
			if err != nil {
				t.Fatalf("LegalMoves(%d): %v", seat, err)
			}
			canPass, err := g.CanPass(seat)
			if err != nil {
				t.Fatalf("CanPass(%d): %v", seat, err)
			}
			switch {
			case len(moves) == 0:
				if !canPass {
					t.Fatalf("seat %d has no legal play and cannot pass", seat)
				}
				mustPass(t, g, seat)
			case canPass && rng.IntN(4) == 0:
				mustPass(t, g, seat)
			default:
				mustPlay(t, g, seat, moves[rng.IntN(len(moves))].Cards...)
			}
			assertLedgerFrozen()
		}
	}
}
