package tienlen

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// gameSummary is a comparable snapshot of observable game state, used to
// verify that rejected actions change nothing.
type gameSummary struct {
	// phase is the current match phase.
	phase Phase
	// turn is the seat to act.
	turn Seat
	// hands is the printed contents of every live hand, in seat order.
	hands string
	// open reports whether a pile is live.
	open bool
	// top is the printed pile top.
	top string
	// plays is the number of plays on the current pile.
	plays int
	// locked is the printed pile lockout flags.
	locked string
	// cardsPlayed is the printed per-seat play counts.
	cardsPlayed string
	// historyLen is the number of closed piles.
	historyLen int
	// places is the number of places decided.
	places int
	// events is the event log length.
	events int
	// selfBeat reports whether the pile holder must self-beat.
	selfBeat bool
	// pending reports whether the pile awaits resolution.
	pending bool
	// windowOpen reports whether the declaration window is open.
	windowOpen bool
}

// TestNewPanicsOnNilRNG verifies that New panics without a random source.
func TestNewPanicsOnNilRNG(t *testing.T) {
	assertPanics(t, func() { New(nil, testKillerConfig(4)) })
}

// TestNewPanicsOnInvalidConfig verifies that New panics on every invalid
// construction parameter.
func TestNewPanicsOnInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"unknown variant", Config{Variant: Variant(99), NumPlayers: 4, Stake: 2}},
		{"too few players", Config{Variant: Killer, NumPlayers: 1, Stake: 2}},
		{"too many players", Config{Variant: Killer, NumPlayers: 5, Stake: 2}},
		{"zero stake", Config{Variant: Killer, NumPlayers: 4, Stake: 0}},
		{"negative stake", Config{Variant: Killer, NumPlayers: 4, Stake: -2}},
		{"odd stake", Config{Variant: Killer, NumPlayers: 4, Stake: 3}},
		{"negative match length", Config{
			Variant: Killer, NumPlayers: 4, Stake: 2, MatchLength: -1,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPanics(t, func() { New(testRNG(), tt.cfg) })
		})
	}
}

// TestNewDefaultsMatchLength verifies that a zero MatchLength selects the
// default.
func TestNewDefaultsMatchLength(t *testing.T) {
	g := New(testRNG(), Config{Variant: Killer, NumPlayers: 4, Stake: 2})
	if g.Config.MatchLength != DefaultMatchLength {
		t.Errorf("got MatchLength %d, want %d", g.Config.MatchLength, DefaultMatchLength)
	}
	if g.Phase != PhaseDeal {
		t.Errorf("got phase %d, want PhaseDeal", g.Phase)
	}
}

// TestDeal verifies the deal for each legal player count: hand sizes,
// undealt cards, sorted hands, the open declaration window, and the
// hand-started event.
func TestDeal(t *testing.T) {
	tests := []struct {
		numPlayers int
		undealt    int
	}{
		{2, 26},
		{3, 13},
		{4, 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d players", tt.numPlayers), func(t *testing.T) {
			g := New(testRNG(), testKillerConfig(tt.numPlayers))
			if err := g.Deal(); err != nil {
				t.Fatalf("Deal: %v", err)
			}
			seen := map[cardcore.Card]bool{}
			for i, h := range g.Hands {
				if got := h.Len(); got != HandSize {
					t.Errorf("seat %d hand size: got %d, want %d", i, got, HandSize)
				}
				if !slices.IsSortedFunc(h.Cards, compareCards) {
					t.Errorf("seat %d hand not sorted by Tiến Lên order", i)
				}
				for _, c := range h.Cards {
					if seen[c] {
						t.Errorf("card %v dealt twice", c)
					}
					seen[c] = true
				}
			}
			if got := len(g.Undealt); got != tt.undealt {
				t.Errorf("undealt: got %d, want %d", got, tt.undealt)
			}
			for _, c := range g.Undealt {
				if seen[c] {
					t.Errorf("card %v both dealt and undealt", c)
				}
				seen[c] = true
			}
			if got := len(seen); got != cardcore.DeckSize {
				t.Errorf("distinct cards: got %d, want %d", got, cardcore.DeckSize)
			}
			if g.Phase != PhasePlay {
				t.Errorf("got phase %d, want PhasePlay", g.Phase)
			}
			if !g.AutoWinWindowOpen {
				t.Error("declaration window not open after deal")
			}
			for i, active := range g.Active {
				if !active {
					t.Errorf("seat %d not active after deal", i)
				}
			}
			starts := eventsOfType[HandStartedEvent](g.Events())
			if len(starts) != 1 || starts[0].Hand != 0 {
				t.Errorf("got %v, want one HandStartedEvent for hand 0", starts)
			}
		})
	}
}

// TestDealDeterministicForSeed verifies that equal seeds produce equal
// deals.
func TestDealDeterministicForSeed(t *testing.T) {
	deal := func() [][]cardcore.Card {
		g := New(rand.New(rand.NewPCG(42, 7)), testKillerConfig(4))
		if err := g.Deal(); err != nil {
			t.Fatalf("Deal: %v", err)
		}
		hands := make([][]cardcore.Card, len(g.Hands))
		for i, h := range g.Hands {
			hands[i] = slices.Clone(h.Cards)
		}
		return hands
	}
	first, second := deal(), deal()
	for i := range first {
		if !slices.Equal(first[i], second[i]) {
			t.Fatalf("seat %d hands differ for equal seeds", i)
		}
	}
}

// TestDealSometimesLeavesLowestCardUndealt verifies that two-player deals
// sometimes leave the 3♠ out of play (a distribution property, so it
// uses the tries pattern).
func TestDealSometimesLeavesLowestCardUndealt(t *testing.T) {
	threeOfSpades := c(rThree, sSpades)
	found := false
	for i := range 200 {
		g := New(rand.New(rand.NewPCG(uint64(i), 1)), testKillerConfig(2))
		if err := g.Deal(); err != nil {
			t.Fatalf("Deal: %v", err)
		}
		if slices.Contains(g.Undealt, threeOfSpades) {
			found = true
			break
		}
	}
	if !found {
		t.Error("3♠ was never undealt in 200 two-player deals")
	}
}

// TestDealOpeningLeadVaries verifies that the opening-lead seat varies
// across deals (a distribution property, so it uses the tries pattern).
func TestDealOpeningLeadVaries(t *testing.T) {
	seen := map[Seat]bool{}
	for i := range 200 {
		g := New(rand.New(rand.NewPCG(uint64(i), 2)), testKillerConfig(3))
		if err := g.Deal(); err != nil {
			t.Fatalf("Deal: %v", err)
		}
		if err := g.StartPlay(); err != nil {
			t.Fatalf("StartPlay: %v", err)
		}
		seen[g.Turn] = true
	}
	if len(seen) < 2 {
		t.Errorf("opening lead seat never varied in 200 deals: %v", seen)
	}
}

// TestDealResetsState verifies that a fresh deal clears all per-hand
// state from the previous hand.
func TestDealResetsState(t *testing.T) {
	g := New(testRNG(), Config{Variant: Killer, NumPlayers: 4, Stake: 2, MatchLength: 2})
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal: %v", err)
	}
	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}
	mustPlay(t, g, g.Turn, g.OpeningCard)

	// Force the hand to end, then deal the next one.
	g.Phase = PhaseScore
	if err := g.EndHand(); err != nil {
		t.Fatalf("EndHand: %v", err)
	}
	if g.Phase != PhaseDeal {
		t.Fatalf("got phase %d, want PhaseDeal", g.Phase)
	}
	if err := g.Deal(); err != nil {
		t.Fatalf("second Deal: %v", err)
	}
	if g.Hand != 1 {
		t.Errorf("got hand %d, want 1", g.Hand)
	}
	if len(g.Places) != 0 {
		t.Errorf("places not reset: %v", g.Places)
	}
	if len(g.PileHistory) != 0 {
		t.Errorf("pile history not reset: %d piles", len(g.PileHistory))
	}
	if g.Pile.Open {
		t.Error("pile still open after deal")
	}
	if g.nextPileID != 0 {
		t.Errorf("pile serial not reset: %d", g.nextPileID)
	}
	if g.PilePendingResolution || g.SelfBeatContinuation {
		t.Error("resolution or self-beat flag not reset")
	}
	if !slices.Equal(g.CardsPlayed, make([]int, 4)) {
		t.Errorf("cards played not reset: %v", g.CardsPlayed)
	}
}

// TestKillerCanDeclareAutoWin verifies the eligibility query against
// Killer's automatic-win set.
func TestKillerCanDeclareAutoWin(t *testing.T) {
	hands := [][]cardcore.Card{
		{c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts)},
		{c(rThree, sSpades)},
	}
	g := newKillerGameWindowOpen(t, hands...)
	ok, err := g.CanDeclareAutoWin(0)
	if err != nil || !ok {
		t.Errorf("CanDeclareAutoWin(0) = %v, %v; want true, nil", ok, err)
	}
	ok, err = g.CanDeclareAutoWin(1)
	if err != nil || ok {
		t.Errorf("CanDeclareAutoWin(1) = %v, %v; want false, nil", ok, err)
	}
	ok, err = g.CanDeclareAutoWin(9)
	if err == nil || ok {
		t.Errorf("CanDeclareAutoWin(9) = %v, %v; want false, error", ok, err)
	}
}

