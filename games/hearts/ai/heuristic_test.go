package ai

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

var _ hearts.Player = (*Heuristic)(nil)

type trickCard struct {
	seat hearts.Seat
	card cardcore.Card
}

// TestHeuristicChoosePassReturnsDistinctCardsFromHand verifies that ChoosePass
// returns three distinct cards that exist in the player's hand.
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

// TestHeuristicChoosePlayReturnsLegalCard verifies that ChoosePlay returns a
// card accepted by PlayCard.
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

// TestPassScoreUnprotectedQueen verifies that Q♠ scores 100 when fewer than 4
// low spades protect it. Hand has only 2 low spades (3♠, 5♠), so Q♠ is
// unprotected.
func TestPassScoreUnprotectedQueen(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rThree, sSpades),
		c(rFive, sSpades),
		queenOfSpades,
	})

	score := passScore(queenOfSpades, hand)
	if score != 100 {
		t.Errorf("unprotected Q♠ score = %d, want 100", score)
	}
}

// TestPassScoreProtectedQueen verifies that Q♠ scores 2 when 4+ low spades
// protect it. Hand has 4 low spades (2♠, 3♠, 5♠, 6♠), making Q♠
// an asset to keep.
func TestPassScoreProtectedQueen(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFive, sSpades),
		c(rSix, sSpades),
		queenOfSpades,
	})

	score := passScore(queenOfSpades, hand)
	if score != 2 {
		t.Errorf("protected Q♠ score = %d, want 2", score)
	}
}

// TestPassScoreHighSpades verifies that A♠ and K♠ receive a +20 bonus when
// Q♠ is not in hand (unprotected spade context). Both should outscore a low
// club.
func TestPassScoreHighSpades(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rTwo, sDiamonds),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		kingOfSpades,
		aceOfSpades,
	})

	aceScore := passScore(aceOfSpades, hand)
	kingScore := passScore(kingOfSpades, hand)
	lowClubScore := passScore(c(rFive, sClubs), hand)

	if aceScore <= lowClubScore {
		t.Errorf("A♠ score (%d) should exceed low club score (%d)", aceScore, lowClubScore)
	}
	if kingScore <= lowClubScore {
		t.Errorf("K♠ score (%d) should exceed low club score (%d)", kingScore, lowClubScore)
	}
}

// TestPassScoreHeartsBonus verifies that hearts receive a +10 bonus. A♥
// (rank 12 + hearts 10 = 22) should outscore A♦ (rank 12, no bonus).
func TestPassScoreHeartsBonus(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rAce, sDiamonds),
		c(rAce, sHearts),
	})

	heartScore := passScore(c(rAce, sHearts), hand)
	diamondScore := passScore(c(rAce, sDiamonds), hand)

	if heartScore <= diamondScore {
		t.Errorf("A♥ score (%d) should exceed A♦ score (%d)", heartScore, diamondScore)
	}
}

// TestPassScoreShortSuitBonus verifies that singletons get a short-suit bonus
// of (4-1)*5 = 15. Singleton A♦ (rank 12 + short 15 = 27) outscores long-suit
// K♣ (rank 11).
func TestPassScoreShortSuitBonus(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sDiamonds),
	})

	singletonScore := passScore(c(rAce, sDiamonds), hand)
	longSuitScore := passScore(c(rKing, sClubs), hand)

	if singletonScore <= longSuitScore {
		t.Errorf("singleton A♦ score (%d) should exceed long-suit K♣ score (%d)", singletonScore, longSuitScore)
	}
}

// TestChoosePassPassesUnprotectedQueen verifies that ChoosePass includes Q♠
// when it is unprotected (only 1 low spade: 2♠).
func TestChoosePassPassesUnprotectedQueen(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rTwo, sDiamonds),
		c(rTwo, sSpades),
		queenOfSpades,
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, queenOfSpades) {
		t.Errorf("expected unprotected Q♠ to be passed, got %v", cards)
	}
}

// TestChoosePassKeepsProtectedQueen verifies that ChoosePass keeps Q♠ when 5+
// low spades protect it. Hand: 5 low spades + Q♠ + 7 hearts.
func TestChoosePassKeepsProtectedQueen(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		c(rFive, sSpades),
		c(rSix, sSpades),
		queenOfSpades,
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if passedContains(cards, queenOfSpades) {
		t.Errorf("expected protected Q♠ to be kept, but it was passed: %v", cards)
	}
}

// TestChoosePassKeepsHighSpadesWithProtectedQueen verifies that A♠ and
// K♠ are kept when Q♠ is protected. No +20 bonus applies when queen is
// safe.
func TestChoosePassKeepsHighSpadesWithProtectedQueen(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		c(rFive, sSpades),
		c(rSix, sSpades),
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

// TestChoosePassPrefersHighSpades verifies that A♠ and K♠ are passed
// when Q♠ is absent and spades are unprotected. The +20 high-spade
// bonus makes them top pass candidates.
func TestChoosePassPrefersHighSpades(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rKing, sDiamonds),
		c(rAce, sDiamonds),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		c(rFive, sSpades),
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

// TestChoosePassVoidsShortSuit verifies that a singleton in a non-club suit is
// passed to create a void. Singleton 7♦ gets short-suit bonus (4-1)*5 = 15.
func TestChoosePassVoidsShortSuit(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rSeven, sDiamonds),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, c(rSeven, sDiamonds)) {
		t.Errorf("expected singleton 7♦ to be passed, got %v", cards)
	}
}

// TestChoosePassVoidsTwoShortSuits verifies that two singletons
// (3♦, 2♥) are both passed when the hand has two short suits.
func TestChoosePassVoidsTwoShortSuits(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rThree, sDiamonds),
		c(rTwo, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	if !passedContains(cards, c(rThree, sDiamonds)) {
		t.Errorf("expected singleton 3♦ to be passed, got %v", cards)
	}
	if !passedContains(cards, c(rTwo, sHearts)) {
		t.Errorf("expected singleton 2♥ to be passed, got %v", cards)
	}
}

// TestChoosePassTieBreaking verifies that RNG-based shuffle produces varied
// selections among tied-score cards across different seeds.
func TestChoosePassTieBreaking(t *testing.T) {
	// 5 clubs + 4 diamonds + 4 spades (all low): 6♣ is the unique
	// highest (score 4), but 5♣, 5♦, 5♠ all tie at score 3. Only 2 of
	// the 3 tied cards can be passed, so the rng should vary which pair.
	hand := []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rTwo, sDiamonds),
		c(rThree, sDiamonds),
		c(rFour, sDiamonds),
		c(rFive, sDiamonds),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		c(rFive, sSpades),
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

