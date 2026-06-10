package ai

import (
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// Compile-time check that Random satisfies hearts.Player.
var _ hearts.Player = (*Random)(nil)

// TestChoosePassReturnsDistinctCardsFromHand verifies that ChoosePass returns
// three distinct cards that exist in the player's hand.
func TestChoosePassReturnsDistinctCardsFromHand(t *testing.T) {
	g := hearts.New(rand.New(rand.NewPCG(1, 2)))
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	r := newSeededRandom(42)
	cards := r.ChoosePass(g, hearts.South)

	for i, card := range cards {
		if !g.Hands[hearts.South].Contains(card) {
			t.Fatalf("ChoosePass card %d (%v) not in hand", i, card)
		}
	}

	for i := range len(cards) {
		for j := i + 1; j < len(cards); j++ {
			if cards[i].Equal(cards[j]) {
				t.Fatalf("ChoosePass returned duplicate: %v at positions %d and %d", cards[i], i, j)
			}
		}
	}
}

// TestChoosePlayReturnsLegalCard verifies that ChoosePlay returns a card accepted by PlayCard.
func TestChoosePlayReturnsLegalCard(t *testing.T) {
	g := hearts.New(rand.New(rand.NewPCG(1, 2)))
	g.PassDir = hearts.PassHold
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	r := newSeededRandom(42)
	seat := g.Turn
	card := r.ChoosePlay(g.Clone(), seat)

	if err := g.PlayCard(seat, card); err != nil {
		t.Fatalf("PlayCard rejected ChoosePlay result %v: %v", card, err)
	}
}

// TestDeterminism verifies that identical seeds produce identical ChoosePass
// and ChoosePlay results.
func TestDeterminism(t *testing.T) {
	g := hearts.New(rand.New(rand.NewPCG(1, 2)))
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	const seed uint64 = 99

	r1 := newSeededRandom(seed)
	pass1 := r1.ChoosePass(g.Clone(), hearts.South)

	r2 := newSeededRandom(seed)
	pass2 := r2.ChoosePass(g.Clone(), hearts.South)

	if pass1 != pass2 {
		t.Fatalf("same seed produced different passes: %v vs %v", pass1, pass2)
	}

	hold := hearts.New(rand.New(rand.NewPCG(2, 3)))
	hold.PassDir = hearts.PassHold
	if err := hold.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	r3 := newSeededRandom(seed)
	play1 := r3.ChoosePlay(hold.Clone(), hold.Turn)

	r4 := newSeededRandom(seed)
	play2 := r4.ChoosePlay(hold.Clone(), hold.Turn)

	if !play1.Equal(play2) {
		t.Fatalf("same seed produced different plays: %v vs %v", play1, play2)
	}
}

// TestLegalityAcrossGames verifies that Random produces legal moves across 200 seeded games.
func TestLegalityAcrossGames(t *testing.T) {
	for seed := uint64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed+1))
		r := NewRandom(rng)
		g := hearts.New(rand.New(rand.NewPCG(seed+1000, seed+1001)))

		playRoundWithPlayer(t, g, r, seed)
	}
}

// TestDeterministicGame ensures that two identical seeds with identical
// Random players produce identical full game outcomes (scores and trick history).
func TestDeterministicGame(t *testing.T) {
	// We'll run two complete games with four Random players seeded identically
	// and compare the final outcomes.
	const seed uint64 = 1234

	// Create four seeded Random players with the same seed for determinism.
	pA1 := newSeededRandom(seed)
	pA2 := newSeededRandom(seed)
	pA3 := newSeededRandom(seed)
	pA4 := newSeededRandom(seed)
	players1 := [hearts.NumPlayers]hearts.Player{pA1, pA2, pA3, pA4}

	rng1 := rand.New(rand.NewPCG(seed, seed+1))
	g1 := hearts.New(rng1)

	// Run a full game to completion using the deterministic players.
	for g1.Phase != hearts.PhaseEnd {
		playRoundWithPlayers(t, g1, players1, seed)
	}

	// Second identical game with new Random instances but same seeds.
	pB1 := newSeededRandom(seed)
	pB2 := newSeededRandom(seed)
	pB3 := newSeededRandom(seed)
	pB4 := newSeededRandom(seed)
	players2 := [hearts.NumPlayers]hearts.Player{pB1, pB2, pB3, pB4}

	rng2 := rand.New(rand.NewPCG(seed, seed+1))
	g2 := hearts.New(rng2)

	for g2.Phase != hearts.PhaseEnd {
		playRoundWithPlayers(t, g2, players2, seed)
	}

	// Compare final outcomes: scores and trick history should be identical.
	if !reflect.DeepEqual(g1.Scores, g2.Scores) {
		t.Fatalf("deterministic game scores differ: %v vs %v", g1.Scores, g2.Scores)
	}
	if !reflect.DeepEqual(g1.TrickHistory, g2.TrickHistory) {
		t.Fatalf(
			"deterministic game trick history differs: %v vs %v",
			g1.TrickHistory, g2.TrickHistory,
		)
	}
	if g1.Phase != g2.Phase {
		t.Fatalf("deterministic game phase mismatch: %d vs %d", g1.Phase, g2.Phase)
	}
}

// TestFullGameIntegration runs 10 complete games with Random players and
// verifies structural invariants: games terminate, winner has the lowest score.
func TestFullGameIntegration(t *testing.T) {
	const (
		numGames  = 10
		maxRounds = 20
	)

	for game := range numGames {
		seed := uint64(game)
		rng := rand.New(rand.NewPCG(seed, seed+1))
		r := NewRandom(rng)
		g := hearts.New(rand.New(rand.NewPCG(seed+1000, seed+1001)))

		for range maxRounds {
			playRoundWithPlayer(t, g, r, seed)

			if g.Phase == hearts.PhaseEnd {
				break
			}
		}

		if g.Phase != hearts.PhaseEnd {
			t.Fatalf("game %d: did not end within %d rounds", game, maxRounds)
		}

		winner, err := g.Winner()
		if err != nil {
			t.Fatalf("game %d: Winner error: %v", game, err)
		}
		for i := hearts.Seat(0); i < hearts.NumPlayers; i++ {
			if g.Scores[i] < g.Scores[winner] {
				t.Errorf("game %d: player %d has score %d, lower than winner %d with %d",
					game, i, g.Scores[i], winner, g.Scores[winner])
			}
		}
	}
}

// newSeededRandom creates a Random player with a deterministic RNG for test reproducibility.
func newSeededRandom(seed uint64) *Random {
	return NewRandom(rand.New(rand.NewPCG(seed, seed+1)))
}