// TestKillerDeclareAutoWin verifies declaration recording and its
// failures under Killer's automatic-win set.
func TestKillerDeclareAutoWin(t *testing.T) {
	newGame := func() *Game {
		return newKillerGameWindowOpen(t,
			[]cardcore.Card{c(rTwo, sSpades), c(rTwo, sClubs),
				c(rTwo, sDiamonds), c(rTwo, sHearts)},
			[]cardcore.Card{c(rThree, sSpades)},
			[]cardcore.Card{c(rFour, sSpades)},
		)
	}

	g := newGame()
	if err := g.DeclareAutoWin(1); !errors.Is(err, ErrIllegalMove) {
		t.Errorf("declare without a win: got %v, want ErrIllegalMove", err)
	}
	if err := g.DeclareAutoWin(0); err != nil {
		t.Fatalf("DeclareAutoWin: %v", err)
	}
	if err := g.DeclareAutoWin(0); !errors.Is(err, ErrIllegalMove) {
		t.Errorf("double declaration: got %v, want ErrIllegalMove", err)
	}
	declared := eventsOfType[AutoWinDeclaredEvent](g.Events())
	if len(declared) != 1 || declared[0].Seat != 0 || declared[0].Kind != AutoWinQuadTwos {
		t.Errorf("got declared events %v, want one quad-2s declaration by seat 0", declared)
	}

	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}
	if err := g.DeclareAutoWin(1); !errors.Is(err, ErrIllegalMove) {
		t.Errorf("declare after window closed: got %v, want ErrIllegalMove", err)
	}
}

// TestKillerStartPlay verifies that closing the window with no
// declarations sets Killer's opening lead: the lowest card in play.
func TestKillerStartPlay(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades)},
		[]cardcore.Card{c(rThree, sSpades), c(rSix, sSpades)},
	)
	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}
	if g.AutoWinWindowOpen {
		t.Error("window still open after StartPlay")
	}
	if g.Turn != 1 {
		t.Errorf("got turn %d, want 1 (holder of 3♠)", g.Turn)
	}
	if !g.OpeningLeadRequired || !g.OpeningCard.Equal(c(rThree, sSpades)) {
		t.Errorf("got opening card %v (required %v), want 3♠ required",
			g.OpeningCard, g.OpeningLeadRequired)
	}
	if err := g.StartPlay(); !errors.Is(err, ErrIllegalMove) {
		t.Errorf("second StartPlay: got %v, want ErrIllegalMove", err)
	}
}

// TestPlayBeforeStartPlay verifies that no play is accepted while the
// declaration window is open.
func TestPlayBeforeStartPlay(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{c(rThree, sSpades)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sSpades))
}

// TestPlayRejectsUnclassifiableAndUnownedCards verifies basic play
// validation.
func TestPlayRejectsUnclassifiableAndUnownedCards(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds)},
		[]cardcore.Card{c(rSix, sSpades)},
	)
	// Duplicate cards are rejected by Classify.
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sSpades), c(rThree, sSpades))
	// Two different ranks form no combination.
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rFour, sClubs), c(rFive, sDiamonds))
	// A card the seat does not hold.
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rKing, sSpades))
}

// TestEndHand verifies the hand-to-hand and match-end transitions.
func TestEndHand(t *testing.T) {
	g := New(testRNG(), Config{Variant: Killer, NumPlayers: 2, Stake: 2, MatchLength: 2})
	if err := g.EndHand(); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("EndHand in PhaseDeal: got %v, want ErrWrongPhase", err)
	}
	g.Phase = PhaseScore
	if err := g.EndHand(); err != nil {
		t.Fatalf("EndHand: %v", err)
	}
	if g.Phase != PhaseDeal || g.Hand != 1 {
		t.Errorf("got phase %d hand %d, want PhaseDeal hand 1", g.Phase, g.Hand)
	}
	g.Phase = PhaseScore
	if err := g.EndHand(); err != nil {
		t.Fatalf("EndHand: %v", err)
	}
	if g.Phase != PhaseEnd {
		t.Errorf("got phase %d, want PhaseEnd", g.Phase)
	}
	if err := g.Deal(); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("Deal after match end: got %v, want ErrWrongPhase", err)
	}
}

// TestClone verifies that clones are independent of the original except
// for the shared random source.
func TestClone(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rSeven, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rEight, sHearts)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))

	clone := g.Clone()
	if clone.rng != g.rng {
		t.Error("clone does not share the random source")
	}

	// Mutate every deep-copied structure of the clone.
	clone.Hands[1].Remove(c(rFour, sSpades))
	clone.Pile.Plays[0].Combo.Cards[0] = c(rTwo, sHearts)
	clone.Pile.Top.Cards[0] = c(rTwo, sHearts)
	clone.Pile.Locked[1] = true
	clone.Active[1] = false
	clone.Places = append(clone.Places, 1)
	clone.CardsPlayed[0] = 99
	clone.events = append(clone.events, HandStartedEvent{Hand: 9})

	if !g.Hands[1].Contains(c(rFour, sSpades)) || !g.Hands[1].Contains(c(rEight, sHearts)) {
		t.Error("original hand changed through clone")
	}
	if !g.Pile.Plays[0].Combo.Cards[0].Equal(c(rThree, sSpades)) {
		t.Error("original pile play changed through clone")
	}
	if g.Pile.Locked[1] || !g.Active[1] {
		t.Error("original lockout or active state changed through clone")
	}
	if len(g.Places) != 0 || g.CardsPlayed[0] != 1 || len(g.Events()) != 2 {
		t.Error("original places, counts, or events changed through clone")
	}
}

// TestKillerDetectAutoWin verifies automatic-win detection over fixture
// hands: Killer's set is a Killer of 2s and any six pairs.
func TestKillerDetectAutoWin(t *testing.T) {
	tests := []struct {
		name     string
		cards    []cardcore.Card
		wantOK   bool
		wantKind AutoWinKind
		wantTop  cardcore.Card
	}{
		{
			name: "quad of 2s",
			cards: []cardcore.Card{
				c(rThree, sSpades),
				c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
			},
			wantOK:   true,
			wantKind: AutoWinQuadTwos,
			wantTop:  c(rTwo, sHearts),
		},
		{
			name: "six pairs",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades), c(rSeven, sClubs),
				c(rNine, sSpades), c(rNine, sHearts),
				c(rTen, sSpades),
			},
			wantOK:   true,
			wantKind: AutoWinSixPairs,
			wantTop:  c(rNine, sHearts),
		},
		{
			name: "five pairs and a triple",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades), c(rSeven, sClubs),
				c(rEight, sSpades), c(rEight, sClubs), c(rEight, sDiamonds),
			},
			wantOK:   true,
			wantKind: AutoWinSixPairs,
			wantTop:  c(rEight, sDiamonds),
		},
		{
			name: "quad of 3s counts as two pairs",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds), c(rThree, sHearts),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades), c(rSeven, sClubs),
				c(rNine, sSpades),
			},
			wantOK:   true,
			wantKind: AutoWinSixPairs,
			wantTop:  c(rSeven, sClubs),
		},
		{
			name: "five pairs and three singles is not enough",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades), c(rSeven, sClubs),
				c(rEight, sSpades), c(rNine, sSpades), c(rTen, sSpades),
			},
			wantOK: false,
		},
		{
			name: "quad of 2s beats six pairs to one candidate",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades),
				c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
			},
			wantOK:   true,
			wantKind: AutoWinQuadTwos,
			wantTop:  c(rTwo, sHearts),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aw, ok := detectAutoWin(0, tt.cards)
			if ok != tt.wantOK {
				t.Fatalf("got ok %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if aw.Kind != tt.wantKind {
				t.Errorf("got kind %d, want %d", aw.Kind, tt.wantKind)
			}
			if !aw.Top.Equal(tt.wantTop) {
				t.Errorf("got top %v, want %v", aw.Top, tt.wantTop)
			}
		})
	}
}

// TestKillerAutoWinPriorityOrder verifies that candidates are stored in
// declaration priority order: a Killer of 2s first, then six-pair hands
// by top pair.
func TestKillerAutoWinPriorityOrder(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{ // six pairs, top pair 8s
			c(rThree, sClubs), c(rThree, sDiamonds),
			c(rFour, sSpades), c(rFour, sClubs),
			c(rFive, sSpades), c(rFive, sClubs),
			c(rSix, sSpades), c(rSix, sClubs),
			c(rSeven, sSpades), c(rSeven, sClubs),
			c(rEight, sSpades), c(rEight, sHearts),
			c(rNine, sSpades),
		},
		[]cardcore.Card{ // quad of 2s
			c(rNine, sClubs),
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
		},
		[]cardcore.Card{ // six pairs, top pair aces
			c(rThree, sHearts), c(rFour, sDiamonds), c(rFour, sHearts),
			c(rFive, sDiamonds), c(rFive, sHearts),
			c(rSix, sDiamonds), c(rSix, sHearts),
			c(rSeven, sDiamonds), c(rSeven, sHearts),
			c(rNine, sDiamonds), c(rNine, sHearts),
			c(rAce, sSpades), c(rAce, sHearts),
		},
		[]cardcore.Card{c(rJack, sSpades), c(rQueen, sSpades)},
	)
	if len(g.AutoWins) != 3 {
		t.Fatalf("got %d candidates, want 3", len(g.AutoWins))
	}
	wantOrder := []Seat{1, 2, 0} // quad of 2s, aces-high six pairs, 8s-high six pairs
	for i, aw := range g.AutoWins {
		if aw.Seat != wantOrder[i] {
			t.Errorf("candidate %d: got seat %d, want %d", i, aw.Seat, wantOrder[i])
		}
	}
}