// TestLeadScorePrefersLowFromLongSuit verifies that low cards are
// preferred from long suits. 6♣ (from 5-card club suit) outscores A♣
// because long suits prefer safe low leads.
func TestLeadScorePrefersLowFromLongSuit(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rEight, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rAce, sClubs),
		c(rThree, sDiamonds),
		c(rSix, sDiamonds),
		c(rNine, sDiamonds),
		c(rKing, sDiamonds),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rNine, sHearts),
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	sixScore := leadScore(c(rSix, sClubs), g, a)
	aceScore := leadScore(c(rAce, sClubs), g, a)

	if sixScore <= aceScore {
		t.Errorf("6♣ lead score (%d) should exceed A♣ lead score (%d) from long suit", sixScore, aceScore)
	}
}

// TestLeadScorePrefersHighFromShortSuitEarly verifies that high cards are
// preferred from
// short suits in early tricks (trickNum ≤ 3). K♦ (from 3-card diamond suit)
// outscores
// 3♦ to void the suit quickly.
func TestLeadScorePrefersHighFromShortSuitEarly(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rThree, sDiamonds),
		c(rSix, sDiamonds),
		c(rKing, sDiamonds),
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	kingScore := leadScore(c(rKing, sDiamonds), g, a)
	threeScore := leadScore(c(rThree, sDiamonds), g, a)

	if kingScore <= threeScore {
		t.Errorf("K♦ lead score (%d) should exceed 3♦ lead score (%d) from short suit early", kingScore, threeScore)
	}
}

// TestLeadScorePrefersLowFromShortSuitLate verifies that low cards are
// preferred from short suits in late-game leads. TrickHistory is nil for
// brevity — only g.TrickNum matters for the early/late threshold.
func TestLeadScorePrefersLowFromShortSuitLate(t *testing.T) {
	g := setupLeadState(hearts.South, 7, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rSix, sDiamonds),
		c(rKing, sDiamonds),
	}, nil)

	a := analyze(g, hearts.South)
	sixScore := leadScore(c(rSix, sDiamonds), g, a)
	kingScore := leadScore(c(rKing, sDiamonds), g, a)

	if sixScore <= kingScore {
		t.Errorf("6♦ lead score (%d) should exceed K♦ score (%d) from short suit late game", sixScore, kingScore)
	}
}

// TestLeadScoreAvoidsOpponentVoidSuit verifies the -40 opponent-void penalty.
// East is known void in diamonds from trick history. Both K♦ and 6♦ score
// below 10♣ because of the void penalty.
func TestLeadScoreAvoidsOpponentVoidSuit(t *testing.T) {
	// East is known void in diamonds from trick history.
	trickHistory := []hearts.Trick{
		validFirstTrick(),
		{
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rThree, sDiamonds),
				hearts.West:  c(rFour, sDiamonds),
				hearts.North: c(rFive, sDiamonds),
				hearts.East:  c(rTwo, sHearts), // void in diamonds
			},
		},
	}

	g := setupLeadState(hearts.South, 2, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rSix, sDiamonds),
		c(rKing, sDiamonds),
	}, trickHistory)
	g.HeartsBroken = true

	a := analyze(g, hearts.South)
	kingDiamondScore := leadScore(c(rKing, sDiamonds), g, a)
	clubScore := leadScore(c(rTen, sClubs), g, a)
	sixDiamondScore := leadScore(c(rSix, sDiamonds), g, a)

	if kingDiamondScore >= clubScore {
		t.Errorf("K♦ lead score (%d) should be less than 10♣ lead score (%d) when opponent void in diamonds",
			kingDiamondScore, clubScore)
	}
	if sixDiamondScore >= clubScore {
		t.Errorf("6♦ lead score (%d) should be less than 10♣ lead score (%d) — lower diamond still penalized by void",
			sixDiamondScore, clubScore)
	}
}

// TestLeadScoreSafeWhenGuaranteedLowest verifies that the guaranteed-lowest
// exemption overrides the opponent-void penalty. 4♦ is the lowest remaining
// diamond (2♦, 3♦ already played) so the -40 penalty is skipped.
func TestLeadScoreSafeWhenGuaranteedLowest(t *testing.T) {
	// 2♦ and 3♦ already played, so 4♦ is guaranteed lowest diamond.
	// Leading 4♦ is safe even if opponent is void in diamonds.
	trickHistory := []hearts.Trick{
		validFirstTrick(),
		{
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rTwo, sDiamonds),
				hearts.West:  c(rThree, sDiamonds),
				hearts.North: c(rFive, sDiamonds),
				hearts.East:  c(rTwo, sHearts), // void in diamonds
			},
		},
	}

	g := setupLeadState(hearts.South, 2, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rFour, sDiamonds),
		c(rSeven, sHearts),
	}, trickHistory)
	g.HeartsBroken = true

	a := analyze(g, hearts.South)
	fourDiamondScore := leadScore(c(rFour, sDiamonds), g, a)
	clubScore := leadScore(c(rSix, sClubs), g, a)

	if fourDiamondScore < clubScore {
		t.Errorf("guaranteed lowest 4♦ lead score (%d) should be >= 6♣ score (%d) despite opponent void", fourDiamondScore, clubScore)
	}
}

// TestLeadScoreUnprotectedQueenUrgency verifies the void-urgency bonus for
// short non-spade
// suits when Q♠ is unprotected. Singleton K♦ gets a bonus to accelerate
// voiding.
func TestLeadScoreUnprotectedQueenUrgency(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rKing, sDiamonds),
		c(rSix, sSpades),
		queenOfSpades,
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	// K♦ is a singleton non-spade — should get Q♠ urgency bonus.
	kingDiamondScore := leadScore(c(rKing, sDiamonds), g, a)
	lowClubScore := leadScore(c(rSix, sClubs), g, a)

	if kingDiamondScore <= lowClubScore {
		t.Errorf("K♦ lead score (%d) should exceed 2♣ score (%d) with unprotected Q♠ urgency",
			kingDiamondScore, lowClubScore)
	}
}

