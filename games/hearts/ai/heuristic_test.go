package ai

import (
	"math/rand/v2"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

var _ hearts.Player = (*Heuristic)(nil)

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
		kingOfSpades,
		aceOfSpades,
	})

	aceScore := passScore(aceOfSpades, hand)
	kingScore := passScore(kingOfSpades, hand)
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
		kingOfSpades,
		aceOfSpades,
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if passedContains(cards, aceOfSpades) {
		t.Errorf("expected A♠ to be kept with protected Q♠, but it was passed: %v", cards)
	}
	if passedContains(cards, kingOfSpades) {
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
		kingOfSpades,
		aceOfSpades,
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, aceOfSpades) {
		t.Errorf("expected A♠ to be passed, got %v", cards)
	}
	if !passedContains(cards, kingOfSpades) {
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

func TestLeadScorePrefersLowFromLongSuit(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.Three, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Nine, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	sixScore := leadScore(c(cardcore.Six, cardcore.Clubs), g, a)
	aceScore := leadScore(c(cardcore.Ace, cardcore.Clubs), g, a)

	if sixScore <= aceScore {
		t.Errorf("6♣ lead score (%d) should exceed A♣ lead score (%d) from long suit", sixScore, aceScore)
	}
}

func TestLeadScorePrefersHighFromShortSuitEarly(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Three, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	kingScore := leadScore(c(cardcore.King, cardcore.Diamonds), g, a)
	threeScore := leadScore(c(cardcore.Three, cardcore.Diamonds), g, a)

	if kingScore <= threeScore {
		t.Errorf("K♦ lead score (%d) should exceed 3♦ lead score (%d) from short suit early", kingScore, threeScore)
	}
}

// trickHistory is nil for brevity — only g.TrickNum matters for the
// early/late threshold under test.
func TestLeadScorePrefersLowFromShortSuitLate(t *testing.T) {
	g := setupLeadState(hearts.South, 7, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
	}, nil)

	a := analyze(g, hearts.South)
	sixScore := leadScore(c(cardcore.Six, cardcore.Diamonds), g, a)
	kingScore := leadScore(c(cardcore.King, cardcore.Diamonds), g, a)

	if sixScore <= kingScore {
		t.Errorf("6♦ lead score (%d) should exceed K♦ score (%d) from short suit late game", sixScore, kingScore)
	}
}

func TestLeadScoreAvoidsOpponentVoidSuit(t *testing.T) {
	// East is known void in diamonds from trick history.
	trickHistory := []hearts.Trick{
		validFirstTrick(),
		{
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Three, cardcore.Diamonds),
				hearts.West:  c(cardcore.Four, cardcore.Diamonds),
				hearts.North: c(cardcore.Five, cardcore.Diamonds),
				hearts.East:  c(cardcore.Two, cardcore.Hearts), // void in diamonds
			},
		},
	}

	g := setupLeadState(hearts.South, 2, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
	}, trickHistory)
	g.HeartsBroken = true

	a := analyze(g, hearts.South)
	kingDiamondScore := leadScore(c(cardcore.King, cardcore.Diamonds), g, a)
	clubScore := leadScore(c(cardcore.Ten, cardcore.Clubs), g, a)
	sixDiamondScore := leadScore(c(cardcore.Six, cardcore.Diamonds), g, a)

	if kingDiamondScore >= clubScore {
		t.Errorf("K♦ lead score (%d) should be less than 10♣ lead score (%d) when opponent void in diamonds",
			kingDiamondScore, clubScore)
	}
	if sixDiamondScore >= clubScore {
		t.Errorf("6♦ lead score (%d) should be less than 10♣ lead score (%d) — lower diamond still penalized by void",
			sixDiamondScore, clubScore)
	}
}

func TestLeadScoreSafeWhenGuaranteedLowest(t *testing.T) {
	// 2♦ and 3♦ already played, so 4♦ is guaranteed lowest diamond.
	// Leading 4♦ is safe even if opponent is void in diamonds.
	trickHistory := []hearts.Trick{
		validFirstTrick(),
		{
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Two, cardcore.Diamonds),
				hearts.West:  c(cardcore.Three, cardcore.Diamonds),
				hearts.North: c(cardcore.Five, cardcore.Diamonds),
				hearts.East:  c(cardcore.Two, cardcore.Hearts), // void in diamonds
			},
		},
	}

	g := setupLeadState(hearts.South, 2, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.Four, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Hearts),
	}, trickHistory)
	g.HeartsBroken = true

	a := analyze(g, hearts.South)
	fourDiamondScore := leadScore(c(cardcore.Four, cardcore.Diamonds), g, a)
	clubScore := leadScore(c(cardcore.Six, cardcore.Clubs), g, a)

	if fourDiamondScore < clubScore {
		t.Errorf("guaranteed lowest 4♦ lead score (%d) should be >= 6♣ score (%d) despite opponent void", fourDiamondScore, clubScore)
	}
}