// TestPlayEmptySliceIsPass verifies that an empty, non-nil card slice
// passes, exactly like the nil slice mustPass uses.
func TestPlayEmptySliceIsPass(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rFour, sClubs)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	if err := g.Play(0, []cardcore.Card{}); err != nil {
		t.Fatalf("Play(0, empty slice): %v", err)
	}
	if !g.Pile.Locked[0] {
		t.Error("got seat 0 unlocked, want locked after passing")
	}
	assertCardIntegrity(t, g)
	assertTurnValid(t, g)
}

// TestKillerSelfBeatAndAtomicNewPileIntegration verifies Killer's
// self-beat rule: when every other player passes, the holder must keep
// playing; a play that could not have beaten the previous one closes
// the pile and opens a new one in the same action, unlocking everyone.
func TestKillerSelfBeatAndAtomicNewPileIntegration(t *testing.T) {
	// Round 1: seat 0 leads its 3♠; seat 1 beats it; the rest pass.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades),
			c(rNine, sSpades), c(rNine, sClubs), c(rNine, sDiamonds), c(rNine, sHearts),
			c(rJack, sSpades)},
		[]cardcore.Card{c(rThree, sClubs), c(rThree, sDiamonds)},
		[]cardcore.Card{c(rSix, sClubs), c(rSix, sDiamonds)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 2)
	mustPass(t, g, 3)
	mustPass(t, g, 0)

	// Seat 1 holds the pile with no contest left: it must keep playing,
	// with no resolution pause and no unlocking.
	if g.PilePendingResolution {
		t.Error("pile pending resolution during self-beat")
	}
	if !g.SelfBeatContinuation || g.Turn != 1 {
		t.Errorf("got self-beat %v turn %d, want self-beat with turn 1",
			g.SelfBeatContinuation, g.Turn)
	}
	mustFailPass(t, g, 1, ErrIllegalMove)

	// A compatible play continues the pile; the lockouts persist.
	mustPlay(t, g, 1, c(rFive, sSpades))
	if len(g.Pile.Plays) != 3 {
		t.Fatalf("got %d plays, want 3", len(g.Pile.Plays))
	}
	for seat, locked := range g.Pile.Locked {
		if seat != 1 && !locked {
			t.Errorf("seat %d unlocked before the pile boundary", seat)
		}
	}

	// Round 2: the quad of 9s cannot beat a single 5, so it opens a new
	// pile in the same action, unlocking everyone.
	mustPlay(t, g, 1,
		c(rNine, sSpades), c(rNine, sClubs), c(rNine, sDiamonds), c(rNine, sHearts))
	if len(g.PileHistory) != 1 || g.PileHistory[0].ID != 1 || len(g.PileHistory[0].Plays) != 3 {
		t.Errorf("got pile history of %d piles, want pile 1 with 3 plays", len(g.PileHistory))
	}
	if g.Pile.ID != 2 || len(g.Pile.Plays) != 1 {
		t.Errorf("got pile %d with %d plays, want pile 2 with 1 play",
			g.Pile.ID, len(g.Pile.Plays))
	}
	lead := g.Pile.Plays[0]
	if lead.Chop || lead.Chain != 0 {
		t.Errorf("new-pile opener: got chop %v chain %d, want a plain lead",
			lead.Chop, lead.Chain)
	}
	for seat, locked := range g.Pile.Locked {
		if locked {
			t.Errorf("seat %d still locked after the boundary", seat)
		}
	}
	if g.Turn != 2 {
		t.Errorf("got turn %d, want 2", g.Turn)
	}
	closed := eventsOfType[PileClosedEvent](g.Events())
	if len(closed) != 1 || closed[0].Pile != 1 || closed[0].NextLeader != 1 ||
		!closed[0].HasNextLeader {
		t.Errorf("got closed events %v, want pile 1 closed with seat 1 leading", closed)
	}
	opened := eventsOfType[PileOpenedEvent](g.Events())
	if len(opened) != 2 || opened[1].Pile != 2 || opened[1].Leader != 1 {
		t.Errorf("got opened events %v, want pile 2 opened by seat 1", opened)
	}
}

// TestKillerSelfChopOwnTwosIntegration verifies that a self-beating
// player may chop their own 2 and pair of 2s, and that chop chains are
// numbered independently per pile.
func TestKillerSelfChopOwnTwosIntegration(t *testing.T) {
	// Round 1: seat 1 takes the pile, then self-beats.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades),
			c(rFive, sClubs), c(rFive, sDiamonds),
			c(rSix, sSpades), c(rSix, sClubs),
			c(rSeven, sSpades), c(rSeven, sClubs),
			c(rEight, sSpades), c(rEight, sClubs),
			c(rJack, sSpades), c(rJack, sClubs), c(rJack, sDiamonds), c(rJack, sHearts),
			c(rKing, sHearts),
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds)},
		[]cardcore.Card{c(rThree, sClubs), c(rThree, sDiamonds)},
		[]cardcore.Card{c(rFour, sClubs), c(rFour, sDiamonds)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 2)
	mustPass(t, g, 3)
	mustPass(t, g, 0)

	mustPlay(t, g, 1, c(rTwo, sSpades))
	if g.Pile.Plays[2].Chop {
		t.Error("an ordinary beat of a single is not a chop")
	}
	// The six-card Bomb chops the player's own single 2.
	mustPlay(t, g, 1,
		c(rSix, sSpades), c(rSix, sClubs), c(rSeven, sSpades), c(rSeven, sClubs),
		c(rEight, sSpades), c(rEight, sClubs))
	selfChop := g.Pile.Plays[3]
	if !selfChop.Chop || selfChop.Chain != 1 || selfChop.Depth != 1 {
		t.Errorf("got chop %v chain %d depth %d, want a chop at chain 1 depth 1",
			selfChop.Chop, selfChop.Chain, selfChop.Depth)
	}

	// Round 2: the pair of 5s cannot beat the Bomb, so it opens a new
	// pile; everyone is unlocked, and the turn moves on.
	mustPlay(t, g, 1, c(rFive, sClubs), c(rFive, sDiamonds))
	if g.Pile.ID != 2 {
		t.Fatalf("got pile %d, want pile 2", g.Pile.ID)
	}
	if g.Turn != 2 {
		t.Errorf("got turn %d, want 2", g.Turn)
	}
	mustPass(t, g, 2)
	mustPass(t, g, 3)
	mustPass(t, g, 0)
	mustPlay(t, g, 1, c(rTwo, sClubs), c(rTwo, sDiamonds))
	if g.Pile.Plays[1].Chop {
		t.Error("an ordinary beat of a pair is not a chop")
	}
	// The quad chops the player's own pair of 2s: the new pile's first
	// chain, independent of the old pile's chain.
	mustPlay(t, g, 1,
		c(rJack, sSpades), c(rJack, sClubs), c(rJack, sDiamonds), c(rJack, sHearts))
	pairChop := g.Pile.Plays[2]
	if !pairChop.Chop || pairChop.Chain != 1 || pairChop.Depth != 1 {
		t.Errorf("got chop %v chain %d depth %d, want a chop at chain 1 depth 1",
			pairChop.Chop, pairChop.Chain, pairChop.Depth)
	}
	var chops [][2]int // (pile, chain) pairs of chop events, in order
	for _, e := range eventsOfType[PlayEvent](g.Events()) {
		if e.Chop {
			chops = append(chops, [2]int{e.Pile, e.Chain})
		}
	}
	if len(chops) != 2 || chops[0] != [2]int{1, 1} || chops[1] != [2]int{2, 1} {
		t.Errorf("got chop events %v, want (pile 1, chain 1) and (pile 2, chain 1)", chops)
	}
}