// TestLeadScoreAvoidsSpadesWithHighSpades verifies the -20
// spade-flush penalty when holding A♠ or K♠. Leading 6♠ risks pulling
// Q♠ onto the player's own high spades.
func TestLeadScoreAvoidsSpadesWithHighSpades(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rSix, sSpades),
		c(rEight, sSpades),
		kingOfSpades,
		aceOfSpades,
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	spadeScore := leadScore(c(rSix, sSpades), g, a)
	clubScore := leadScore(c(rSix, sClubs), g, a)

	if spadeScore >= clubScore {
		t.Errorf("6♠ lead score (%d) should be less than 6♣ score (%d) when holding A♠/K♠",
			spadeScore, clubScore)
	}
}

// TestLeadScoreFlushesQueenWithoutHighSpades verifies that spade leads
// are preferred when Q♠ is not in hand and no A♠/K♠ are held.
// J♠ (highest below Q♠) is preferred to flush Q♠ from opponents.
func TestLeadScoreFlushesQueenWithoutHighSpades(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rSix, sSpades),
		c(rEight, sSpades),
		c(rJack, sSpades),
	}, []hearts.Trick{validFirstTrick()})

	a := analyze(g, hearts.South)
	jackSpadeScore := leadScore(c(rJack, sSpades), g, a)
	sixSpadeScore := leadScore(c(rSix, sSpades), g, a)
	clubScore := leadScore(c(rJack, sClubs), g, a)

	if jackSpadeScore <= clubScore {
		t.Errorf("J♠ flush score (%d) should exceed J♣ score (%d) when flushing Q♠ safely",
			jackSpadeScore, clubScore)
	}
	if jackSpadeScore <= sixSpadeScore {
		t.Errorf("J♠ flush score (%d) should exceed 6♠ score (%d) — prefer highest below Q♠",
			jackSpadeScore, sixSpadeScore)
	}
}

// TestChooseLeadAvoidsHearts verifies the -15 heart lead penalty. Even with
// hearts broken, the AI prefers non-heart leads when alternatives exist.
func TestChooseLeadAvoidsHearts(t *testing.T) {
	g := setupLeadState(hearts.South, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rQueen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rSix, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
	}, []hearts.Trick{validFirstTrick()})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card.Suit == sHearts {
		t.Errorf("expected non-heart lead, got %v", card)
	}
}

// TestFollowLastCleanTrickWinsWithHighest verifies the last-player clean-trick
// branch. East plays last into a clean club trick. J♣ (highest club, wins
// trick)
// is preferred: score = rank - danger*2.
func TestFollowLastCleanTrickWinsWithHighest(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rTen, sClubs),
		c(rJack, sClubs),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rNine, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
		c(rNine, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSeven, sClubs)},
			{hearts.West, c(rEight, sClubs)},
			{hearts.North, c(rNine, sClubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(rJack, sClubs) {
		t.Errorf("expected J♣ (highest to win clean trick), got %v", card)
	}
}

// TestFollowLastTrickHasPointsShedsHighest verifies the last-player forced-win
// branch when points are present. East must win (all cards beat the current
// winner). Forced to win → shed highest: J♥ (rank 9).
func TestFollowLastTrickHasPointsShedsHighest(t *testing.T) {
	g := setupFollowState(hearts.East, 2, []cardcore.Card{
		c(rSix, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rNine, sDiamonds),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
		c(rNine, sSpades),
	}, []hearts.Trick{validFirstTrick(), {
		Leader: hearts.South,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rSix, sClubs),
			hearts.West:  c(rKing, sClubs),
			hearts.North: c(rThree, sHearts),
			hearts.East:  c(rAce, sClubs),
		},
	}},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sHearts)},
			{hearts.West, c(rSeven, sHearts)},
			{hearts.North, c(rEight, sHearts)},
		})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(rJack, sHearts) {
		t.Errorf("expected J♥ (forced to win, shed highest), got %v", card)
	}
}

// TestFollowLastTrickHasPointsPrefersDuck verifies that the highest
// losing card is preferred when the trick contains penalty points.
// East has 6♣, 8♣ (lose to J♣) and K♣ (wins). North sloughed 5♥ so
// trickPts=1. Losers get +50 bonus: 8♣ scores 56, K♣ scores 11.
func TestFollowLastTrickHasPointsPrefersDuck(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rEight, sClubs),
		c(rKing, sClubs),
		c(rSix, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rNine, sDiamonds),
		c(rSix, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
		c(rNine, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rTen, sClubs)},
			{hearts.West, c(rJack, sClubs)},
			{hearts.North, c(rFive, sHearts)},
		})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(rEight, sClubs) {
		t.Errorf("expected 8♣ (highest loser, +50 duck bonus over K♣), got %v", card)
	}
}

// TestFollowLastCleanTrickWins verifies that the highest winning card
// is preferred when the trick is clean and we play last. East is
// last, trick is clean. Both 9♦ and K♦ win; K♦ preferred (higher
// rank). 3♦ loses (score 1) and should rank below winners.
func TestFollowLastCleanTrickWins(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rThree, sDiamonds),
		c(rNine, sDiamonds),
		c(rKing, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sDiamonds)},
			{hearts.West, c(rSeven, sDiamonds)},
			{hearts.North, c(rEight, sDiamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(rKing, sDiamonds) {
		t.Errorf("expected K♦ (win clean trick, higher rank preferred), got %v", card)
	}
}

// TestFollowLastCleanHighDangerPrefersDuck verifies that high hand
// danger discourages winning a clean trick when playing last. East is
// last, trick is clean, hand is almost all Ten+. highCardRatio =
// 10*10/12 = 8, so danger*2 = 16. 9♦ wins (score 7-16 = -9), 3♦ loses
// (score 1). High danger makes losing preferable over winning.
func TestFollowLastCleanHighDangerPrefersDuck(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(rTen, sClubs),
		c(rThree, sDiamonds),
		c(rNine, sDiamonds),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTen, sSpades),
		c(rJack, sSpades),
		c(rKing, sSpades),
		c(rAce, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sDiamonds)},
			{hearts.West, c(rSeven, sDiamonds)},
			{hearts.North, c(rEight, sDiamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != c(rThree, sDiamonds) {
		t.Errorf("expected 3♦ (high danger discourages winning clean trick), got %v", card)
	}
}

// TestFollowNotLastPointsDucksLowestWinner verifies that the lowest
// winning card is preferred when not last and the trick has points.
// North is not last (East still to play). West sloughed 5♥ (1 pt). All
// of North's diamonds win; prefers lowest winner 8♦ (Ace-Rank = 6).
func TestFollowNotLastPointsDucksLowestWinner(t *testing.T) {
	g := setupFollowState(hearts.North, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rEight, sDiamonds),
		c(rJack, sDiamonds),
		c(rKing, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sDiamonds)},
			{hearts.West, c(rFive, sHearts)},
		})
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.North)

	if card != c(rEight, sDiamonds) {
		t.Errorf("expected 8♦ (lowest winner to duck points), got %v", card)
	}
}