func TestLeadScoreUnprotectedQueenUrgency(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	// K♦ is a singleton non-spade — should get Q♠ urgency bonus.
	kingDiamondScore := leadScore(c(cardcore.King, cardcore.Diamonds), g, a)
	lowClubScore := leadScore(c(cardcore.Six, cardcore.Clubs), g, a)

	if kingDiamondScore <= lowClubScore {
		t.Errorf("K♦ lead score (%d) should exceed 2♣ score (%d) with unprotected Q♠ urgency",
			kingDiamondScore, lowClubScore)
	}
}

func TestLeadScoreAvoidsSpadesWithHighSpades(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		kingOfSpades,
		aceOfSpades,
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	spadeScore := leadScore(c(cardcore.Six, cardcore.Spades), g, a)
	clubScore := leadScore(c(cardcore.Six, cardcore.Clubs), g, a)

	if spadeScore >= clubScore {
		t.Errorf("6♠ lead score (%d) should be less than 6♣ score (%d) when holding A♠/K♠",
			spadeScore, clubScore)
	}
}

func TestLeadScoreFlushesQueenWithoutHighSpades(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Jack, cardcore.Spades),
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	jackSpadeScore := leadScore(c(cardcore.Jack, cardcore.Spades), g, a)
	sixSpadeScore := leadScore(c(cardcore.Six, cardcore.Spades), g, a)
	clubScore := leadScore(c(cardcore.Jack, cardcore.Clubs), g, a)

	if jackSpadeScore <= clubScore {
		t.Errorf("J♠ flush score (%d) should exceed J♣ score (%d) when flushing Q♠ safely",
			jackSpadeScore, clubScore)
	}
	if jackSpadeScore <= sixSpadeScore {
		t.Errorf("J♠ flush score (%d) should exceed 6♠ score (%d) — prefer highest below Q♠",
			jackSpadeScore, sixSpadeScore)
	}
}

func TestChooseLeadAvoidsHearts(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card.Suit == cardcore.Hearts {
		t.Errorf("expected non-heart lead, got %v", card)
	}
}

func TestFollowLastCleanTrickWinsWithHighest(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Nine, cardcore.Spades),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Seven, cardcore.Clubs)},
			{hearts.West, c(cardcore.Eight, cardcore.Clubs)},
			{hearts.North, c(cardcore.Nine, cardcore.Clubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(cardcore.Jack, cardcore.Clubs) {
		t.Errorf("expected J♣ (highest to win clean trick), got %v", card)
	}
}

func TestFollowLastTrickHasPointsShedsHighest(t *testing.T) {
	g := setupFollowState(hearts.East, 2, []cardcore.Card{
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Nine, cardcore.Spades),
		c(cardcore.Nine, cardcore.Hearts),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick(), {
		Leader: hearts.South,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Six, cardcore.Clubs),
			hearts.West:  c(cardcore.King, cardcore.Clubs),
			hearts.North: c(cardcore.Three, cardcore.Hearts),
			hearts.East:  c(cardcore.Ace, cardcore.Clubs),
		},
	}},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Hearts)},
			{hearts.West, c(cardcore.Seven, cardcore.Hearts)},
			{hearts.North, c(cardcore.Eight, cardcore.Hearts)},
		})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(cardcore.Jack, cardcore.Hearts) {
		t.Errorf("expected J♥ (forced to win, shed highest), got %v", card)
	}
}

// East has 6♣, 8♣ (lose to J♣) and K♣ (wins). North sloughed 5♥
// so trickPts=1. Losers get +50 bonus: 8♣ scores 56, K♣ scores 11.
func TestFollowLastTrickHasPointsPrefersDuck(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Nine, cardcore.Spades),
		c(cardcore.Six, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Ten, cardcore.Clubs)},
			{hearts.West, c(cardcore.Jack, cardcore.Clubs)},
			{hearts.North, c(cardcore.Five, cardcore.Hearts)},
		})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(cardcore.Eight, cardcore.Clubs) {
		t.Errorf("expected 8♣ (highest loser, +50 duck bonus over K♣), got %v", card)
	}
}