// TestKillerTripleTwosForcesNewPileIntegration verifies that an
// unbeatable top does not wedge the pile: the holder's triple 2s beat a
// lower triple as an ordinary play (un-choppable is not unplayable), and
// the holder's next, incompatible play closes the pile and opens a new
// one in the same action.
func TestKillerTripleTwosForcesNewPileIntegration(t *testing.T) {
	// Round 1: triple 3s are led; seat 1 beats them and self-beats.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds),
			c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rFour, sClubs), c(rFour, sDiamonds),
			c(rSeven, sClubs), c(rSeven, sDiamonds), c(rQueen, sHearts),
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds)},
		[]cardcore.Card{c(rFive, sClubs), c(rFive, sDiamonds)},
		[]cardcore.Card{c(rSix, sClubs), c(rSix, sDiamonds)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds))
	mustPlay(t, g, 1, c(rFour, sSpades), c(rFour, sClubs), c(rFour, sDiamonds))
	mustPass(t, g, 2)
	mustPass(t, g, 3)
	mustPass(t, g, 0)

	mustPlay(t, g, 1, c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds))
	if g.Pile.ID != 1 {
		t.Fatalf("got pile %d, want triple 2s to stay in pile 1", g.Pile.ID)
	}
	// Round 2: no combination beats triple 2s, so any play opens a new
	// pile and unlocks everyone.
	mustPlay(t, g, 1, c(rSeven, sClubs), c(rSeven, sDiamonds))
	if g.Pile.ID != 2 {
		t.Errorf("got pile %d, want pile 2", g.Pile.ID)
	}
	if g.Turn != 2 {
		t.Errorf("got turn %d, want 2", g.Turn)
	}
	for seat, locked := range g.Pile.Locked {
		if locked {
			t.Errorf("seat %d still locked after the boundary", seat)
		}
	}
}

// TestKillerCompatibleSelfBeatFinisherKeepsLockoutIntegration verifies
// that when a self-beating player goes out with a play that could have
// beaten their previous one, the pile stays live, locked-out players
// stay locked out, and the lead passes to the next active player.
func TestKillerCompatibleSelfBeatFinisherKeepsLockoutIntegration(t *testing.T) {
	// Round 1: seat 1 self-beats and goes out on the second play.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades)},
		[]cardcore.Card{c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds),
			c(rSeven, sHearts)},
		[]cardcore.Card{c(rSix, sClubs), c(rSix, sDiamonds)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 2)
	mustPass(t, g, 3)
	mustPass(t, g, 0)
	mustPlay(t, g, 1, c(rFive, sSpades))

	if !slices.Equal(g.Places, []Seat{1}) {
		t.Errorf("got places %v, want [1]", g.Places)
	}
	if g.Active[1] {
		t.Error("seat 1 still active after going out")
	}
	if !g.PilePendingResolution {
		t.Error("finisher's live last play should pause the pile")
	}
	if g.Turn != 1 {
		t.Errorf("got turn %d, want 1 (the last actor stays)", g.Turn)
	}
	// No responses are possible while the pile awaits resolution.
	mustFailPlay(t, g, 2, ErrIllegalMove,
		c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts))

	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}
	if len(g.PileHistory) != 1 {
		t.Errorf("got %d closed piles, want 1", len(g.PileHistory))
	}
	if g.Turn != 2 {
		t.Errorf("got turn %d, want 2 (next active after the finisher)", g.Turn)
	}
	// Round 2: seat 2 leads freely; a bomb-class lead earns nothing.
	mustPlay(t, g, 2,
		c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts))
	if g.Pile.Plays[0].Chop || g.Pile.Plays[0].Chain != 0 {
		t.Error("a bomb-class combination played to open a pile must not be a chop")
	}
}

// TestKillerIncompatibleSelfBeatFinisherUnlocksNewPileIntegration verifies
// that when a self-beating player goes out with a play that could not
// have beaten their previous one, the play opens a new pile, everyone is
// unlocked, and the new pile is answered normally.
func TestKillerIncompatibleSelfBeatFinisherUnlocksNewPileIntegration(t *testing.T) {
	// Round 1: seat 1 self-beats and goes out on the boundary play.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sClubs), c(rFour, sDiamonds), c(rNine, sSpades)},
		[]cardcore.Card{c(rFive, sClubs), c(rFive, sDiamonds), c(rKing, sClubs)},
		[]cardcore.Card{c(rSix, sClubs), c(rSix, sDiamonds), c(rQueen, sHearts)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rNine, sSpades))
	mustPass(t, g, 2)
	mustPass(t, g, 3)
	mustPass(t, g, 0)

	// Round 2: the pair of 4s cannot beat a single 9, so it opens a new
	// pile; seat 1's place is fixed and everyone is unlocked.
	mustPlay(t, g, 1, c(rFour, sClubs), c(rFour, sDiamonds))
	if !slices.Equal(g.Places, []Seat{1}) {
		t.Errorf("got places %v, want [1]", g.Places)
	}
	if g.PilePendingResolution {
		t.Error("a boundary play must not pause: the new pile is live")
	}
	if g.Pile.ID != 2 || g.Pile.Holder != 1 {
		t.Errorf("got pile %d holder %d, want pile 2 held by seat 1", g.Pile.ID, g.Pile.Holder)
	}
	for seat, locked := range g.Pile.Locked {
		if locked {
			t.Errorf("seat %d still locked after the boundary", seat)
		}
	}
	if g.Turn != 2 {
		t.Errorf("got turn %d, want 2", g.Turn)
	}

	// The finisher's last play is live: the remaining players answer it
	// in turn order.
	mustPlay(t, g, 2, c(rFive, sClubs), c(rFive, sDiamonds))
	mustPlay(t, g, 3, c(rSix, sClubs), c(rSix, sDiamonds))
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sHearts))
	mustPass(t, g, 0)
	if g.Turn != 2 {
		t.Errorf("got turn %d, want 2", g.Turn)
	}
	if !slices.Equal(g.Places, []Seat{1}) {
		t.Errorf("beating the finisher changed places: %v", g.Places)
	}
}

// TestFinisherTurnOrderAndStayOutIntegration verifies that a finisher's
// last play is answered in strict turn order, that locked and finished
// seats are skipped, and that a finisher never re-enters play.
func TestFinisherTurnOrderAndStayOutIntegration(t *testing.T) {
	// Round 1: seat 3 goes out on its 9♠.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFive, sDiamonds), c(rTen, sSpades)},
		[]cardcore.Card{c(rEight, sSpades), c(rJack, sSpades)},
		[]cardcore.Card{c(rKing, sSpades)},
		[]cardcore.Card{c(rNine, sSpades)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rEight, sSpades))
	mustPass(t, g, 2)
	mustPlay(t, g, 3, c(rNine, sSpades))
	if !slices.Equal(g.Places, []Seat{3}) {
		t.Fatalf("got places %v, want [3]", g.Places)
	}
	if g.Turn != 0 {
		t.Fatalf("got turn %d, want 0", g.Turn)
	}

	// Seat 2 passed earlier and is locked out; it is also not its turn.
	mustFailPlay(t, g, 2, ErrOutOfTurn, c(rKing, sSpades))
	// Seat 0, active and unlocked, may beat the finisher's play.
	mustPlay(t, g, 0, c(rTen, sSpades))
	mustPlay(t, g, 1, c(rJack, sSpades)) // seat 1 goes out in second place
	// The turn skips the locked seat 2 and the finished seat 3.
	if g.Turn != 0 {
		t.Errorf("got turn %d, want 0 (seats 2 and 3 are skipped)", g.Turn)
	}
	mustFailPass(t, g, 3, ErrOutOfTurn)
	if !slices.Equal(g.Places, []Seat{3, 1}) || g.Active[3] {
		t.Errorf("got places %v active[3] %v, want places [3 1] with seat 3 out",
			g.Places, g.Active[3])
	}
}

// TestProtectedFinisherChopStartsFreshChainIntegration verifies that
// chopping a finisher's last play earns nothing but continuation, and
// that a chop of that payment-free chop starts a new payment chain.
func TestProtectedFinisherChopStartsFreshChainIntegration(t *testing.T) {
	// Round 1: seat 3 goes out on its 2♦. Seats 1 and 2 answer in turn;
	// neither has passed, so both are eligible to chop.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFour, sSpades)},
		[]cardcore.Card{c(rFive, sSpades),
			c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
			c(rKing, sDiamonds)},
		[]cardcore.Card{c(rSix, sSpades),
			c(rEight, sSpades), c(rEight, sClubs),
			c(rNine, sSpades), c(rNine, sClubs),
			c(rTen, sSpades), c(rTen, sClubs),
			c(rJack, sSpades), c(rJack, sClubs)},
		[]cardcore.Card{c(rTwo, sDiamonds)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFive, sSpades))
	mustPlay(t, g, 2, c(rSix, sSpades))
	mustPlay(t, g, 3, c(rTwo, sDiamonds))
	mustPass(t, g, 0)

	// Seat 1's quad chops the finisher's 2♦: continuation, no payment.
	mustPlay(t, g, 1,
		c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts))
	protected := g.Pile.Plays[len(g.Pile.Plays)-1]
	if !protected.Chop || !protected.PaymentFree || protected.Chain != 0 || protected.Depth != 0 {
		t.Errorf("got chop %v payment-free %v chain %d depth %d, want a payment-free chop",
			protected.Chop, protected.PaymentFree, protected.Chain, protected.Depth)
	}

	// Seat 2's Bomb chops the quad: the first payable chop of a new
	// chain.
	mustPlay(t, g, 2,
		c(rEight, sSpades), c(rEight, sClubs),
		c(rNine, sSpades), c(rNine, sClubs),
		c(rTen, sSpades), c(rTen, sClubs),
		c(rJack, sSpades), c(rJack, sClubs))
	stacked := g.Pile.Plays[len(g.Pile.Plays)-1]
	if !stacked.Chop || stacked.PaymentFree || stacked.Chain != 1 || stacked.Depth != 1 {
		t.Errorf("got chop %v payment-free %v chain %d depth %d, want chain 1 depth 1",
			stacked.Chop, stacked.PaymentFree, stacked.Chain, stacked.Depth)
	}
}