// TestFollowNotLastCleanWins verifies that the highest winning card is
// preferred when
// not last and the trick is clean. West is not last (Count=1, playersLeft=2).
// Clean
// trick. Both 8♦ and K♦ win; K♦ preferred (higher bonus after playersLeft
// penalty).
func TestFollowNotLastCleanWins(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rTen, sClubs),
		c(rEight, sDiamonds),
		c(rKing, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sDiamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != c(rKing, sDiamonds) {
		t.Errorf("expected K♦ (win clean trick, higher rank preferred), got %v", card)
	}
}

// TestFollowUnderWinnerShedsHighest verifies that the highest losing card is
// preferred when playing under the current winner.
func TestFollowUnderWinnerShedsHighest(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(rSix, sClubs),
		c(rEight, sClubs),
		c(rJack, sClubs),
		c(rKing, sClubs),
		c(rNine, sDiamonds),
		c(rTen, sDiamonds),
		c(rJack, sDiamonds),
		c(rQueen, sDiamonds),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
		c(rNine, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rTen, sClubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != c(rEight, sClubs) {
		t.Errorf("expected 8♣ (highest that still loses to 10♣), got %v", card)
	}
}

// TestFollowQueenOfSpadesDumpsUnderHigherSpade verifies the Q♠
// early-return in followScore. When A♠ is in the trick, Q♠ gets
// score 200 (dump it safely under a higher spade).
func TestFollowQueenOfSpadesDumpsUnderHigherSpade(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(rSix, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rNine, sDiamonds),
		c(rTen, sDiamonds),
		c(rJack, sDiamonds),
		c(rQueen, sDiamonds),
		c(rKing, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rSix, sSpades),
		queenOfSpades,
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rNine, sSpades)},
			{hearts.West, aceOfSpades},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card != queenOfSpades {
		t.Errorf("expected Q♠ (dump under A♠ in trick), got %v", card)
	}
}

