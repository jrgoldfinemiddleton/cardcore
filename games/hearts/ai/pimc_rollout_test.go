package ai

import (
	"fmt"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// firstLegalPolicy is a deterministic test player that always plays the
// first card returned by LegalMoves. It is used to make rollout output
// predictable enough to test wire-up and determinism without depending
// on Random's RNG behavior.
type firstLegalPolicy struct{}

// TestLeafScoreNoShoot verifies the no-shoot branch returns RoundPts[seat].
func TestLeafScoreNoShoot(t *testing.T) {
	// Seat enum order: South=0, West=1, North=2, East=3.
	pts := [hearts.NumPlayers]int{5, 8, 10, 3}
	tests := []struct {
		seat hearts.Seat
		want int
	}{
		{hearts.South, 5},
		{hearts.West, 8},
		{hearts.North, 10},
		{hearts.East, 3},
	}
	for _, tt := range tests {
		got := leafScore(pts, tt.seat)
		if got != tt.want {
			t.Errorf("seat %d: got %d, want %d", tt.seat, got, tt.want)
		}
	}
}

// TestLeafScoreSeatShoots verifies the shooter receives 0 and other seats
// receive MoonPoints.
func TestLeafScoreSeatShoots(t *testing.T) {
	// South (index 0) shoots.
	pts := [hearts.NumPlayers]int{hearts.MoonPoints, 0, 0, 0}
	tests := []struct {
		seat hearts.Seat
		want int
	}{
		{hearts.South, 0},
		{hearts.West, hearts.MoonPoints},
		{hearts.North, hearts.MoonPoints},
		{hearts.East, hearts.MoonPoints},
	}
	for _, tt := range tests {
		got := leafScore(pts, tt.seat)
		if got != tt.want {
			t.Errorf("seat %d: got %d, want %d", tt.seat, got, tt.want)
		}
	}
}

// TestLeafScoreOpponentShoots verifies non-shooter seats receive
// MoonPoints when an opponent shoots.
func TestLeafScoreOpponentShoots(t *testing.T) {
	// North (index 2) shoots.
	pts := [hearts.NumPlayers]int{0, 0, hearts.MoonPoints, 0}
	got := leafScore(pts, hearts.South)
	if got != hearts.MoonPoints {
		t.Errorf("got %d, want %d", got, hearts.MoonPoints)
	}
}

// TestLeafScoreSumIs26ButNoShoot verifies the no-shoot branch correctly
// handles the case where points sum to MoonPoints across multiple seats
// (no single seat hit MoonPoints alone).
func TestLeafScoreSumIs26ButNoShoot(t *testing.T) {
	// Seat enum order: South=0, West=1, North=2, East=3.
	pts := [hearts.NumPlayers]int{3, 5, 8, 10}
	got := leafScore(pts, hearts.North)
	if got != 8 {
		t.Errorf("got %d, want 8 (no shoot, return RoundPts[seat])", got)
	}
}

// TestRolloutDoesNotMutateInput verifies rollout leaves the input game
// unchanged. Snapshot key fields before and after.
func TestRolloutDoesNotMutateInput(t *testing.T) {
	g := freshPlayGame(t)
	seat := g.Turn

	wantPhase := g.Phase
	wantTurn := g.Turn
	wantTrickNum := g.TrickNum
	wantTrickCount := g.Trick.Count
	wantHandLen := len(g.Hands[seat].Cards)
	wantHearts := g.HeartsBroken

	legal, err := g.LegalMoves(seat)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	deal := swappedDeal(t, g, seat)

	_ = rollout(g, seat, legal[0], deal, firstLegalPolicy{})

	if g.Phase != wantPhase {
		t.Errorf("g.Phase mutated: got %d, want %d", g.Phase, wantPhase)
	}
	if g.Turn != wantTurn {
		t.Errorf("g.Turn mutated: got %d, want %d", g.Turn, wantTurn)
	}
	if g.TrickNum != wantTrickNum {
		t.Errorf("g.TrickNum mutated: got %d, want %d", g.TrickNum, wantTrickNum)
	}
	if g.Trick.Count != wantTrickCount {
		t.Errorf("g.Trick.Count mutated: got %d, want %d", g.Trick.Count, wantTrickCount)
	}
	if len(g.Hands[seat].Cards) != wantHandLen {
		t.Errorf("g.Hands[%d] length mutated: got %d, want %d",
			seat, len(g.Hands[seat].Cards), wantHandLen)
	}
	if g.HeartsBroken != wantHearts {
		t.Errorf("g.HeartsBroken mutated: got %v, want %v", g.HeartsBroken, wantHearts)
	}
}

// TestRolloutCompletesAndIsDeterministic verifies rollout runs to
// completion and produces the same leaf score on two calls with the
// same inputs and a deterministic policy. This validates wire-up:
// clone, install deal, play candidate, run policy to terminal state,
// extract leaf score.
func TestRolloutCompletesAndIsDeterministic(t *testing.T) {
	g := freshPlayGame(t)
	seat := g.Turn
	legal, err := g.LegalMoves(seat)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	deal := swappedDeal(t, g, seat)

	got1 := rollout(g, seat, legal[0], deal, firstLegalPolicy{})
	got2 := rollout(g, seat, legal[0], deal, firstLegalPolicy{})

	if got1 != got2 {
		t.Errorf("rollout not deterministic: got %d then %d", got1, got2)
	}
	if got1 < 0 || got1 > hearts.MoonPoints {
		t.Errorf("leaf score out of range: got %d, want [0, %d]", got1, hearts.MoonPoints)
	}
}

// TestRolloutAcceptsAnyPlayer verifies rollout works with any
// hearts.Player implementation, not just firstLegalPolicy.
func TestRolloutAcceptsAnyPlayer(t *testing.T) {
	g := freshPlayGame(t)
	seat := g.Turn
	legal, err := g.LegalMoves(seat)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	deal := swappedDeal(t, g, seat)

	policy := NewRandom(rngWithSeed(1, 2))
	got := rollout(g, seat, legal[0], deal, policy)
	if got < 0 || got > hearts.MoonPoints {
		t.Errorf("leaf score out of range: got %d, want [0, %d]", got, hearts.MoonPoints)
	}
}

// TestRolloutPanicsOnWrongPhase verifies rollout panics when g is not
// in PhasePlay.
func TestRolloutPanicsOnWrongPhase(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePass
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("rollout did not panic on wrong phase")
		}
	}()
	rollout(g, hearts.North, twoOfClubs, sampledDeal{}, firstLegalPolicy{})
}