// TestFinisherTripleTwosIntegration verifies that a finisher going out
// on triple 2s leaves nothing to beat: the pile stands and the lead is
// inherited.
func TestFinisherTripleTwosIntegration(t *testing.T) {
	// Round 1: seat 3 goes out on triple 2s.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds),
			c(rThree, sHearts)},
		[]cardcore.Card{c(rAce, sSpades), c(rAce, sClubs), c(rAce, sDiamonds),
			c(rAce, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rFour, sClubs),
			c(rFive, sSpades), c(rFive, sClubs),
			c(rSix, sSpades), c(rSix, sClubs),
			c(rSeven, sSpades), c(rSeven, sClubs),
			c(rEight, sSpades), c(rEight, sClubs),
			c(rNine, sSpades), c(rNine, sClubs)},
		[]cardcore.Card{c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds))
	// A quad cannot chop a triple, nor can a twelve-card Bomb.
	mustFailPlay(t, g, 1, ErrIllegalMove,
		c(rAce, sSpades), c(rAce, sClubs), c(rAce, sDiamonds), c(rAce, sHearts))
	mustPass(t, g, 1)
	mustFailPlay(t, g, 2, ErrIllegalMove,
		c(rFour, sSpades), c(rFour, sClubs), c(rFive, sSpades), c(rFive, sClubs),
		c(rSix, sSpades), c(rSix, sClubs), c(rSeven, sSpades), c(rSeven, sClubs),
		c(rEight, sSpades), c(rEight, sClubs), c(rNine, sSpades), c(rNine, sClubs))
	mustPass(t, g, 2)
	mustPlay(t, g, 3, c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds))

	// Nothing beats triple 2s, not even for the one still-unlocked
	// player.
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sHearts))
	mustPass(t, g, 0)
	// Everyone else passed or went out: the finisher's play stands.
	if !g.PilePendingResolution {
		t.Fatal("the finisher's unbeatable play should pause the pile")
	}
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}
	if g.Turn != 0 {
		t.Errorf("got turn %d, want 0 (next active after the finisher)", g.Turn)
	}
	// Round 2: seat 0 leads freely.
	mustPlay(t, g, 0, c(rThree, sHearts))
}

// TestKillerTwoPlayerHandEndPauseIntegration verifies Killer's
// two-player self-beat dynamics and the terminal pause before the hand
// scores.
func TestKillerTwoPlayerHandEndPauseIntegration(t *testing.T) {
	// Round 1: seat 1 takes the pile and self-beats after one pass.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts), c(rFive, sHearts)},
		[]cardcore.Card{c(rFour, sSpades), c(rFour, sClubs), c(rSix, sHearts)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	mustPlay(t, g, 1, c(rFour, sSpades))
	mustPass(t, g, 0)
	if !g.SelfBeatContinuation || g.Turn != 1 {
		t.Fatalf("got self-beat %v turn %d, want self-beat with turn 1",
			g.SelfBeatContinuation, g.Turn)
	}
	mustPlay(t, g, 1, c(rFour, sClubs))

	// Seat 1 goes out: seat 0 is last, and the pile pauses so the
	// terminal state stays observable until ResolvePile.
	mustPlay(t, g, 1, c(rSix, sHearts))
	if !slices.Equal(g.Places, []Seat{1}) {
		t.Errorf("got places %v, want [1]", g.Places)
	}
	if !g.PilePendingResolution {
		t.Fatal("the final play should pause the pile before scoring")
	}
	if g.Turn != 1 {
		t.Errorf("got turn %d, want 1 (the last actor stays)", g.Turn)
	}
	mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sHearts))
	mustFailPass(t, g, 0, ErrIllegalMove)
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}
	if err := g.ResolvePile(); !errors.Is(err, ErrWrongPhase) {
		t.Errorf("second ResolvePile: got %v, want ErrWrongPhase", err)
	}
	if !slices.Equal(g.Places, []Seat{1, 0}) {
		t.Errorf("got places %v, want [1 0]", g.Places)
	}
	if g.Phase != PhaseScore {
		t.Errorf("got phase %d, want PhaseScore", g.Phase)
	}
	if len(g.PileHistory) != 1 {
		t.Errorf("got %d closed piles, want 1", len(g.PileHistory))
	}
	ended := eventsOfType[HandEndedEvent](g.Events())
	if len(ended) != 1 || !slices.Equal(ended[0].Places, []Seat{1, 0}) ||
		!slices.Equal(ended[0].CardsPlayed, []int{1, 3}) {
		t.Errorf("got hand-ended events %v, want one with places [1 0] and cards [1 3]", ended)
	}
	if err := g.EndHand(); err != nil {
		t.Fatalf("EndHand: %v", err)
	}
	if g.Phase != PhaseEnd {
		t.Errorf("got phase %d, want PhaseEnd (match length 1)", g.Phase)
	}
}

// TestKillerBombRecencyAlternationIntegration verifies that chops are
// always judged against the current top: a quad and an eight-card Bomb
// chop each other in both directions, most recent play wins, regardless
// of height.
func TestKillerBombRecencyAlternationIntegration(t *testing.T) {
	// Round 1: seat 0 opens with an eight-card Bomb (out of the blue),
	// and the table alternates quads and Bombs.
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sClubs),
			c(rFour, sSpades), c(rFour, sClubs),
			c(rFive, sSpades), c(rFive, sClubs),
			c(rSix, sSpades), c(rSix, sClubs), c(rJack, sDiamonds)},
		[]cardcore.Card{c(rQueen, sHearts),
			c(rAce, sSpades), c(rAce, sClubs), c(rAce, sDiamonds), c(rAce, sHearts)},
		[]cardcore.Card{c(rFour, sHearts),
			c(rSeven, sSpades), c(rSeven, sClubs),
			c(rEight, sSpades), c(rEight, sClubs),
			c(rNine, sDiamonds), c(rNine, sHearts),
			c(rTen, sSpades), c(rTen, sClubs)},
		[]cardcore.Card{c(rKing, sSpades), c(rKing, sClubs), c(rKing, sDiamonds),
			c(rKing, sHearts), c(rTwo, sSpades)},
	)
	// A bomb-class lead is legal and earns nothing.
	mustPlay(t, g, 0,
		c(rThree, sSpades), c(rThree, sClubs), c(rFour, sSpades), c(rFour, sClubs),
		c(rFive, sSpades), c(rFive, sClubs), c(rSix, sSpades), c(rSix, sClubs))
	if g.Pile.Plays[0].Chop || g.Pile.Plays[0].Chain != 0 {
		t.Error("an out-of-the-blue Bomb must not be a chop")
	}
	// A quad chops the Bomb; the Bomb chops the quad though its top
	// (10♣) is lower than the quad's (A♥); a lower quad then chops the
	// Bomb.
	mustPlay(t, g, 1, c(rAce, sSpades), c(rAce, sClubs), c(rAce, sDiamonds), c(rAce, sHearts))
	mustPlay(t, g, 2,
		c(rSeven, sSpades), c(rSeven, sClubs), c(rEight, sSpades), c(rEight, sClubs),
		c(rNine, sDiamonds), c(rNine, sHearts), c(rTen, sSpades), c(rTen, sClubs))
	mustPlay(t, g, 3,
		c(rKing, sSpades), c(rKing, sClubs), c(rKing, sDiamonds), c(rKing, sHearts))

	if len(g.Pile.Plays) != 4 {
		t.Fatalf("got %d plays, want 4", len(g.Pile.Plays))
	}
	for i := 1; i < len(g.Pile.Plays); i++ {
		play := g.Pile.Plays[i]
		if !play.Chop {
			t.Errorf("play %d: not marked a chop", i)
		}
		if play.Chain != 1 || play.Depth != i {
			t.Errorf("play %d: got chain %d depth %d, want chain 1 depth %d",
				i, play.Chain, play.Depth, i)
		}
	}
	if g.Turn != 0 {
		t.Errorf("got turn %d, want 0", g.Turn)
	}
}

// TestKillerOpeningLeadMustContainLowestCardIntegration verifies
// Killer's opening-lead rule: the first play of a hand must include the
// lowest card in play, as a single or within any combination.
func TestKillerOpeningLeadMustContainLowestCardIntegration(t *testing.T) {
	hands := [][]cardcore.Card{
		{c(rThree, sSpades), c(rThree, sClubs), c(rFour, sDiamonds), c(rFive, sHearts),
			c(rSix, sClubs)},
		{c(rSeven, sClubs)},
		{c(rEight, sClubs)},
		{c(rNine, sClubs)},
	}

	t.Run("as a pair", func(t *testing.T) {
		g := newKillerGame(t, hands...)
		mustPlay(t, g, 0, c(rThree, sSpades), c(rThree, sClubs))
	})
	t.Run("within a straight", func(t *testing.T) {
		g := newKillerGame(t, hands...)
		mustPlay(t, g, 0, c(rThree, sSpades), c(rFour, sDiamonds), c(rFive, sHearts))
	})
	t.Run("omitted", func(t *testing.T) {
		g := newKillerGame(t, hands...)
		mustFailPlay(t, g, 0, ErrIllegalMove,
			c(rFour, sDiamonds), c(rFive, sHearts), c(rSix, sClubs))
	})
}