// TestFollowQueenOfSpadesAvoidsWithoutHigherSpade verifies the Q♠
// penalty when no higher spade is in the trick. Q♠ gets -100, making
// the AI avoid playing it.
func TestFollowQueenOfSpadesAvoidsWithoutHigherSpade(t *testing.T) {
	g := setupFollowState(hearts.East, 1, []cardcore.Card{
		c(rSix, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rNine, sDiamonds),
		c(rTen, sDiamonds),
		c(rJack, sDiamonds),
		c(rQueen, sDiamonds),
		c(rKing, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rSix, sSpades),
		queenOfSpades,
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rEight, sSpades)},
			{hearts.West, c(rJack, sSpades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card == queenOfSpades {
		t.Errorf("expected non-Q♠ (no spade above queen to hide behind), got Q♠")
	}
}

// TestVoidDumpsQueenOfSpades verifies the Q♠ priority in voidScore.
// Q♠ always scores +100, making it the first card dumped when void in
// the led suit.
func TestVoidDumpsQueenOfSpades(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(rTwo, sDiamonds),
		c(rThree, sDiamonds),
		c(rFour, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rSix, sSpades),
		queenOfSpades,
		aceOfSpades,
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sClubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != queenOfSpades {
		t.Errorf("expected Q♠ (dump over A♠), got %v", card)
	}
}

// TestVoidDumpsAceOfSpades verifies the A♠/K♠ priority in voidScore.
// A♠ scores +50, outranking hearts and non-penalty cards.
func TestVoidDumpsAceOfSpades(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(rSix, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rAce, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rAce, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
		aceOfSpades,
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sClubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != aceOfSpades {
		t.Errorf("expected A♠ (dump high spade over A♥ and A♦), got %v", card)
	}
}

// TestVoidPrefersHeartsOverNonPenalty verifies the hearts dump bonus
// in voidScore. K♥ (+10 + rank 11 = 21) outscores K♦ (baseline
// rank 11) because hearts carry a +10 dump bonus.
func TestVoidPrefersHeartsOverNonPenalty(t *testing.T) {
	g := setupFollowState(hearts.West, 1, []cardcore.Card{
		c(rSix, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rKing, sDiamonds),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rKing, sHearts),
		c(rSix, sSpades),
		c(rSeven, sSpades),
		c(rEight, sSpades),
		c(rNine, sSpades),
	}, []hearts.Trick{validFirstTrick()},
		hearts.South,
		[]trickCard{
			{hearts.South, c(rSix, sClubs)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.West)

	if card != c(rKing, sHearts) {
		t.Errorf("expected K♥ (hearts over same-rank non-penalty), got %v", card)
	}
}

// --- Moon blocking tests ---

// TestFollowMoonBlockPrefersWinning verifies that the moon-block
// heuristic prefers winning the trick to deny the shooter lead
// control. Moon threat active (East has all pts, trickNum=7). South
// follows spades. 10♠ would win (beats 6♠). Moon block: rank+30 = 38
// vs 2♠ loser = 0. Without moon block, 10♠ would score -1 (not-last
// clean, danger penalty).
func TestFollowMoonBlockPrefersWinning(t *testing.T) {
	g := setupFollowState(hearts.South, 7, []cardcore.Card{
		c(rQueen, sClubs),
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rTwo, sSpades),
		c(rTen, sSpades),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(rFive, sSpades)},
			{hearts.East, c(rSix, sSpades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card != c(rTen, sSpades) {
		t.Errorf("expected 10♠ (moon block prefers winning), got %v", card)
	}
}

// TestFollowMoonBlockGateTrickNumTooLow verifies that the moon-block heuristic
// does
// not activate before trick 6. Same setup as TestFollowMoonBlockPrefersWinning
// but
// trickNum=5 (gate fails). Normal scoring: South ducks with 2♠.
func TestFollowMoonBlockGateTrickNumTooLow(t *testing.T) {
	g := setupFollowState(hearts.South, 5, []cardcore.Card{
		c(rQueen, sClubs),
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rTwo, sSpades),
		c(rFour, sSpades),
		c(rTen, sSpades),
	}, earlyMoonThreatHistory(),
		hearts.East,
		[]trickCard{
			{hearts.East, c(rFive, sSpades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card == c(rTen, sSpades) {
		t.Errorf("expected duck (trickNum < 6, no moon block), got 10♠")
	}
}

// TestFollowMoonBlockGateSelfIsThreat verifies that the moon-block heuristic
// does not
// activate when the player is the threat. East is the moon threat and is
// following —
// self-as-threat gate fails, normal scoring used.
func TestFollowMoonBlockGateSelfIsThreat(t *testing.T) {
	g := setupFollowState(hearts.East, 7, []cardcore.Card{
		c(rQueen, sClubs),
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rFour, sSpades),
		c(rTen, sSpades),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(rFive, sSpades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card == c(rTen, sSpades) {
		t.Errorf("expected normal play (self is threat, no block), got 10♠")
	}
}

// TestVoidMoonBlockSuppressesHeartsDumpWhenThreatWins verifies that
// hearts are suppressed when void and the moon-threat seat is winning.
// Moon threat active. South void in led suit (spades). East (moon
// threat) is currently winning. Dumping hearts feeds the shooter, so
// hearts get -10 instead of +10. Q♣ (rank 10 + short suit 15 = 25)
// beats 3♥ (rank 1 - 10 = -9).
func TestVoidMoonBlockSuppressesHeartsDumpWhenThreatWins(t *testing.T) {
	g := setupFollowState(hearts.South, 7, []cardcore.Card{
		c(rQueen, sClubs),
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rSeven, sHearts),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(rFive, sSpades)},
			{hearts.East, c(rAce, sSpades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card.Suit == sHearts {
		t.Errorf("expected non-heart (suppress hearts dump when threat wins), got %v", card)
	}
}

// TestVoidMoonBlockAllowsHeartsDumpWhenThreatLoses verifies that
// hearts dumping is allowed when the moon-threat seat is not winning.
// Moon threat active but threat (East) is NOT winning the trick.
// North is winning. South is void in led suit (clubs) and has a
// non-heart alternative (4♠). Hearts dump allowed: K♥ (+10 + 11 = 21)
// beats 4♠ (baseline 2 + short suit 15 = 17).
func TestVoidMoonBlockAllowsHeartsDumpWhenThreatLoses(t *testing.T) {
	g := setupFollowState(hearts.South, 7, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rKing, sHearts),
		c(rFour, sSpades),
	}, moonThreatHistory(),
		hearts.North,
		[]trickCard{
			{hearts.North, c(rQueen, sClubs)},
			{hearts.East, c(rSix, sSpades)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card != c(rKing, sHearts) {
		t.Errorf("expected K♥ (highest heart, dump allowed when threat loses), got %v", card)
	}
}

// TestVoidMoonBlockGateTrickNumTooLow verifies that the void
// moon-block heuristic does not activate before trick 6. trickNum=5,
// moon threat exists but gate fails. Normal scoring: hearts get +10
// dump bonus. South void in diamonds, K♥ (+10 + 11 = 21) beats 4♠
// (baseline 2 + short suit 15 = 17).
func TestVoidMoonBlockGateTrickNumTooLow(t *testing.T) {
	g := setupFollowState(hearts.South, 5, []cardcore.Card{
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rKing, sHearts),
		c(rFour, sSpades),
	}, earlyMoonThreatHistory(),
		hearts.East,
		[]trickCard{
			{hearts.East, c(rJack, sDiamonds)},
		})

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.South)

	if card != c(rKing, sHearts) {
		t.Errorf("expected K♥ (gate fails, normal +10 dump, highest heart), got %v", card)
	}
}

// TestLeadMoonBlockPrefersHighHearts verifies that high hearts are preferred
// when
// leading under an active moon threat. A♥ gets +30 (high heart, win the trick
// to
// dash shooter's hopes). All hearts in hand — A♥ preferred over low hearts
// (+15).
func TestLeadMoonBlockPrefersHighHearts(t *testing.T) {
	g := setupLeadState(hearts.North, 7, []cardcore.Card{
		c(rThree, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rAce, sHearts),
	}, moonThreatHistory())
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.North)

	if card != c(rAce, sHearts) {
		t.Errorf("expected A♥ (high heart moon block lead), got %v", card)
	}
}

// TestLeadMoonBlockLowHeartsStillPreferred verifies that even low
// hearts are preferred over penalized spades under a moon threat.
// North has only low hearts (no K♥/A♥) plus K♠ and A♠ (penalized by
// flush -20). Low hearts get +15 from moon block, beating the
// penalized spades.
func TestLeadMoonBlockLowHeartsStillPreferred(t *testing.T) {
	g := setupLeadState(hearts.North, 7, []cardcore.Card{
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rKing, sSpades),
		c(rAce, sSpades),
	}, moonThreatHistory())
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.North)

	if card.Suit != sHearts {
		t.Errorf("expected heart lead (moon block, +15 bonus), got %v", card)
	}
}

// TestLeadNoMoonThreatNormalHeartPenalty verifies that hearts are penalized for
// leading when no moon threat exists. No moon threat (penalties distributed to
// North and West). Hearts get normal -15 lead penalty.
func TestLeadNoMoonThreatNormalHeartPenalty(t *testing.T) {
	cleanHistory := moonThreatHistory()
	cleanHistory = cleanHistory[:4]
	cleanHistory = append(cleanHistory,
		hearts.Trick{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rQueen, sDiamonds),
			hearts.West:  c(rSeven, sHearts),
			hearts.North: c(rAce, sDiamonds),
			hearts.East:  c(rKing, sDiamonds),
		}},
		hearts.Trick{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rTwo, sHearts),
			hearts.West:  c(rThree, sSpades),
			hearts.North: c(rTwo, sSpades),
			hearts.East:  c(rAce, sClubs),
		}},
		hearts.Trick{Leader: hearts.West, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFour, sSpades),
			hearts.West:  c(rFive, sSpades),
			hearts.North: c(rSix, sSpades),
			hearts.East:  c(rSeven, sSpades),
		}},
	)

	g := setupLeadState(hearts.East, 7, []cardcore.Card{
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rEight, sSpades),
		c(rNine, sSpades),
	}, cleanHistory)
	g.HeartsBroken = true

	h := newSeededHeuristic(42)
	card := h.ChoosePlay(g, hearts.East)

	if card.Suit == sHearts {
		t.Errorf("expected non-heart (no moon threat, -15 penalty), got %v", card)
	}
}

// --- Moon shooting tests ---

// TestShootPassScoreKeepsHearts verifies that shootPassScore ranks hearts
// far below non-penalty cards, preventing them from being passed. A♥ (-100)
// scores far below 2♣ (Ace - 0 = 12).
func TestShootPassScoreKeepsHearts(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rAce, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	})

	heartScore := shootPassScore(c(rAce, sHearts), hand)
	clubScore := shootPassScore(c(rTwo, sClubs), hand)

	if heartScore >= clubScore {
		t.Errorf("A♥ shoot pass score (%d) should be less than 2♣ score (%d)", heartScore, clubScore)
	}
}