// East is last, trick is clean. Both 9♦ and K♦ win; K♦ preferred
// (higher rank). 3♦ loses (score 1) and should rank below winners.
func TestFollowLastCleanTrickWins(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Three, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Diamonds)},
			{hearts.West, c(cardcore.Seven, cardcore.Diamonds)},
			{hearts.North, c(cardcore.Eight, cardcore.Diamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(cardcore.King, cardcore.Diamonds) {
		t.Errorf("expected K♦ (win clean trick, higher rank preferred), got %v", card)
	}
}

// East is last, trick is clean, hand is almost all Ten+.
// highCardRatio = 10*10/12 = 8, so danger*2 = 16.
// 9♦ wins (score 7-16 = -9), 3♦ loses (score 1).
// High danger makes losing preferable over winning.
func TestFollowLastCleanHighDangerPrefersDuck(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Three, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Ten, cardcore.Spades),
		c(cardcore.Jack, cardcore.Spades),
		c(cardcore.King, cardcore.Spades),
		c(cardcore.Ace, cardcore.Spades),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
		c(cardcore.Queen, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
		c(cardcore.Ace, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Diamonds)},
			{hearts.West, c(cardcore.Seven, cardcore.Diamonds)},
			{hearts.North, c(cardcore.Eight, cardcore.Diamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(cardcore.Three, cardcore.Diamonds) {
		t.Errorf("expected 3♦ (high danger discourages winning clean trick), got %v", card)
	}
}

// North is not last (East still to play). West sloughed 5♥ (1 pt).
// All of North's diamonds win; prefers lowest winner 8♦ (Ace-Rank = 6).
func TestFollowNotLastPointsDucksLowestWinner(t *testing.T) {
	g := setupFollowState(hearts.North, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Jack, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Diamonds)},
			{hearts.West, c(cardcore.Five, cardcore.Hearts)},
		})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.North)

	if card != c(cardcore.Eight, cardcore.Diamonds) {
		t.Errorf("expected 8♦ (lowest winner to duck points), got %v", card)
	}
}

// West is not last (Count=1, playersLeft=2). Clean trick. Both 8♦
// and K♦ win; K♦ preferred (higher bonus after playersLeft penalty).
func TestFollowNotLastCleanWins(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Seven, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Clubs),
		c(cardcore.Ten, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Diamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != c(cardcore.King, cardcore.Diamonds) {
		t.Errorf("expected K♦ (win clean trick, higher rank preferred), got %v", card)
	}
}

// Verifies AI picks 8♣ (highest loser), not J♣/K♣ (winners).
func TestFollowUnderWinnerShedsHighest(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Clubs),
		c(cardcore.Eight, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.King, cardcore.Clubs),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Ten, cardcore.Diamonds),
		c(cardcore.Jack, cardcore.Diamonds),
		c(cardcore.Queen, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Nine, cardcore.Spades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Ten, cardcore.Clubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != c(cardcore.Eight, cardcore.Clubs) {
		t.Errorf("expected 8♣ (highest that still loses to 10♣), got %v", card)
	}
}

func TestFollowQueenOfSpadesDumpsUnderHigherSpade(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Ten, cardcore.Diamonds),
		c(cardcore.Jack, cardcore.Diamonds),
		c(cardcore.Queen, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Nine, cardcore.Spades)},
			{hearts.West, aceOfSpades},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != queenOfSpades {
		t.Errorf("expected Q♠ (dump under A♠ in trick), got %v", card)
	}
}