// TestKillerLowestDealtCardLeadsIntegration verifies that under Killer
// rules, when the 3♠ is out of play, the lowest card actually dealt
// leads the hand.
func TestKillerLowestDealtCardLeadsIntegration(t *testing.T) {
	t.Run("two players", func(t *testing.T) {
		// The 3♠ and 3♣ are out of play; the 3♦ is the lowest card dealt.
		g := newKillerGame(t,
			[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades)},
			[]cardcore.Card{c(rThree, sDiamonds), c(rSix, sSpades)},
		)
		if g.Turn != 1 {
			t.Fatalf("got turn %d, want 1 (holder of 3♦)", g.Turn)
		}
		mustFailPlay(t, g, 1, ErrIllegalMove, c(rSix, sSpades))
		mustPlay(t, g, 1, c(rThree, sDiamonds))
	})
	t.Run("three players", func(t *testing.T) {
		// The 3♠ is out of play; the 3♣ is the lowest card dealt.
		g := newKillerGame(t,
			[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades)},
			[]cardcore.Card{c(rFour, sHearts), c(rSix, sSpades)},
			[]cardcore.Card{c(rThree, sClubs), c(rSeven, sSpades)},
		)
		if g.Turn != 2 {
			t.Fatalf("got turn %d, want 2 (holder of 3♣)", g.Turn)
		}
		mustPlay(t, g, 2, c(rThree, sClubs))
	})
}

// TestKillerAutoWinDeclareDeclineAndCloseIntegration verifies that
// Killer's automatic wins are optional, that declaration removes the
// winner's hand from play and recomputes the lead, and that the window
// closes at StartPlay.
func TestKillerAutoWinDeclareDeclineAndCloseIntegration(t *testing.T) {
	newGame := func() *Game {
		return newKillerGameWindowOpen(t,
			[]cardcore.Card{c(rThree, sSpades), c(rNine, sDiamonds),
				c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts)},
			[]cardcore.Card{c(rThree, sClubs), c(rSeven, sDiamonds)},
			[]cardcore.Card{c(rFour, sSpades), c(rNine, sSpades)},
			[]cardcore.Card{c(rFive, sSpades), c(rSix, sSpades)},
		)
	}

	t.Run("declare", func(t *testing.T) {
		g := newGame()
		if err := g.DeclareAutoWin(0); err != nil {
			t.Fatalf("DeclareAutoWin: %v", err)
		}
		if err := g.StartPlay(); err != nil {
			t.Fatalf("StartPlay: %v", err)
		}
		if !slices.Equal(g.Places, []Seat{0}) {
			t.Errorf("got places %v, want [0]", g.Places)
		}
		if got := g.DeclaredHands[0].Len(); got != 6 {
			t.Errorf("got %d declared cards, want 6", got)
		}
		if g.Hands[0].Len() != 0 || g.Active[0] {
			t.Error("the declarer's hand was not removed from play")
		}
		// The 3♠ left play with the declared hand; the 3♣ leads instead.
		if g.Turn != 1 || !g.OpeningCard.Equal(c(rThree, sClubs)) {
			t.Errorf("got turn %d opening card %v, want seat 1 with 3♣",
				g.Turn, g.OpeningCard)
		}
		mustFailPlay(t, g, 1, ErrIllegalMove, c(rSeven, sDiamonds))
		mustPlay(t, g, 1, c(rThree, sClubs))
		// The declared seat is out of the hand for good.
		mustFailPlay(t, g, 0, ErrOutOfTurn, c(rTwo, sSpades))
	})

	t.Run("decline", func(t *testing.T) {
		g := newGame()
		if err := g.StartPlay(); err != nil {
			t.Fatalf("StartPlay: %v", err)
		}
		if len(g.Places) != 0 {
			t.Errorf("got places %v, want none", g.Places)
		}
		if g.Turn != 0 {
			t.Errorf("got turn %d, want 0 (holder of 3♠)", g.Turn)
		}
		mustPlay(t, g, 0, c(rThree, sSpades))
		// The window has closed; declaration is no longer possible.
		mustFailDeclare := g.DeclareAutoWin(0)
		if !errors.Is(mustFailDeclare, ErrIllegalMove) {
			t.Errorf("declare after play began: got %v, want ErrIllegalMove", mustFailDeclare)
		}
	})
}

// TestKillerMultipleAutoWinPriorityIntegration verifies that Killer's
// declarations are resolved in priority order regardless of the order
// they arrive in, and that a hand ends when declarations leave one
// player.
func TestKillerMultipleAutoWinPriorityIntegration(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{ // seat 0: quad of 2s
			c(rThree, sSpades), c(rEight, sDiamonds), c(rJack, sSpades), c(rQueen, sDiamonds),
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
		},
		[]cardcore.Card{ // seat 1: six pairs, top pair aces
			c(rFour, sDiamonds),
			c(rNine, sSpades), c(rNine, sClubs),
			c(rTen, sSpades), c(rTen, sClubs),
			c(rJack, sDiamonds), c(rJack, sHearts),
			c(rQueen, sSpades), c(rQueen, sClubs),
			c(rKing, sSpades), c(rKing, sClubs),
			c(rAce, sSpades), c(rAce, sHearts),
		},
		[]cardcore.Card{ // seat 2: six pairs, top pair 8s
			c(rThree, sClubs), c(rThree, sDiamonds),
			c(rFour, sSpades), c(rFour, sClubs),
			c(rFive, sSpades), c(rFive, sClubs),
			c(rSix, sSpades), c(rSix, sClubs),
			c(rSeven, sSpades), c(rSeven, sClubs),
			c(rEight, sSpades), c(rEight, sHearts),
			c(rTen, sHearts),
		},
		[]cardcore.Card{ // seat 3: no automatic win
			c(rThree, sHearts),
			c(rFour, sHearts), c(rFive, sDiamonds), c(rSix, sHearts),
			c(rSeven, sDiamonds), c(rNine, sHearts), c(rJack, sClubs),
			c(rQueen, sHearts), c(rKing, sDiamonds), c(rAce, sClubs),
		},
	)

	// Claims arrive in reverse priority order.
	for _, seat := range []Seat{2, 1, 0} {
		if err := g.DeclareAutoWin(seat); err != nil {
			t.Fatalf("DeclareAutoWin(%d): %v", seat, err)
		}
	}
	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}

	// Places follow priority, not call order: quad of 2s, then aces-high
	// six pairs, then 8s-high six pairs; the one remaining player is last.
	want := []Seat{0, 1, 2, 3}
	if !slices.Equal(g.Places, want) {
		t.Errorf("got places %v, want %v", g.Places, want)
	}
	if g.Phase != PhaseScore {
		t.Errorf("got phase %d, want PhaseScore (one player left)", g.Phase)
	}
	for seat := range 3 {
		if g.DeclaredHands[seat] == nil {
			t.Errorf("seat %d's hand was not removed from play", seat)
		}
	}
	ended := eventsOfType[HandEndedEvent](g.Events())
	if len(ended) != 1 || !slices.Equal(ended[0].Places, want) {
		t.Errorf("got hand-ended events %v, want one with places %v", ended, want)
	}
}

// TestKillerSixPairsCountingIntegration verifies Killer's six-pairs
// partition rule through the public API: a quad of 2s is one quad-2s
// candidate, and a triple contributes one pair.
func TestKillerSixPairsCountingIntegration(t *testing.T) {
	t.Run("quad of 2s with four pairs is one quad-2s candidate", func(t *testing.T) {
		g := newKillerGameWindowOpen(t,
			[]cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades),
				c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
			},
			[]cardcore.Card{c(rEight, sSpades)},
		)
		if len(g.AutoWins) != 1 || g.AutoWins[0].Kind != AutoWinQuadTwos {
			t.Fatalf("got candidates %v, want one quad-2s", g.AutoWins)
		}
		if err := g.DeclareAutoWin(0); err != nil {
			t.Fatalf("DeclareAutoWin: %v", err)
		}
		if err := g.StartPlay(); err != nil {
			t.Fatalf("StartPlay: %v", err)
		}
		if !slices.Equal(g.Places, []Seat{0, 1}) || g.Phase != PhaseScore {
			t.Errorf("got places %v phase %d, want [0 1] at PhaseScore", g.Places, g.Phase)
		}
	})

	t.Run("five pairs and a triple qualifies", func(t *testing.T) {
		g := newKillerGameWindowOpen(t,
			[]cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs),
				c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs),
				c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades), c(rSeven, sClubs),
				c(rEight, sSpades), c(rEight, sClubs), c(rEight, sDiamonds),
			},
			[]cardcore.Card{c(rNine, sSpades)},
		)
		ok, err := g.CanDeclareAutoWin(0)
		if err != nil || !ok {
			t.Fatalf("CanDeclareAutoWin(0) = %v, %v; want true, nil", ok, err)
		}
	})
}