// TestShootPassScoreKeepsQueenOfSpades verifies that shootPassScore ranks Q♠
// (-100) far below any non-penalty card.
func TestShootPassScoreKeepsQueenOfSpades(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rAce, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		queenOfSpades,
	})

	queenScore := shootPassScore(queenOfSpades, hand)
	clubScore := shootPassScore(c(rTwo, sClubs), hand)

	if queenScore >= clubScore {
		t.Errorf("Q♠ shoot pass score (%d) should be less than 2♣ score (%d)", queenScore, clubScore)
	}
}

// TestShootPassScorePassesLowCardsFirst verifies that shootPassScore prefers
// passing low non-heart cards. 2♣ (Ace - 0 = 12) outscores K♣ (Ace - 11 = 1).
func TestShootPassScorePassesLowCardsFirst(t *testing.T) {
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
	})

	twoScore := shootPassScore(c(rTwo, sClubs), hand)
	kingScore := shootPassScore(c(rKing, sClubs), hand)

	if twoScore <= kingScore {
		t.Errorf("2♣ shoot pass score (%d) should exceed K♣ score (%d)", twoScore, kingScore)
	}
}

// TestChoosePassShootKeepsHearts verifies that ChoosePass keeps all
// hearts when the hand triggers considerShoot. Hand: A♥ K♥ Q♥ J♥ 10♥
// 9♥ 8♥ + A♣ + 5 low clubs + 2♠. All 3 passed cards should be
// non-hearts.
//
// Why this works: normal passScore gives hearts a +10 bonus (eagerly passes
// them), so without shoot mode the heuristic would happily pass high hearts.
// Shoot mode's shootPassScore returns -100 for hearts, completely inverting
// that preference. The test proves shoot vs non-shoot produce opposite
// behavior for hearts.
func TestChoosePassShootKeepsHearts(t *testing.T) {
	g := hearts.New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	setupPassHand(t, g, hearts.South, []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rAce, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	})

	h := newSeededHeuristic(42)
	cards := h.ChoosePass(g, hearts.South)

	for _, card := range cards {
		if card.Suit == sHearts {
			t.Errorf("expected no hearts passed when shooting, but passed %v (all 3: %v)", card, cards)
			break
		}
	}
}

// TestShootLeadScorePrefersSideAcesOverHearts verifies that non-heart aces
// outscore hearts when shooting. A♣ (40) exceeds A♥ (12).
func TestShootLeadScorePrefersSideAcesOverHearts(t *testing.T) {
	g := setupLeadState(hearts.South, 2, []cardcore.Card{
		c(rAce, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	})

	a := analyze(g, hearts.South)
	a.shootActive = true

	aceClubScore := shootLeadScore(c(rAce, sClubs), g, a)
	aceHeartScore := shootLeadScore(c(rAce, sHearts), g, a)

	if aceClubScore <= aceHeartScore {
		t.Errorf("A♣ shoot lead score (%d) should exceed A♥ score (%d)", aceClubScore, aceHeartScore)
	}
}

// TestShootLeadScoreAvoidsQueenOfSpades verifies that Q♠ scores negative when
// leading and shooting. Leading Q♠ signals intent and risks losing the card.
func TestShootLeadScoreAvoidsQueenOfSpades(t *testing.T) {
	g := setupLeadState(hearts.South, 2, []cardcore.Card{
		c(rTwo, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		queenOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	})

	a := analyze(g, hearts.South)
	a.shootActive = true

	queenScore := shootLeadScore(queenOfSpades, g, a)

	if queenScore >= 0 {
		t.Errorf("Q♠ shoot lead score (%d) should be negative", queenScore)
	}
}

// TestShootFollowScorePrefersWinning verifies that winning cards
// outscore losing cards when shooting. South follows diamonds. K♦
// wins (rank 11 + 30 = 41), 3♦ loses (Ace - 1 = 11).
func TestShootFollowScorePrefersWinning(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rThree, sDiamonds),
		c(rKing, sDiamonds),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rSix, sDiamonds)},
			{hearts.West, c(rSeven, sDiamonds)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	kingScore := shootFollowScore(c(rKing, sDiamonds), g, a)
	threeScore := shootFollowScore(c(rThree, sDiamonds), g, a)

	if kingScore <= threeScore {
		t.Errorf("K♦ shoot follow score (%d) should exceed 3♦ score (%d)", kingScore, threeScore)
	}
}

// TestShootFollowScoreQueenWouldWinPlayed verifies that Q♠ scores
// 100 when it would win the trick and no K♠/A♠ in hand. South
// follows spades, Q♠ beats 8♠.
func TestShootFollowScoreQueenWouldWinPlayed(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		queenOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rEight, sSpades)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	queenScore := shootFollowScore(queenOfSpades, g, a)

	if queenScore != 100 {
		t.Errorf("Q♠ shoot follow score = %d, want 100 (would win, no K♠/A♠)", queenScore)
	}
}

// TestShootFollowScoreQueenDeferredForHigherSpade verifies that Q♠
// scores 10 (deferred) when it would win but K♠ or A♠ is in hand.
// The higher spade is preferred to preserve Q♠ optionality.
// Q♠ == 10 proves the defer branch (not the would-win branch at 100
// or would-lose branch at -50).
// Branch: shootFollowScore Q♠ would win, K♠/A♠ in hand → 10.
func TestShootFollowScoreQueenDeferredForHigherSpade(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		queenOfSpades,
		kingOfSpades,
		aceOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rEight, sSpades)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	queenScore := shootFollowScore(queenOfSpades, g, a)

	if queenScore != 10 {
		t.Errorf("Q♠ shoot follow score = %d, want 10 (deferred: K♠/A♠ in hand)", queenScore)
	}
}

