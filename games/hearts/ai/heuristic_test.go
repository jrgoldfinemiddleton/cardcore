package ai

import (
	"math/rand/v2"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

func newSeededHeuristic(seed uint64) *Heuristic {
	return NewHeuristic(rand.New(rand.NewPCG(seed, seed+1)))
}

func TestHeuristicChoosePassReturnsDistinctCardsFromHand(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

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

func TestHeuristicChoosePlayReturnsLegalCard(t *testing.T) {
	g := hearts.New()
	g.PassDir = hearts.PassHold
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	h := newSeededHeuristic(42)
	seat := g.Turn
	card := h.ChoosePlay(g.Clone(), seat)

	if err := g.PlayCard(seat, card); err != nil {
		t.Fatalf("PlayCard rejected ChoosePlay result %v: %v", card, err)
	}
}

func TestPassScoreUnprotectedQueen(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Five, cardcore.Spades),
		queenOfSpades,
	})

	score := passScore(queenOfSpades, hand)
	if score != 100 {
		t.Errorf("unprotected Q♠ score = %d, want 100", score)
	}
}

func TestPassScoreProtectedQueen(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Five, cardcore.Spades),
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
	})

	score := passScore(queenOfSpades, hand)
	if score != 2 {
		t.Errorf("protected Q♠ score = %d, want 2", score)
	}
}

func TestPassScoreHighSpades(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Two, cardcore.Diamonds),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.King, cardcore.Spades),
		c(cardcore.Ace, cardcore.Spades),
	})

	aceScore := passScore(c(cardcore.Ace, cardcore.Spades), hand)
	kingScore := passScore(c(cardcore.King, cardcore.Spades), hand)
	lowClubScore := passScore(c(cardcore.Five, cardcore.Clubs), hand)

	if aceScore <= lowClubScore {
		t.Errorf("A♠ score (%d) should exceed low club score (%d)", aceScore, lowClubScore)
	}
	if kingScore <= lowClubScore {
		t.Errorf("K♠ score (%d) should exceed low club score (%d)", kingScore, lowClubScore)
	}
}

func TestPassScoreHeartsBonus(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Diamonds),
		c(cardcore.Ace, cardcore.Hearts),
	})

	heartScore := passScore(c(cardcore.Ace, cardcore.Hearts), hand)
	diamondScore := passScore(c(cardcore.Ace, cardcore.Diamonds), hand)

	if heartScore <= diamondScore {
		t.Errorf("A♥ score (%d) should exceed A♦ score (%d)", heartScore, diamondScore)
	}
}

func TestPassScoreShortSuitBonus(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Diamonds),
	})

	singletonScore := passScore(c(cardcore.Ace, cardcore.Diamonds), hand)
	longSuitScore := passScore(c(cardcore.King, cardcore.Clubs), hand)

	if singletonScore <= longSuitScore {
		t.Errorf("singleton A♦ score (%d) should exceed long-suit K♣ score (%d)", singletonScore, longSuitScore)
	}
}

func TestChoosePassPassesUnprotectedQueen(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Two, cardcore.Diamonds),
		c(cardcore.Two, cardcore.Spades),
		queenOfSpades,
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, queenOfSpades) {
		t.Errorf("expected unprotected Q♠ to be passed, got %v", cards)
	}
}

func TestChoosePassKeepsProtectedQueen(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Nine, cardcore.Hearts),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
		c(cardcore.Queen, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
		c(cardcore.Ace, cardcore.Hearts),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.Five, cardcore.Spades),
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if passedContains(cards, queenOfSpades) {
		t.Errorf("expected protected Q♠ to be kept, but it was passed: %v", cards)
	}
}

func TestChoosePassKeepsHighSpadesWithProtectedQueen(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(cardcore.Ace, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
		c(cardcore.Queen, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.Five, cardcore.Spades),
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
		c(cardcore.King, cardcore.Spades),
		c(cardcore.Ace, cardcore.Spades),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if passedContains(cards, c(cardcore.Ace, cardcore.Spades)) {
		t.Errorf("expected A♠ to be kept with protected Q♠, but it was passed: %v", cards)
	}
	if passedContains(cards, c(cardcore.King, cardcore.Spades)) {
		t.Errorf("expected K♠ to be kept with protected Q♠, but it was passed: %v", cards)
	}
}

func TestChoosePassPrefersHighSpades(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Ace, cardcore.Diamonds),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.Five, cardcore.Spades),
		c(cardcore.King, cardcore.Spades),
		c(cardcore.Ace, cardcore.Spades),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, c(cardcore.Ace, cardcore.Spades)) {
		t.Errorf("expected A♠ to be passed, got %v", cards)
	}
	if !passedContains(cards, c(cardcore.King, cardcore.Spades)) {
		t.Errorf("expected K♠ to be passed, got %v", cards)
	}
}

func TestChoosePassVoidsShortSuit(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Diamonds),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, c(cardcore.Seven, cardcore.Diamonds)) {
		t.Errorf("expected singleton 7♦ to be passed, got %v", cards)
	}
}

func TestChoosePassVoidsTwoShortSuits(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Three, cardcore.Diamonds),
		c(cardcore.Two, cardcore.Hearts),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, c(cardcore.Three, cardcore.Diamonds)) {
		t.Errorf("expected singleton 3♦ to be passed, got %v", cards)
	}
	if !passedContains(cards, c(cardcore.Two, cardcore.Hearts)) {
		t.Errorf("expected singleton 2♥ to be passed, got %v", cards)
	}
}

func TestChoosePassTieBreaking(t *testing.T) {
	// 5 clubs + 4 diamonds + 4 spades (all low): 6♣ is the unique
	// highest (score 4), but 5♣, 5♦, 5♠ all tie at score 3. Only 2 of
	// the 3 tied cards can be passed, so the rng should vary which pair.
	hand := []cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Clubs),
		c(cardcore.Four, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Two, cardcore.Diamonds),
		c(cardcore.Three, cardcore.Diamonds),
		c(cardcore.Four, cardcore.Diamonds),
		c(cardcore.Five, cardcore.Diamonds),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Three, cardcore.Spades),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.Five, cardcore.Spades),
	}

	seen := make(map[cardcore.Card]bool)
	for seed := uint64(0); seed < 20; seed++ {
		g := hearts.New()
		if err := g.Deal(); err != nil {
			t.Fatalf("Deal error: %v", err)
		}
		setupPassHand(t, g, hearts.South, hand)

		h := newSeededHeuristic(seed)
		cards := h.ChoosePass(g, hearts.South)
		for _, card := range cards {
			seen[card] = true
		}
	}

	if len(seen) <= hearts.PassCount {
		t.Errorf("expected tie-breaking to vary selections across seeds, but only saw %d distinct cards", len(seen))
	}
}

func TestHeuristicFullGameIntegration(t *testing.T) {
	const (
		numGames  = 10
		maxRounds = 20
	)

	for game := range numGames {
		seed := uint64(game)
		h := newSeededHeuristic(seed)
		g := hearts.New()

		for range maxRounds {
			playRoundWithPlayer(t, g, h, seed)

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

var _ hearts.Player = (*Heuristic)(nil)

func setupPassHand(t *testing.T, g *hearts.Game, seat hearts.Seat, cards []cardcore.Card) {
	t.Helper()
	g.Hands[seat] = cardcore.NewHand(cards)
}

func passedContains(cards [hearts.PassCount]cardcore.Card, target cardcore.Card) bool {
	for _, c := range cards {
		if c == target {
			return true
		}
	}
	return false
}