// TestKillerOneAutoWinnerThenNormalPlacesIntegration verifies that a
// declarer takes first place and play continues for the remaining
// places.
func TestKillerOneAutoWinnerThenNormalPlacesIntegration(t *testing.T) {
	g := newKillerGameWindowOpen(t,
		[]cardcore.Card{c(rEight, sDiamonds), c(rNine, sDiamonds),
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts)},
		[]cardcore.Card{c(rFour, sClubs), c(rEight, sSpades)},
		[]cardcore.Card{c(rFour, sDiamonds)},
		[]cardcore.Card{c(rThree, sClubs), c(rFive, sHearts), c(rNine, sHearts)},
	)
	if err := g.DeclareAutoWin(0); err != nil {
		t.Fatalf("DeclareAutoWin: %v", err)
	}
	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}
	if !slices.Equal(g.Places, []Seat{0}) {
		t.Fatalf("got places %v, want [0]", g.Places)
	}

	// Round 1: the 3♣ leads among the remaining players.
	if g.Turn != 3 {
		t.Fatalf("got turn %d, want 3 (holder of 3♣)", g.Turn)
	}
	mustPlay(t, g, 3, c(rThree, sClubs))
	mustPlay(t, g, 1, c(rFour, sClubs))
	mustPlay(t, g, 2, c(rFour, sDiamonds)) // seat 2 goes out in second place
	mustPlay(t, g, 3, c(rFive, sHearts))
	mustPlay(t, g, 1, c(rEight, sSpades)) // seat 1 goes out in third place
	if !g.PilePendingResolution {
		t.Fatal("the final play should pause the pile before scoring")
	}
	if err := g.ResolvePile(); err != nil {
		t.Fatalf("ResolvePile: %v", err)
	}
	want := []Seat{0, 2, 1, 3}
	if !slices.Equal(g.Places, want) {
		t.Errorf("got places %v, want %v", g.Places, want)
	}
	if g.Phase != PhaseScore {
		t.Errorf("got phase %d, want PhaseScore", g.Phase)
	}
}

// TestRejectedActionsAreAtomicIntegration verifies that rejected actions
// return the expected error and leave the game untouched; the mustFail
// helpers compare a summarizeGame snapshot around each call.
func TestRejectedActionsAreAtomicIntegration(t *testing.T) {
	t.Run("out-of-turn beat", func(t *testing.T) {
		g := newKillerGame(t,
			[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
			[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades)},
			[]cardcore.Card{c(rSix, sSpades), c(rSix, sClubs)},
			[]cardcore.Card{c(rSeven, sSpades), c(rSeven, sClubs)},
		)
		mustPlay(t, g, 0, c(rThree, sSpades))
		mustFailPlay(t, g, 2, ErrOutOfTurn, c(rSix, sSpades))
	})
	t.Run("finished seat", func(t *testing.T) {
		g := newKillerGame(t,
			[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts), c(rFive, sHearts)},
			[]cardcore.Card{c(rFour, sSpades), c(rFour, sClubs), c(rSix, sHearts)},
		)
		mustPlay(t, g, 0, c(rThree, sSpades))
		mustPlay(t, g, 1, c(rFour, sSpades))
		mustPass(t, g, 0)
		mustPlay(t, g, 1, c(rFour, sClubs))
		mustPlay(t, g, 1, c(rSix, sHearts)) // seat 1 goes out; pending
		mustFailPlay(t, g, 1, ErrIllegalMove, c(rFour, sSpades))
	})
	t.Run("non-owned and duplicate cards", func(t *testing.T) {
		g := newKillerGame(t,
			[]cardcore.Card{c(rThree, sSpades), c(rFive, sHearts)},
			[]cardcore.Card{c(rFour, sSpades)},
		)
		mustFailPlay(t, g, 0, ErrIllegalMove, c(rKing, sSpades))
		mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sSpades), c(rThree, sSpades))
	})
	t.Run("locked seat", func(t *testing.T) {
		g := newKillerGame(t,
			[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
			[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades), c(rSix, sHearts)},
		)
		mustPlay(t, g, 0, c(rThree, sSpades))
		mustPlay(t, g, 1, c(rFour, sSpades))
		mustPass(t, g, 0)
		// Seat 0 passed and is locked out; force its turn to verify the
		// lockout guard rejects even a beating play.
		g.Turn = 0
		mustFailPlay(t, g, 0, ErrIllegalMove, c(rThree, sHearts))
	})
	t.Run("declared seat", func(t *testing.T) {
		g := newKillerGameWindowOpen(t,
			[]cardcore.Card{c(rThree, sSpades),
				c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts)},
			[]cardcore.Card{c(rFour, sSpades), c(rFive, sSpades)},
			[]cardcore.Card{c(rSix, sSpades), c(rSeven, sSpades)},
		)
		if err := g.DeclareAutoWin(0); err != nil {
			t.Fatalf("DeclareAutoWin: %v", err)
		}
		if err := g.StartPlay(); err != nil {
			t.Fatalf("StartPlay: %v", err)
		}
		// Force the declared seat's turn to verify the guard.
		g.Turn = 0
		mustFailPlay(t, g, 0, ErrIllegalMove, c(rTwo, sSpades))
	})
}

// TestKillerRejectsInterruptChopIntegration verifies that out-of-turn
// chops do not exist under Killer rules.
func TestKillerRejectsInterruptChopIntegration(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rThree, sHearts)},
		[]cardcore.Card{c(rFour, sSpades)},
	)
	mustPlay(t, g, 0, c(rThree, sSpades))
	before := summarizeGame(g)
	err := g.InterruptChop(1, []cardcore.Card{
		c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
	})
	if !errors.Is(err, ErrIllegalMove) {
		t.Errorf("got %v, want ErrIllegalMove", err)
	}
	if after := summarizeGame(g); after != before {
		t.Errorf("rejected interrupt changed state: got %+v, want %+v", after, before)
	}
}

// TestFullKillerHandIntegration plays complete seeded hands from deal
// to places for each player count, declining all automatic wins.
func TestFullKillerHandIntegration(t *testing.T) {
	tests := []struct {
		numPlayers int
		seed       uint64
		undealt    int
	}{
		{2, 2002, 26},
		{3, 3003, 13},
		{4, 4004, 0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d players", tt.numPlayers), func(t *testing.T) {
			g := New(rand.New(rand.NewPCG(tt.seed, 1)), testKillerConfig(tt.numPlayers))
			if err := g.Deal(); err != nil {
				t.Fatalf("Deal: %v", err)
			}
			for i, h := range g.Hands {
				if got := h.Len(); got != HandSize {
					t.Fatalf("seat %d hand size: got %d, want %d", i, got, HandSize)
				}
			}
			if got := len(g.Undealt); got != tt.undealt {
				t.Fatalf("undealt: got %d, want %d", got, tt.undealt)
			}
			driveHand(t, g, false)
			assertPlacesPermutation(t, g)
			if got := len(eventsOfType[HandEndedEvent](g.Events())); got != 1 {
				t.Errorf("got %d hand-ended events, want 1", got)
			}
		})
	}
}

// TestFullKillerMatchIntegration plays a complete seeded match to
// PhaseEnd, declaring every automatic win found.
func TestFullKillerMatchIntegration(t *testing.T) {
	const matchLength = 5
	g := New(rand.New(rand.NewPCG(777, 7)), Config{
		Variant: Killer, NumPlayers: 3, Stake: 2, MatchLength: matchLength,
	})
	for g.Phase != PhaseEnd {
		if err := g.Deal(); err != nil {
			t.Fatalf("Deal: %v", err)
		}
		driveHand(t, g, true)
		if err := g.EndHand(); err != nil {
			t.Fatalf("EndHand: %v", err)
		}
	}
	if g.Hand != matchLength {
		t.Errorf("got %d hands, want %d", g.Hand, matchLength)
	}
	if got := len(eventsOfType[HandStartedEvent](g.Events())); got != matchLength {
		t.Errorf("got %d hand-started events, want %d", got, matchLength)
	}
	if got := len(eventsOfType[HandEndedEvent](g.Events())); got != matchLength {
		t.Errorf("got %d hand-ended events, want %d", got, matchLength)
	}
}

// testRNG returns a deterministic random source for tests.
func testRNG() *rand.Rand {
	return rand.New(rand.NewPCG(1, 2))
}

// testKillerConfig returns a Killer configuration for a single-hand match.
func testKillerConfig(numPlayers int) Config {
	return Config{Variant: Killer, NumPlayers: numPlayers, Stake: 2, MatchLength: 1}
}

// newKillerGame returns a Killer game with the hands dealt as given and
// play underway: the declaration window is closed and the holder of the
// lowest card is to open the first pile.
func newKillerGame(t *testing.T, hands ...[]cardcore.Card) *Game {
	t.Helper()
	g := newKillerGameWindowOpen(t, hands...)
	if err := g.StartPlay(); err != nil {
		t.Fatalf("StartPlay: %v", err)
	}
	return g
}