// TestShootFollowScoreQueenWouldWinNoHigherSpade verifies that Q♠ scores 100
// when it would win and no K♠/A♠ is in hand. This is the paired control for
// TestShootFollowScoreQueenDeferredForHigherSpade — same setup minus K♠/A♠,
// proving the defer branch (10) vs would-win branch (100) distinction.
// Branch: shootFollowScore Q♠ would win, no K♠/A♠ → 100.
func TestShootFollowScoreQueenWouldWinNoHigherSpade(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		queenOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rEight, sSpades)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	queenScore := shootFollowScore(queenOfSpades, g, a)

	if queenScore != 100 {
		t.Errorf("Q♠ shoot follow score = %d, want 100 (would win, no K♠/A♠)", queenScore)
	}
}

// TestShootFollowScoreQueenWouldLose verifies that Q♠ scores -50
// when it would lose the trick. Losing Q♠ gives an opponent a penalty
// card, killing the shoot.
func TestShootFollowScoreQueenWouldLose(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		queenOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, aceOfSpades},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	queenScore := shootFollowScore(queenOfSpades, g, a)

	if queenScore != -50 {
		t.Errorf("Q♠ shoot follow score = %d, want -50 (would lose)", queenScore)
	}
}

// TestShootVoidScoreNeverDumpsHearts verifies that hearts score -100 when
// void and shooting. Dumping a heart gives an opponent a penalty card.
func TestShootVoidScoreNeverDumpsHearts(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		queenOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rSix, sDiamonds)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	heartScore := shootVoidScore(c(rAce, sHearts), g, a)

	if heartScore != -100 {
		t.Errorf("A♥ shoot void score = %d, want -100", heartScore)
	}
}

// TestShootVoidScoreNeverDumpsQueenOfSpades verifies that Q♠ scores -100 when
// void and shooting.
func TestShootVoidScoreNeverDumpsQueenOfSpades(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		queenOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rSix, sDiamonds)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	queenScore := shootVoidScore(queenOfSpades, g, a)

	if queenScore != -100 {
		t.Errorf("Q♠ shoot void score = %d, want -100", queenScore)
	}
}

// TestShootVoidScoreDumpsLowCardsFirst verifies that low non-penalty
// cards are preferred for dumping when shooting. 2♠ (Ace - 0 = 12)
// outscores K♠ (Ace - 11 = 1).
func TestShootVoidScoreDumpsLowCardsFirst(t *testing.T) {
	g := setupFollowState(hearts.South, 2, []cardcore.Card{
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
		c(rFour, sSpades),
		kingOfSpades,
	}, []hearts.Trick{
		validFirstTrick(),
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sClubs),
			hearts.West:  c(rSix, sClubs),
			hearts.North: c(rSeven, sClubs),
			hearts.East:  c(rEight, sClubs),
		}},
	},
		hearts.East,
		[]trickCard{
			{hearts.East, c(rSix, sDiamonds)},
		})

	a := analyze(g, hearts.South)
	a.shootActive = true

	twoScore := shootVoidScore(c(rTwo, sSpades), g, a)
	kingScore := shootVoidScore(kingOfSpades, g, a)

	if twoScore <= kingScore {
		t.Errorf("2♠ shoot void score (%d) should exceed K♠ score (%d)", twoScore, kingScore)
	}
}

// TestShootOrDuckDeterministic verifies that two contrasting hands
// produce opposite shooting decisions through ChoosePass. The
// "shooter" hand has 7 hearts including A♥+K♥+Q♥ plus A♣ (triggers
// considerShoot), so ChoosePass keeps all hearts and passes only
// non-hearts. The "ducker" hand has the same 13-card count but only
// 3 low hearts and no side ace (does not trigger considerShoot), so
// ChoosePass eagerly passes hearts via the normal passScore +10
// hearts bonus.
//
// This is an end-to-end test of the shoot detection → pass scoring pipeline.
// The two hands are identical in size but differ in heart quality, proving the
// heuristic distinguishes shooters from duckers.
func TestShootOrDuckDeterministic(t *testing.T) {
	// Shooter hand: 7 hearts (A♥ K♥ Q♥ J♥ 10♥ 9♥ 8♥) + A♣ + 5 low cards.
	// considerShoot triggers (≥7 hearts + side ace).
	shooterHand := []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rAce, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	}

	// Ducker hand: 3 low hearts, no ace — does not trigger considerShoot.
	duckerHand := []cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rTwo, sDiamonds),
		c(rThree, sDiamonds),
		c(rFour, sDiamonds),
		c(rFive, sDiamonds),
		c(rTwo, sHearts),
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rTwo, sSpades),
	}

	h := newSeededHeuristic(42)

	// Shooter: all 3 passed cards should be non-hearts.
	gShoot := hearts.New()
	if err := gShoot.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	setupPassHand(t, gShoot, hearts.South, shooterHand)
	shootPassed := h.ChoosePass(gShoot, hearts.South)
	for _, card := range shootPassed {
		if card.Suit == sHearts {
			t.Errorf("shooter should keep all hearts, but passed %v (all 3: %v)", card, shootPassed)
			break
		}
	}

	// Ducker: should pass at least one heart (normal passScore gives
	// hearts +10 bonus, making them high-priority pass candidates).
	gDuck := hearts.New()
	if err := gDuck.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	setupPassHand(t, gDuck, hearts.South, duckerHand)
	duckPassed := h.ChoosePass(gDuck, hearts.South)
	passedHeart := false
	for _, card := range duckPassed {
		if card.Suit == sHearts {
			passedHeart = true
			break
		}
	}
	if !passedHeart {
		t.Errorf("ducker should pass at least one heart, but passed %v", duckPassed)
	}
}

// TestHeuristicFullGameIntegration runs 10 complete games with Heuristic
// players
// and verifies structural invariants: games terminate, winner has the lowest
// score.
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