// TestRolloutPanicsOnWrongTurn verifies rollout panics when g.Turn != seat.
func TestRolloutPanicsOnWrongTurn(t *testing.T) {
	g := freshPlayGame(t)
	wrongSeat := (g.Turn + 1) % hearts.NumPlayers
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("rollout did not panic on wrong turn")
		}
	}()
	rollout(g, wrongSeat, twoOfClubs, sampledDeal{}, firstLegalPolicy{})
}

// TestRolloutPanicsOnIllegalCandidate verifies rollout panics when the
// candidate card is not a legal move (engine PlayCard error wrapped).
func TestRolloutPanicsOnIllegalCandidate(t *testing.T) {
	g := freshPlayGame(t)
	seat := g.Turn
	deal := swappedDeal(t, g, seat)

	legal, err := g.LegalMoves(seat)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	legalSet := make(map[cardcore.Card]bool, len(legal))
	for _, c := range legal {
		legalSet[c] = true
	}
	var illegal cardcore.Card
	found := false
	for _, card := range g.Hands[seat].Cards {
		if !legalSet[card] {
			illegal = card
			found = true
			break
		}
	}
	if !found {
		for r := rTwo; r <= rAce; r++ {
			for _, s := range []cardcore.Suit{sClubs, sDiamonds, sHearts, sSpades} {
				cand := c(r, s)
				if !legalSet[cand] {
					illegal = cand
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	if !found {
		t.Skip("could not construct an illegal candidate for this fixture")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("rollout did not panic on illegal candidate %v", illegal)
		}
	}()
	rollout(g, seat, illegal, deal, firstLegalPolicy{})
}

// ChoosePlay returns the first legal move; deterministic for testing.
func (firstLegalPolicy) ChoosePlay(g *hearts.Game, seat hearts.Seat) cardcore.Card {
	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic("firstLegalPolicy: LegalMoves error: " + err.Error())
	}
	return legal[0]
}

// ChoosePass returns the first PassCount cards from seat's hand;
// deterministic for testing. Not exercised by rollout (rollout enters
// the game in PhasePlay) but required by the hearts.Player interface.
func (firstLegalPolicy) ChoosePass(
	g *hearts.Game, seat hearts.Seat,
) [hearts.PassCount]cardcore.Card {
	var out [hearts.PassCount]cardcore.Card
	for i := range hearts.PassCount {
		out[i] = g.Hands[seat].Cards[i]
	}
	return out
}

// freshPlayGame deals a new Hearts game and advances it to PhasePlay,
// handling the pass phase if needed by delegating to firstLegalPolicy.
// The resulting game has all four hands populated and Turn set to the
// holder of the 2♣ (the rules-mandated opener).
func freshPlayGame(t *testing.T) *hearts.Game {
	t.Helper()
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	if g.Phase == hearts.PhasePass {
		policy := firstLegalPolicy{}
		for i := hearts.Seat(0); i < hearts.NumPlayers; i++ {
			cards := policy.ChoosePass(g.Clone(), i)
			if err := g.SetPass(i, cards); err != nil {
				t.Fatalf("SetPass(%d) error: %v", i, err)
			}
		}
	}
	if g.Phase != hearts.PhasePlay {
		t.Fatalf("expected PhasePlay after setup, got phase %d", g.Phase)
	}
	return g
}

// dealFromGame extracts a sampledDeal that exactly mirrors g's current
// hands. Useful only as a baseline for tests that don't care about
// exercising rollout's deal-install step (every installed hand equals
// what's already there, so the install is a no-op).
//
// CALLER BEWARE: this leaks every seat's hand by construction. In
// production the PIMC player must never see opponents' true hands —
// only sampled hypothetical hands — or it would exploit hidden
// information. This is safe in tests because tests own the game and
// know everything by construction; the helper exists only to provide
// a valid sampledDeal shape, not to model what a real sampler would
// produce.
func dealFromGame(g *hearts.Game) sampledDeal {
	var d sampledDeal
	for s := range hearts.NumPlayers {
		d[s] = *cardcore.NewHand(g.Hands[s].Cards)
	}
	return d
}

// swappedDeal returns a sampledDeal mirroring g's hands except that two
// cards are swapped between two opponent seats. This exercises rollout's
// deal-install step (the installed hands differ from clone's existing
// hands) without breaking the engine's first-trick rule that the seat
// holding 2♣ must be the one to lead trick 0. The swap deliberately
// avoids 2♣ so the 2♣-holder seat is unchanged and matches g.Turn.
//
// Seat's own entry in the deal equals g.Hands[seat] unchanged, satisfying
// rollout's caller contract that deal[seat] match seat's real hand.
//
// t.Fatal if g doesn't have at least two non-seat opponents holding
// non-2♣ cards (impossible at round 0 trick 0 with full 13-card hands;
// the helper is intended for that fixture shape).
func swappedDeal(t *testing.T, g *hearts.Game, seat hearts.Seat) sampledDeal {
	t.Helper()
	d := dealFromGame(g)

	var opponents []hearts.Seat
	for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
		if s != seat {
			opponents = append(opponents, s)
		}
	}
	if len(opponents) < 2 {
		t.Fatalf("swappedDeal: need at least 2 opponents, got %d", len(opponents))
	}
	a, b := opponents[0], opponents[1]

	cardA, okA := firstNonTwoOfClubs(d[a].Cards)
	cardB, okB := firstNonTwoOfClubs(d[b].Cards)
	if !okA || !okB {
		t.Fatalf("swappedDeal: opponents %d and %d don't both have a non-2♣ card", a, b)
	}

	handA := replaceCard(d[a].Cards, cardA, cardB)
	handB := replaceCard(d[b].Cards, cardB, cardA)
	d[a] = *cardcore.NewHand(handA)
	d[b] = *cardcore.NewHand(handB)
	return d
}

// firstNonTwoOfClubs returns the first card in cards that is not 2♣.
// Returns (zero, false) if every card is 2♣ (or cards is empty).
func firstNonTwoOfClubs(cards []cardcore.Card) (cardcore.Card, bool) {
	for _, c := range cards {
		if c != twoOfClubs {
			return c, true
		}
	}
	return cardcore.Card{}, false
}

// replaceCard returns a fresh slice equal to cards with the first
// occurrence of old replaced by new. Panics if old is not present.
func replaceCard(cards []cardcore.Card, old, new cardcore.Card) []cardcore.Card {
	out := make([]cardcore.Card, len(cards))
	copy(out, cards)
	for i, c := range out {
		if c == old {
			out[i] = new
			return out
		}
	}
	panic(fmt.Sprintf("replaceCard: %v not found in %v", old, cards))
}