// newKillerGameWindowOpen returns a Killer game with the hands dealt as
// given and the declaration window still open.
func newKillerGameWindowOpen(t *testing.T, hands ...[]cardcore.Card) *Game {
	t.Helper()
	g := New(testRNG(), testKillerConfig(len(hands)))
	for i, h := range hands {
		g.Hands[i] = cardcore.NewHand(h)
		slices.SortFunc(g.Hands[i].Cards, compareCards)
	}
	g.Active = make([]bool, len(hands))
	for i := range g.Active {
		g.Active[i] = true
	}
	g.CardsPlayed = make([]int, len(hands))
	g.AutoWinClaims = make([]bool, len(hands))
	g.DeclaredHands = make([]*cardcore.Hand, len(hands))
	g.detectAutoWins()
	g.Phase = PhasePlay
	g.AutoWinWindowOpen = true
	assertCardIntegrity(t, g)
	return g
}

// mustPlay plays the cards for seat, fails the test on error, and checks
// the cross-cutting invariants.
func mustPlay(t *testing.T, g *Game, seat Seat, cards ...cardcore.Card) {
	t.Helper()
	if err := g.Play(seat, cards); err != nil {
		t.Fatalf("Play(%d, %v): %v", seat, cards, err)
	}
	assertCardIntegrity(t, g)
	assertTurnValid(t, g)
}

// mustPass passes for seat, fails the test on error, and checks the
// cross-cutting invariants.
func mustPass(t *testing.T, g *Game, seat Seat) {
	t.Helper()
	if err := g.Play(seat, nil); err != nil {
		t.Fatalf("pass(%d): %v", seat, err)
	}
	assertCardIntegrity(t, g)
	assertTurnValid(t, g)
}

// mustFailPlay plays the cards for seat, fails the test unless the error
// matches want, and verifies that nothing changed.
func mustFailPlay(t *testing.T, g *Game, seat Seat, want error, cards ...cardcore.Card) {
	t.Helper()
	before := summarizeGame(g)
	err := g.Play(seat, cards)
	if !errors.Is(err, want) {
		t.Fatalf("Play(%d, %v) error = %v, want %v", seat, cards, err, want)
	}
	if after := summarizeGame(g); after != before {
		t.Fatalf("rejected play changed state: got %+v, want %+v", after, before)
	}
}

// mustFailPass passes for seat, fails the test unless the error matches
// want, and verifies that nothing changed.
func mustFailPass(t *testing.T, g *Game, seat Seat, want error) {
	t.Helper()
	before := summarizeGame(g)
	err := g.Play(seat, nil)
	if !errors.Is(err, want) {
		t.Fatalf("pass(%d) error = %v, want %v", seat, err, want)
	}
	if after := summarizeGame(g); after != before {
		t.Fatalf("rejected pass changed state: got %+v, want %+v", after, before)
	}
}

// assertPanics fails the test unless f panics.
func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected a panic")
		}
	}()
	f()
}

// assertCardIntegrity verifies that every card appears at most once
// across hands, undealt cards, piles, and declared hands. It returns
// the total number of cards seen.
func assertCardIntegrity(t *testing.T, g *Game) int {
	t.Helper()
	seen := map[cardcore.Card]bool{}
	add := func(c cardcore.Card, zone string) {
		if seen[c] {
			t.Errorf("card %v appears twice (in %s)", c, zone)
		}
		seen[c] = true
	}
	for i, h := range g.Hands {
		if h == nil {
			continue
		}
		for _, c := range h.Cards {
			add(c, fmt.Sprintf("hand %d", i))
		}
	}
	for i, h := range g.DeclaredHands {
		if h == nil {
			continue
		}
		for _, c := range h.Cards {
			add(c, fmt.Sprintf("declared hand %d", i))
		}
	}
	for _, c := range g.Undealt {
		add(c, "undealt")
	}
	for _, play := range g.Pile.Plays {
		for _, c := range play.Combo.Cards {
			add(c, "current pile")
		}
	}
	for _, pile := range g.PileHistory {
		for _, play := range pile.Plays {
			for _, c := range play.Combo.Cards {
				add(c, "pile history")
			}
		}
	}
	return len(seen)
}

// assertTurnValid checks that whenever play is possible, the turn points
// at a seat that may act.
func assertTurnValid(t *testing.T, g *Game) {
	t.Helper()
	if g.Phase != PhasePlay || g.AutoWinWindowOpen || g.PilePendingResolution {
		return
	}
	if g.Turn < 0 || g.Turn >= Seat(len(g.Hands)) {
		t.Errorf("turn seat %d out of range", g.Turn)
		return
	}
	if !g.Active[g.Turn] {
		t.Errorf("turn seat %d is not active", g.Turn)
	}
	if g.Pile.Open && g.Pile.Locked[g.Turn] {
		t.Errorf("turn seat %d is locked out", g.Turn)
	}
}

// summarizeGame captures the observable state of a game for equality
// checks around rejected actions.
func summarizeGame(g *Game) gameSummary {
	var hands, top, locked, cardsPlayed strings.Builder
	for _, h := range g.Hands {
		if h == nil {
			continue
		}
		fmt.Fprintf(&hands, "%v;", h.Cards)
	}
	fmt.Fprintf(&top, "%v", g.Pile.Top.Cards)
	fmt.Fprintf(&locked, "%v", g.Pile.Locked)
	fmt.Fprintf(&cardsPlayed, "%v", g.CardsPlayed)
	return gameSummary{
		phase:       g.Phase,
		turn:        g.Turn,
		hands:       hands.String(),
		open:        g.Pile.Open,
		top:         top.String(),
		plays:       len(g.Pile.Plays),
		locked:      locked.String(),
		cardsPlayed: cardsPlayed.String(),
		historyLen:  len(g.PileHistory),
		places:      len(g.Places),
		events:      len(g.Events()),
		selfBeat:    g.SelfBeatContinuation,
		pending:     g.PilePendingResolution,
		windowOpen:  g.AutoWinWindowOpen,
	}
}

// eventsOfType returns the events of type T in order.
func eventsOfType[T Event](events []Event) []T {
	var out []T
	for _, e := range events {
		if ev, ok := e.(T); ok {
			out = append(out, ev)
		}
	}
	return out
}

// driveHand plays out a hand with a fixed policy: declare every
// automatic win when declare is true (decline all otherwise), always
// play the lowest legal combination, pass when there is none, and
// resolve every pile pause immediately. It requires g to be a freshly
// dealt game in the play phase — declaration window open or already
// closed — and drives it to PhaseScore. It checks the cross-cutting
// invariants after every action and fails if the hand does not end
// within a generous action cap.
func driveHand(t *testing.T, g *Game, declare bool) {
	t.Helper()
	for actions := 0; ; actions++ {
		if actions > 1000 {
			t.Fatal("hand did not end within 1000 actions")
		}
		if g.Phase == PhaseScore {
			assertFullCardIntegrity(t, g)
			return
		}
		if g.Phase != PhasePlay {
			t.Fatalf("driver reached phase %d before scoring", g.Phase)
		}
		if g.AutoWinWindowOpen {
			for seat := range g.Hands {
				ok, err := g.CanDeclareAutoWin(Seat(seat))
				if err != nil {
					t.Fatalf("CanDeclareAutoWin(%d): %v", seat, err)
				}
				if ok && declare {
					if err := g.DeclareAutoWin(Seat(seat)); err != nil {
						t.Fatalf("DeclareAutoWin(%d): %v", seat, err)
					}
				}
			}
			if err := g.StartPlay(); err != nil {
				t.Fatalf("StartPlay: %v", err)
			}
			assertFullCardIntegrity(t, g)
			continue
		}
		if g.PilePendingResolution {
			if err := g.ResolvePile(); err != nil {
				t.Fatalf("ResolvePile: %v", err)
			}
			assertFullCardIntegrity(t, g)
			continue
		}
		seat := g.Turn
		moves, err := g.LegalMoves(seat)
		if err != nil {
			t.Fatalf("LegalMoves(%d): %v", seat, err)
		}
		if len(moves) == 0 {
			canPass, err := g.CanPass(seat)
			if err != nil || !canPass {
				t.Fatalf("seat %d has no legal play and cannot pass "+
					"(canPass=%v, err=%v)", seat, canPass, err)
			}
			mustPass(t, g, seat)
			continue
		}
		mustPlay(t, g, seat, moves[0].Cards...)
		assertFullCardIntegrity(t, g)
	}
}

// assertFullCardIntegrity extends assertCardIntegrity by requiring all
// 52 cards to be accounted for (true of real deals, unlike fixtures).
func assertFullCardIntegrity(t *testing.T, g *Game) {
	t.Helper()
	if got := assertCardIntegrity(t, g); got != cardcore.DeckSize {
		t.Errorf("got %d cards across all zones, want %d", got, cardcore.DeckSize)
	}
}

// assertPlacesPermutation checks that places hold every seat exactly
// once.
func assertPlacesPermutation(t *testing.T, g *Game) {
	t.Helper()
	if len(g.Places) != len(g.Hands) {
		t.Fatalf("got %d places, want %d", len(g.Places), len(g.Hands))
	}
	seen := make([]bool, len(g.Hands))
	for _, seat := range g.Places {
		if seat < 0 || seat >= Seat(len(g.Hands)) || seen[seat] {
			t.Fatalf("places %v are not a permutation of the seats", g.Places)
		}
		seen[seat] = true
	}
}