func TestFollowQueenOfSpadesAvoidsWithoutHigherSpade(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Nine, cardcore.Diamonds),
		c(cardcore.Ten, cardcore.Diamonds),
		c(cardcore.Jack, cardcore.Diamonds),
		c(cardcore.Queen, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Eight, cardcore.Spades)},
			{hearts.West, c(cardcore.Jack, cardcore.Spades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card == queenOfSpades {
		t.Errorf("expected non-Q♠ (no spade above queen to hide behind), got Q♠")
	}
}

func TestVoidDumpsQueenOfSpades(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Spades),
		queenOfSpades,
		aceOfSpades,
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Nine, cardcore.Hearts),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
		c(cardcore.Queen, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
		c(cardcore.Ace, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Clubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != queenOfSpades {
		t.Errorf("expected Q♠ (dump over A♠), got %v", card)
	}
}

func TestVoidDumpsAceOfSpades(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.Ace, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		aceOfSpades,
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Ace, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Clubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != aceOfSpades {
		t.Errorf("expected A♠ (dump high spade over A♥ and A♦), got %v", card)
	}
}

func TestVoidPrefersHeartsOverNonPenalty(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(cardcore.Six, cardcore.Diamonds),
		c(cardcore.Seven, cardcore.Diamonds),
		c(cardcore.Eight, cardcore.Diamonds),
		c(cardcore.King, cardcore.Diamonds),
		c(cardcore.Six, cardcore.Spades),
		c(cardcore.Seven, cardcore.Spades),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Nine, cardcore.Spades),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(cardcore.Six, cardcore.Clubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != c(cardcore.King, cardcore.Hearts) {
		t.Errorf("expected K♥ (hearts over same-rank non-penalty), got %v", card)
	}
}

// --- Moon blocking tests ---

// Moon threat active (East has all pts, trickNum=7). South follows spades.
// 10♠ would win (beats 6♠). Moon block: rank+30 = 38 vs 2♠ loser = 0.
// Without moon block, 10♠ would score -1 (not-last clean, danger penalty).
func TestFollowMoonBlockPrefersWinning(t *testing.T) {
	g := setupFollowState(hearts.South, 7, []cardcore.Card{
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Ten, cardcore.Spades),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(cardcore.Five, cardcore.Spades)},
			{hearts.East, c(cardcore.Six, cardcore.Spades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card != c(cardcore.Ten, cardcore.Spades) {
		t.Errorf("expected 10♠ (moon block prefers winning), got %v", card)
	}
}

// Same setup as TestFollowMoonBlockPrefersWinning but trickNum=5 (gate
// fails). Normal scoring: South ducks with 2♠ (losing card scores higher
// than winning under danger).
func TestFollowMoonBlockGateTrickNumTooLow(t *testing.T) {
	g := setupFollowState(hearts.South, 5, []cardcore.Card{
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Two, cardcore.Spades),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.Ten, cardcore.Spades),
	}, earlyMoonThreatHistory(),
		hearts.East,
		[]trickCard{
			{hearts.East, c(cardcore.Five, cardcore.Spades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card == c(cardcore.Ten, cardcore.Spades) {
		t.Errorf("expected duck (trickNum < 6, no moon block), got 10♠")
	}
}

// East is the moon threat and is following. Self-as-threat gate fails —
// East uses normal scoring, not moon blocking.
func TestFollowMoonBlockGateSelfIsThreat(t *testing.T) {
	g := setupFollowState(hearts.East, 7, []cardcore.Card{
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Four, cardcore.Spades),
		c(cardcore.Ten, cardcore.Spades),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(cardcore.Five, cardcore.Spades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card == c(cardcore.Ten, cardcore.Spades) {
		t.Errorf("expected normal play (self is threat, no block), got 10♠")
	}
}

// Moon threat active. South void in led suit (spades). East (moon
// threat) is currently winning. Dumping hearts feeds the shooter, so hearts
// get -10 instead of +10. Q♣ (rank 10 + short suit 15 = 25) beats 3♥
// (rank 1 - 10 = -9).
func TestVoidMoonBlockSuppressesHeartsDumpWhenThreatWins(t *testing.T) {
	g := setupFollowState(hearts.South, 7, []cardcore.Card{
		c(cardcore.Queen, cardcore.Clubs),
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(cardcore.Five, cardcore.Spades)},
			{hearts.East, c(cardcore.Ace, cardcore.Spades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card.Suit == cardcore.Hearts {
		t.Errorf("expected non-heart (suppress hearts dump when threat wins), got %v", card)
	}
}

// Moon threat active but threat (East) is NOT winning the trick. North is
// winning. South is void in led suit (clubs) and has a non-heart alternative
// (4♠). Hearts dump allowed: K♥ (+10 + 11 = 21) beats 4♠ (baseline 2 +
// short suit 15 = 17).
func TestVoidMoonBlockAllowsHeartsDumpWhenThreatLoses(t *testing.T) {
	g := setupFollowState(hearts.South, 7, []cardcore.Card{
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Nine, cardcore.Hearts),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
		c(cardcore.Four, cardcore.Spades),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(cardcore.Queen, cardcore.Clubs)},
			{hearts.East, c(cardcore.Six, cardcore.Spades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card != c(cardcore.King, cardcore.Hearts) {
		t.Errorf("expected K♥ (highest heart, dump allowed when threat loses), got %v", card)
	}
}

// trickNum=5, moon threat exists but gate fails (trickNum < 6). Normal
// scoring: hearts get +10 dump bonus. South void in led suit (diamonds)
// and has a non-heart alternative (4♠). K♥ (+10 + 11 = 21) beats 4♠
// (baseline 2 + short suit 15 = 17).
func TestVoidMoonBlockGateTrickNumTooLow(t *testing.T) {
	g := setupFollowState(hearts.South, 5, []cardcore.Card{
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Nine, cardcore.Hearts),
		c(cardcore.Ten, cardcore.Hearts),
		c(cardcore.Jack, cardcore.Hearts),
		c(cardcore.King, cardcore.Hearts),
		c(cardcore.Four, cardcore.Spades),
	}, earlyMoonThreatHistory(),
		hearts.East,
		[]trickCard{
			{hearts.East, c(cardcore.Jack, cardcore.Diamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card != c(cardcore.King, cardcore.Hearts) {
		t.Errorf("expected K♥ (gate fails, normal +10 dump, highest heart), got %v", card)
	}
}

// Moon threat active. North leads. A♥ gets +30 (high heart, win the trick
// to dash shooter's hopes). All hearts in hand — A♥ preferred over low
// hearts (+15).
func TestLeadMoonBlockPrefersHighHearts(t *testing.T) {
	g := setupLeadState(hearts.North, 7, []cardcore.Card{
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Seven, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Hearts),
		c(cardcore.Ace, cardcore.Hearts),
	}, moonThreatHistory())
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.North)

	if card != c(cardcore.Ace, cardcore.Hearts) {
		t.Errorf("expected A♥ (high heart moon block lead), got %v", card)
	}
}

// Moon threat active. North has only low hearts (no K♥/A♥) plus K♠ and A♠
// (penalized by flush -20). Low hearts get +15 from moon block, beating the
// penalized spades.
func TestLeadMoonBlockLowHeartsStillPreferred(t *testing.T) {
	g := setupLeadState(hearts.North, 7, []cardcore.Card{
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.King, cardcore.Spades),
		c(cardcore.Ace, cardcore.Spades),
	}, moonThreatHistory())
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.North)

	if card.Suit != cardcore.Hearts {
		t.Errorf("expected heart lead (moon block, +15 bonus), got %v", card)
	}
}

// No moon threat (penalties distributed to North and West). Hearts get
// normal -15 lead penalty. East should avoid leading hearts.
func TestLeadNoMoonThreatNormalHeartPenalty(t *testing.T) {
	cleanHistory := moonThreatHistory()
	cleanHistory = cleanHistory[:4]
	cleanHistory = append(cleanHistory,
		hearts.Trick{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Queen, cardcore.Diamonds),
			hearts.West:  c(cardcore.Seven, cardcore.Hearts),
			hearts.North: c(cardcore.Ace, cardcore.Diamonds),
			hearts.East:  c(cardcore.King, cardcore.Diamonds),
		}},
		hearts.Trick{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Two, cardcore.Hearts),
			hearts.West:  c(cardcore.Three, cardcore.Spades),
			hearts.North: c(cardcore.Two, cardcore.Spades),
			hearts.East:  c(cardcore.Ace, cardcore.Clubs),
		}},
		hearts.Trick{Leader: hearts.West, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Four, cardcore.Spades),
			hearts.West:  c(cardcore.Five, cardcore.Spades),
			hearts.North: c(cardcore.Six, cardcore.Spades),
			hearts.East:  c(cardcore.Seven, cardcore.Spades),
		}},
	)

	g := setupLeadState(hearts.East, 7, []cardcore.Card{
		c(cardcore.Three, cardcore.Hearts),
		c(cardcore.Four, cardcore.Hearts),
		c(cardcore.Five, cardcore.Hearts),
		c(cardcore.Six, cardcore.Hearts),
		c(cardcore.Eight, cardcore.Spades),
		c(cardcore.Nine, cardcore.Spades),
	}, cleanHistory)
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card.Suit == cardcore.Hearts {
		t.Errorf("expected non-heart (no moon threat, -15 penalty), got %v", card)
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

// setupLeadState creates a game in PhasePlay where seat is about to lead.
// trickNum sets how many tricks have been played. trickHistory provides
// completed tricks (may be nil).
func setupLeadState(seat hearts.Seat, trickNum int, hand []cardcore.Card, trickHistory []hearts.Trick) *hearts.Game {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.Turn = seat
	g.TrickNum = trickNum
	g.Trick = hearts.Trick{Leader: seat}
	g.Hands[seat] = cardcore.NewHand(hand)
	g.TrickHistory = trickHistory
	// Give other players dummy hands so the game is structurally valid.
	for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
		if s != seat {
			g.Hands[s] = cardcore.NewHand(nil)
		}
	}
	return g
}

// validFirstTrick returns a trick led with 2♣ by South for use in
// test trick histories that need a realistic first trick.
func validFirstTrick() hearts.Trick {
	return hearts.Trick{
		Leader: hearts.South,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: twoOfClubs,
			hearts.West:  c(cardcore.Three, cardcore.Clubs),
			hearts.North: c(cardcore.Four, cardcore.Clubs),
			hearts.East:  c(cardcore.Five, cardcore.Clubs),
		},
	}
}

type trickCard struct {
	seat hearts.Seat
	card cardcore.Card
}

func setupFollowState(
	seat hearts.Seat,
	trickNum int,
	hand []cardcore.Card,
	trickHistory []hearts.Trick,
	leader hearts.Seat,
	played []trickCard,
) *hearts.Game {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.Turn = seat
	g.TrickNum = trickNum
	g.TrickHistory = trickHistory
	g.Hands[seat] = cardcore.NewHand(hand)
	for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
		if s != seat {
			g.Hands[s] = cardcore.NewHand(nil)
		}
	}

	g.Trick = hearts.Trick{Leader: leader}
	for _, tc := range played {
		g.Trick.Cards[tc.seat] = tc.card
	}
	g.Trick.Count = len(played)
	return g
}

// earlyMoonThreatHistory returns 5 completed tricks where East wins all
// penalty points (2♥ in trick 4). For use with trickNum=5 gate tests.
//
// Cards used: 2♣–J♣, K♣, 3♦–10♦, 2♥ (20 cards).
func earlyMoonThreatHistory() []hearts.Trick {
	return []hearts.Trick{
		validFirstTrick(),
		// Trick 1: East leads clubs, North wins (clean).
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Seven, cardcore.Clubs),
			hearts.West:  c(cardcore.Eight, cardcore.Clubs),
			hearts.North: c(cardcore.Nine, cardcore.Clubs),
			hearts.East:  c(cardcore.Six, cardcore.Clubs),
		}},
		// Trick 2: North leads diamonds, East wins (clean).
		{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Four, cardcore.Diamonds),
			hearts.West:  c(cardcore.Five, cardcore.Diamonds),
			hearts.North: c(cardcore.Three, cardcore.Diamonds),
			hearts.East:  c(cardcore.Eight, cardcore.Diamonds),
		}},
		// Trick 3: East leads diamonds, North wins (clean).
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Seven, cardcore.Diamonds),
			hearts.West:  c(cardcore.Nine, cardcore.Diamonds),
			hearts.North: c(cardcore.Ten, cardcore.Diamonds),
			hearts.East:  c(cardcore.Six, cardcore.Diamonds),
		}},
		// Trick 4: North leads clubs, East wins. West sloughs 2♥ (1 pt → East).
		{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Jack, cardcore.Clubs),
			hearts.West:  c(cardcore.Two, cardcore.Hearts),
			hearts.North: c(cardcore.Ten, cardcore.Clubs),
			hearts.East:  c(cardcore.King, cardcore.Clubs),
		}},
	}
}

// moonThreatHistory returns 7 completed tricks where East wins all penalty
// points. Extends earlyMoonThreatHistory with 2 clean tricks.
//
// Cards used: 2♣–J♣, K♣, A♣, 2♦–A♦, 2♠, 3♠, 2♥ (28 cards).
// Available for hands: Q♣, spades (4♠–A♠), hearts (3♥–A♥).
func moonThreatHistory() []hearts.Trick {
	h := earlyMoonThreatHistory()
	return append(h,
		// Trick 5: East leads diamonds, North wins (clean).
		hearts.Trick{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Queen, cardcore.Diamonds),
			hearts.West:  c(cardcore.Jack, cardcore.Diamonds),
			hearts.North: c(cardcore.Ace, cardcore.Diamonds),
			hearts.East:  c(cardcore.King, cardcore.Diamonds),
		}},
		// Trick 6: North leads diamonds, North wins (clean).
		hearts.Trick{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Ace, cardcore.Clubs),
			hearts.West:  c(cardcore.Three, cardcore.Spades),
			hearts.North: c(cardcore.Two, cardcore.Diamonds),
			hearts.East:  c(cardcore.Two, cardcore.Spades),
		}},
	)
}