// TestHeuristicStatisticalCompetenceIntegration runs 1000 complete games with
// one Heuristic player against three Random opponents and asserts the
// Heuristic wins more than 75% of the time, both in aggregate and per seat. A
// purely random player would win ~25% by chance (1 in 4); the 75% threshold
// represents a meaningful skill margin while leaving headroom for honest
// future tuning. The Heuristic seat rotates across games (game % 4) so the
// per-seat assertion catches position-dependent skill that the aggregate
// would hide. The master seed is time-based and logged on every run so a
// failure can be reproduced by temporarily hardcoding the logged seed.
func TestHeuristicStatisticalCompetenceIntegration(t *testing.T) {
	const (
		numGames          = 1000
		maxRounds         = 20
		minWinRate        = 0.75
		minWinRatePerSeat = 0.75
	)
	masterSeed := uint64(time.Now().UnixNano())
	t.Logf("masterSeed = %d (hardcode this to reproduce a failure)", masterSeed)

	var (
		winsPerSeat  [hearts.NumPlayers]int
		gamesPerSeat [hearts.NumPlayers]int
	)

	for game := range numGames {
		heuristicSeat := hearts.Seat(game % int(hearts.NumPlayers))

		// Each player gets its own deterministic RNG derived from the master
		// seed and (game, seat) so runs are reproducible and players don't
		// share RNG state.
		var players [hearts.NumPlayers]hearts.Player
		for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
			playerSeed := masterSeed + uint64(game)*uint64(hearts.NumPlayers) + uint64(seat)
			if seat == heuristicSeat {
				players[seat] = newSeededHeuristic(playerSeed)
			} else {
				players[seat] = newSeededRandom(playerSeed)
			}
		}

		g := hearts.New()
		for range maxRounds {
			playRoundWithPlayers(t, g, players, uint64(game))
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
		gamesPerSeat[heuristicSeat]++
		if winner == heuristicSeat {
			winsPerSeat[heuristicSeat]++
		}
	}

	totalWins := 0
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		totalWins += winsPerSeat[seat]
	}
	aggregateRate := float64(totalWins) / float64(numGames)

	// Always log the breakdown so passing runs show the distribution too.
	t.Logf("Heuristic wins: %d/%d aggregate (%.1f%%)",
		totalWins, numGames, aggregateRate*100)
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		t.Logf("  seat %d: %d/%d (%.1f%%)",
			seat, winsPerSeat[seat], gamesPerSeat[seat],
			float64(winsPerSeat[seat])*100/float64(gamesPerSeat[seat]))
	}

	if aggregateRate <= minWinRate {
		t.Errorf("aggregate: Heuristic won %d/%d (%.1f%%); want > %.0f%%",
			totalWins, numGames, aggregateRate*100, minWinRate*100)
	}
	// Per-seat threshold catches position-dependent skill that aggregate hides.
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		seatRate := float64(winsPerSeat[seat]) / float64(gamesPerSeat[seat])
		if seatRate <= minWinRatePerSeat {
			t.Errorf("seat %d: Heuristic won %d/%d (%.1f%%); want > %.0f%%",
				seat, winsPerSeat[seat], gamesPerSeat[seat],
				seatRate*100, minWinRatePerSeat*100)
		}
	}
}

// newSeededHeuristic creates a Heuristic player with a deterministic RNG for
// test
// reproducibility.
func newSeededHeuristic(seed uint64) *Heuristic {
	return NewHeuristic(rand.New(rand.NewPCG(seed, seed+1)))
}

// TestHeuristicStatisticalCompetence runs 1000 complete games with one

// setupPassHand replaces a player's hand with the given cards for pass testing.
func setupPassHand(t *testing.T, g *hearts.Game, seat hearts.Seat, cards []cardcore.Card) {
	t.Helper()
	g.Hands[seat] = cardcore.NewHand(cards)
}

// passedContains reports whether target appears in the passed cards array.
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

// validFirstTrick returns a trick led with 2♣ by South for use in test trick
// histories
// that need a realistic first trick.
func validFirstTrick() hearts.Trick {
	return hearts.Trick{
		Leader: hearts.South,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: twoOfClubs,
			hearts.West:  c(rThree, sClubs),
			hearts.North: c(rFour, sClubs),
			hearts.East:  c(rFive, sClubs),
		},
	}
}

// setupFollowState creates a game in PhasePlay where seat must play into an
// in-progress
// trick. The leader and played cards define the partial trick state.
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
			hearts.South: c(rSeven, sClubs),
			hearts.West:  c(rEight, sClubs),
			hearts.North: c(rNine, sClubs),
			hearts.East:  c(rSix, sClubs),
		}},
		// Trick 2: North leads diamonds, East wins (clean).
		{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFour, sDiamonds),
			hearts.West:  c(rFive, sDiamonds),
			hearts.North: c(rThree, sDiamonds),
			hearts.East:  c(rEight, sDiamonds),
		}},
		// Trick 3: East leads diamonds, North wins (clean).
		{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rSeven, sDiamonds),
			hearts.West:  c(rNine, sDiamonds),
			hearts.North: c(rTen, sDiamonds),
			hearts.East:  c(rSix, sDiamonds),
		}},
		// Trick 4: North leads clubs, East wins. West sloughs 2♥ (1 pt → East).
		{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rJack, sClubs),
			hearts.West:  c(rTwo, sHearts),
			hearts.North: c(rTen, sClubs),
			hearts.East:  c(rKing, sClubs),
		}},
	}
}

// moonThreatHistory returns 7 completed tricks where East wins all penalty
// points. Extends earlyMoonThreatHistory with 2 clean tricks.
//
// Cards used: 2♣–J♣, K♣, A♣, 2♦–A♦, 2♠, 3♠, 2♥
// (28 cards).
// Available for hands: Q♣, spades (4♠–A♠), hearts (3♥–A♥).
func moonThreatHistory() []hearts.Trick {
	h := earlyMoonThreatHistory()
	return append(h,
		// Trick 5: East leads diamonds, North wins (clean).
		hearts.Trick{Leader: hearts.East, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rQueen, sDiamonds),
			hearts.West:  c(rJack, sDiamonds),
			hearts.North: c(rAce, sDiamonds),
			hearts.East:  c(rKing, sDiamonds),
		}},
		// Trick 6: North leads diamonds, North wins (clean).
		hearts.Trick{Leader: hearts.North, Count: hearts.NumPlayers, Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rAce, sClubs),
			hearts.West:  c(rThree, sSpades),
			hearts.North: c(rTwo, sDiamonds),
			hearts.East:  c(rTwo, sSpades),
		}},
	)
}
