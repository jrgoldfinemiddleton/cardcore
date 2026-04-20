package hearts

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// Rank aliases for test readability.
const (
	rAce   = cardcore.Ace
	rTwo   = cardcore.Two
	rThree = cardcore.Three
	rFour  = cardcore.Four
	rFive  = cardcore.Five
	rSix   = cardcore.Six
	rSeven = cardcore.Seven
	rEight = cardcore.Eight
	rNine  = cardcore.Nine
	rTen   = cardcore.Ten
	rJack  = cardcore.Jack
	rQueen = cardcore.Queen
	rKing  = cardcore.King
)

// Suit aliases for test readability.
const (
	sClubs    = cardcore.Clubs
	sDiamonds = cardcore.Diamonds
	sHearts   = cardcore.Hearts
	sSpades   = cardcore.Spades
)

// TestNewGame verifies that a new game starts in PhaseDeal at round 0.
func TestNewGame(t *testing.T) {
	g := New()
	if g.Phase != PhaseDeal {
		t.Fatalf("new game phase = %d, want PhaseDeal", g.Phase)
	}
	if g.Round != 0 {
		t.Fatalf("new game round = %d, want 0", g.Round)
	}
}

// TestCloneGame verifies that Clone produces an independent deep copy of game state.
func TestCloneGame(t *testing.T) {
	g := New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	g.Scores[South] = 42
	g.RoundPts[North] = 7
	g.HeartsBroken = true

	clone := g.Clone()

	// Verify clone matches original.
	for i := range NumPlayers {
		if clone.Hands[i].Len() != g.Hands[i].Len() {
			t.Fatalf("clone hand %d length = %d, want %d",
				i, clone.Hands[i].Len(), g.Hands[i].Len())
		}
		for j, card := range g.Hands[i].Cards {
			if !clone.Hands[i].Cards[j].Equal(card) {
				t.Fatalf("clone hand %d card %d = %v, want %v", i, j, clone.Hands[i].Cards[j], card)
			}
		}
	}
	if clone.Phase != g.Phase {
		t.Fatalf("clone phase = %d, want %d", clone.Phase, g.Phase)
	}
	if clone.Scores[South] != 42 {
		t.Fatalf("clone Scores[South] = %d, want 42", clone.Scores[South])
	}
	if clone.RoundPts[North] != 7 {
		t.Fatalf("clone RoundPts[North] = %d, want 7", clone.RoundPts[North])
	}
	if !clone.HeartsBroken {
		t.Fatalf("clone HeartsBroken = false, want true")
	}

	// Mutate clone hands — original must be unchanged.
	originalFirstCard := g.Hands[South].Cards[0]
	clone.Hands[South].Remove(clone.Hands[South].Cards[0])
	if !g.Hands[South].Contains(originalFirstCard) {
		t.Fatalf("original hand lost card %v after clone mutation", originalFirstCard)
	}
	if g.Hands[South].Len() == clone.Hands[South].Len() {
		t.Fatalf("original and clone hand lengths should differ after removal")
	}

	// Mutate clone scores — original must be unchanged.
	clone.Scores[South] = 50
	if g.Scores[South] != 42 {
		t.Fatalf("original Scores[South] = %d after clone mutation, want 42", g.Scores[South])
	}

	// Mutate clone trick — original must be unchanged.
	clone.Trick.Cards[South] = c(rAce, sSpades)
	clone.Trick.Count = 1
	if g.Trick.Cards[South] != (cardcore.Card{}) {
		t.Fatalf("original Trick.Cards[South] = %v after clone mutation, want zero card",
			g.Trick.Cards[South])
	}
	if g.Trick.Count != 0 {
		t.Fatalf("original Trick.Count = %d after clone mutation, want 0", g.Trick.Count)
	}
}

// TestLegalMovesWrongPhase verifies that LegalMoves returns an error outside PhasePlay.
func TestLegalMovesWrongPhase(t *testing.T) {
	g := New()
	_, err := g.LegalMoves(South)
	if err == nil {
		t.Fatalf("LegalMoves in PhaseDeal returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "phase") {
		t.Fatalf("error = %q, want mention of phase", err)
	}
}

// TestLegalMovesWrongTurn verifies that LegalMoves returns an error when called for the wrong seat.
func TestLegalMovesWrongTurn(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.Turn = South
	g.Hands[North] = cardcore.NewHand([]cardcore.Card{c(rAce, sSpades)})

	_, err := g.LegalMoves(North)
	if err == nil {
		t.Fatalf("LegalMoves(North) when South's turn returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "turn") {
		t.Fatalf("error = %q, want mention of turn", err)
	}
}

// TestLegalMovesFirstTrickLeader verifies that only 2♣ is legal when leading the first trick.
func TestLegalMovesFirstTrickLeader(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 0
	g.Turn = South
	g.Trick = Trick{Leader: South}
	g.Hands[South] = cardcore.NewHand([]cardcore.Card{
		twoOfClubs, c(rKing, sHearts), c(rAce, sSpades),
	})

	legal, err := g.LegalMoves(South)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 1 || legal[0] != twoOfClubs {
		t.Fatalf("legal = %v, want [2♣]", legal)
	}
}

// TestLegalMovesFollowSuit verifies that only cards of the led suit are legal when following.
func TestLegalMovesFollowSuit(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 1
	g.Turn = East
	g.Trick = Trick{Leader: South, Cards: [NumPlayers]cardcore.Card{
		South: c(rFive, sDiamonds),
	}, Count: 1}
	g.Hands[East] = cardcore.NewHand([]cardcore.Card{
		c(rThree, sDiamonds), c(rKing, sDiamonds), c(rQueen, sHearts), c(rAce, sSpades),
	})

	legal, err := g.LegalMoves(East)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 2 {
		t.Fatalf("legal = %v, want 2 diamonds", legal)
	}
	for _, card := range legal {
		if card.Suit != sDiamonds {
			t.Fatalf("legal contains %v, want only diamonds", card)
		}
	}
}

// TestLegalMovesVoidInLedSuit verifies that all cards are legal when void in the led suit.
func TestLegalMovesVoidInLedSuit(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 1
	g.Turn = West
	g.Trick = Trick{Leader: South, Cards: [NumPlayers]cardcore.Card{
		South: c(rFive, sDiamonds),
	}, Count: 1}
	g.Hands[West] = cardcore.NewHand([]cardcore.Card{
		c(rKing, sClubs), c(rQueen, sHearts), c(rAce, sSpades),
	})

	legal, err := g.LegalMoves(West)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 3 {
		t.Fatalf("legal = %v, want all 3 cards (void in diamonds)", legal)
	}
}

// TestLegalMovesHeartsNotBroken verifies that hearts cannot be led when hearts are not broken.
func TestLegalMovesHeartsNotBroken(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 1
	g.HeartsBroken = false
	g.Turn = South
	g.Trick = Trick{Leader: South}
	g.Hands[South] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sHearts), c(rKing, sHearts), c(rAce, sSpades),
	})

	legal, err := g.LegalMoves(South)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 1 || legal[0] != c(rAce, sSpades) {
		t.Fatalf("legal = %v, want [A♠] (hearts not broken)", legal)
	}
}

// TestLegalMovesQueenOfSpadesLead verifies that Q♠ can be led when hearts are not broken.
func TestLegalMovesQueenOfSpadesLead(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 1
	g.HeartsBroken = false
	g.Turn = South
	g.Trick = Trick{Leader: South}
	g.Hands[South] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sHearts), c(rKing, sHearts), queenOfSpades,
	})

	legal, err := g.LegalMoves(South)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 1 || legal[0] != queenOfSpades {
		t.Fatalf("legal = %v, want [Q♠] (not a heart, legal to lead)", legal)
	}
}

// TestLegalMovesOnlyHeartsRemain verifies that hearts can be led when the hand has only hearts.
func TestLegalMovesOnlyHeartsRemain(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 1
	g.HeartsBroken = false
	g.Turn = South
	g.Trick = Trick{Leader: South}
	g.Hands[South] = cardcore.NewHand([]cardcore.Card{
		c(rJack, sHearts), c(rQueen, sHearts), c(rKing, sHearts),
	})

	legal, err := g.LegalMoves(South)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 3 {
		t.Fatalf("legal = %v, want all 3 hearts (only hearts remain)", legal)
	}
}

// TestLegalMovesFirstTrickNoPoints verifies that point cards are excluded on
// the first trick when following void.
func TestLegalMovesFirstTrickNoPoints(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 0
	g.Turn = East
	g.Trick = Trick{Leader: South, Cards: [NumPlayers]cardcore.Card{
		South: twoOfClubs,
	}, Count: 1}
	g.Hands[East] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sHearts), queenOfSpades, c(rAce, sSpades),
	})

	legal, err := g.LegalMoves(East)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 1 || legal[0] != c(rAce, sSpades) {
		t.Fatalf("legal = %v, want [A♠] (no points on first trick)", legal)
	}
}

// TestLegalMovesFirstTrickOnlyPointCards verifies that all cards are legal on
// the first trick when only point cards remain.
func TestLegalMovesFirstTrickOnlyPointCards(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 0
	g.Turn = East
	g.Trick = Trick{Leader: South, Cards: [NumPlayers]cardcore.Card{
		South: twoOfClubs,
	}, Count: 1}
	g.Hands[East] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sHearts), c(rKing, sHearts), queenOfSpades,
	})

	legal, err := g.LegalMoves(East)
	if err != nil {
		t.Fatalf("LegalMoves error: %v", err)
	}
	if len(legal) != 3 {
		t.Fatalf("legal = %v, want all 3 (only point cards available)", legal)
	}
}

// TestLegalMovesRoundtrip verifies consistency between LegalMoves and PlayCard
// across 50 random games.
func TestLegalMovesRoundtrip(t *testing.T) {
	for seed := range 50 {
		rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed+1)))
		g := New()
		if err := g.Deal(); err != nil {
			t.Fatalf("seed %d: Deal error: %v", seed, err)
		}

		// Skip pass phase by setting up directly.
		g.Phase = PhasePlay
		g.startFirstTrick()

		for g.Phase == PhasePlay {
			seat := g.Turn
			legal, err := g.LegalMoves(seat)
			if err != nil {
				t.Fatalf("seed %d: LegalMoves error: %v", seed, err)
			}
			if len(legal) == 0 {
				t.Fatalf("seed %d: no legal moves for seat %d with hand %v",
					seed, seat, g.Hands[seat].Cards)
			}

			// Every card NOT in legal must be rejected by PlayCard.
			for _, card := range g.Hands[seat].Cards {
				isLegal := false
				for _, lc := range legal {
					if card.Equal(lc) {
						isLegal = true
						break
					}
				}
				if !isLegal {
					clone := g.Clone()
					if err := clone.PlayCard(seat, card); err == nil {
						t.Fatalf("seed %d: PlayCard accepted %v but LegalMoves excluded it",
							seed, card)
					}
				}
			}

			// Play a random legal card.
			pick := legal[rng.IntN(len(legal))]
			if err := g.PlayCard(seat, pick); err != nil {
				t.Fatalf("seed %d: PlayCard rejected legal move %v: %v", seed, pick, err)
			}
		}
	}
}

// TestDeal verifies that Deal distributes 13 cards to each player and advances to PhasePass.
func TestDeal(t *testing.T) {
	g := New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	if g.Phase != PhasePass {
		t.Fatalf("phase after deal = %d, want PhasePass", g.Phase)
	}

	totalCards := 0
	for i := range NumPlayers {
		if g.Hands[i] == nil {
			t.Fatalf("player %d hand is nil", i)
		}
		totalCards += g.Hands[i].Len()
		if g.Hands[i].Len() != HandSize {
			t.Errorf("player %d has %d cards, want %d", i, g.Hands[i].Len(), HandSize)
		}
	}
	if totalCards != 52 {
		t.Fatalf("total cards = %d, want 52", totalCards)
	}
}

// TestDealHoldRound verifies that Deal skips PhasePass on hold rounds and goes to PhasePlay.
func TestDealHoldRound(t *testing.T) {
	g := New()
	g.PassDir = PassHold
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	if g.Phase != PhasePlay {
		t.Fatalf("phase after deal with hold = %d, want PhasePlay", g.Phase)
	}
}

// TestDealWrongPhase verifies that Deal returns an error outside PhaseDeal.
func TestDealWrongPhase(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	if err := g.Deal(); err == nil {
		t.Error("Deal in PhasePlay returned nil error, want non-nil")
	}
}

// TestPassWrongPhase verifies that SetPass returns an error outside PhasePass.
func TestPassWrongPhase(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	cards := [PassCount]cardcore.Card{}
	if err := g.SetPass(South, cards); err == nil {
		t.Error("SetPass in PhasePlay returned nil error, want non-nil")
	}
}

// TestPlayCardWrongPhase verifies that PlayCard returns an error outside PhasePlay.
func TestPlayCardWrongPhase(t *testing.T) {
	g := New()
	g.Phase = PhasePass
	if err := g.PlayCard(South, twoOfClubs); err == nil {
		t.Error("PlayCard in PhasePass returned nil error, want non-nil")
	}
}

// TestEndRoundWrongPhase verifies that EndRound returns an error outside PhaseScore.
func TestEndRoundWrongPhase(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	if err := g.EndRound(); err == nil {
		t.Error("EndRound in PhasePlay returned nil error, want non-nil")
	}
}

// TestWinnerWrongPhase verifies that Winner returns an error before PhaseEnd.
func TestWinnerWrongPhase(t *testing.T) {
	g := New()
	g.Phase = PhasePlay
	if _, err := g.Winner(); err == nil {
		t.Error("Winner before game over returned nil error, want non-nil")
	}
}

// TestPassValidation verifies that SetPass rejects cards not in hand and duplicate cards.
func TestPassValidation(t *testing.T) {
	g := newPassGame(t)

	hand := g.Hands[South]
	var cards [PassCount]cardcore.Card
	copy(cards[:], hand.Cards[:PassCount])

	// Find three cards not in South's hand.
	var missing [PassCount]cardcore.Card
	found := 0
	for _, suit := range cardcore.AllSuits() {
		for _, rank := range cardcore.AllRanks() {
			candidate := c(rank, suit)
			if !hand.Contains(candidate) {
				missing[found] = candidate
				found++
				if found == PassCount {
					break
				}
			}
		}
		if found == PassCount {
			break
		}
	}
	if err := g.SetPass(South, missing); err == nil {
		t.Error("SetPass with cards not in hand returned nil error, want non-nil")
	}

	dupes := [PassCount]cardcore.Card{hand.Cards[0], hand.Cards[0], hand.Cards[1]}
	if err := g.SetPass(South, dupes); err == nil {
		t.Error("SetPass with duplicate cards returned nil error, want non-nil")
	}

	if err := g.SetPass(South, cards); err != nil {
		t.Fatalf("SetPass error: %v", err)
	}
}

// TestPassExchange verifies that passed cards are removed from the sender and
// added to the receiver.
func TestPassExchange(t *testing.T) {
	g := newPassGame(t)

	passedCards := [NumPlayers][PassCount]cardcore.Card{}
	for i := Seat(0); i < NumPlayers; i++ {
		copy(passedCards[i][:], g.Hands[i].Cards[:PassCount])
	}

	for i := Seat(0); i < NumPlayers; i++ {
		if err := g.SetPass(i, passedCards[i]); err != nil {
			t.Fatalf("SetPass(%d) error: %v", i, err)
		}
	}

	if g.Phase != PhasePlay {
		t.Fatalf("phase after all passes = %d, want PhasePlay", g.Phase)
	}

	for i := Seat(0); i < NumPlayers; i++ {
		if g.Hands[i].Len() != HandSize {
			t.Errorf("player %d has %d cards after pass, want %d", i, g.Hands[i].Len(), HandSize)
		}
	}

	// PassLeft: South's cards should go to West.
	for _, card := range passedCards[South] {
		if !g.Hands[West].Contains(card) {
			t.Errorf("West should have received %v from South", card)
		}
		if g.Hands[South].Contains(card) {
			t.Errorf("South should no longer have %v after passing", card)
		}
	}
}

// TestFirstTrickMustLead2C verifies that the 2♣ holder must lead it on the first trick.
func TestFirstTrickMustLead2C(t *testing.T) {
	g := newHoldGame(t)

	holder := findHolder(g, twoOfClubs)
	if g.Turn != holder {
		t.Fatalf("turn = %d, want %d (2♣ holder)", g.Turn, holder)
	}

	wrongCard := findAnyOtherCard(g, holder, twoOfClubs)
	err := g.PlayCard(holder, wrongCard)
	if err == nil {
		t.Fatal("PlayCard non-2♣ on first trick returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "first trick") {
		t.Errorf("err = %v, want to contain %q", err, "first trick")
	}

	if err := g.PlayCard(holder, twoOfClubs); err != nil {
		t.Fatalf("PlayCard 2♣ error: %v", err)
	}
}

// TestMustFollowSuit verifies that a player with cards in the led suit must play one.
func TestMustFollowSuit(t *testing.T) {
	g := setupFixedHands()

	if err := g.PlayCard(South, twoOfClubs); err != nil {
		t.Fatalf("PlayCard 2♣ error: %v", err)
	}

	// West has clubs and must follow suit.
	if err := g.PlayCard(West, c(rTwo, sDiamonds)); err == nil {
		t.Error("PlayCard off-suit when able to follow returned nil error, want non-nil")
	}

	if err := g.PlayCard(West, c(rThree, sClubs)); err != nil {
		t.Fatalf("PlayCard 3♣ error: %v", err)
	}
}

// TestCannotPlayPointsOnFirstTrick verifies that hearts and Q♠ cannot be played
// on the first trick when void.
func TestCannotPlayPointsOnFirstTrick(t *testing.T) {
	g := setupVoidClubs()

	if err := g.PlayCard(South, twoOfClubs); err != nil {
		t.Fatalf("PlayCard 2♣ error: %v", err)
	}

	// West is void in clubs and has non-penalty cards.
	// Playing a heart should be rejected.
	if err := g.PlayCard(West, c(rTwo, sHearts)); err == nil {
		t.Error("PlayCard heart on first trick returned nil error, want non-nil")
	}

	// Playing Q♠ should be rejected.
	if err := g.PlayCard(West, queenOfSpades); err == nil {
		t.Error("PlayCard Q♠ on first trick returned nil error, want non-nil")
	}

	// Playing a non-penalty card should succeed.
	if err := g.PlayCard(West, c(rTwo, sDiamonds)); err != nil {
		t.Fatalf("PlayCard 2♦ error: %v", err)
	}
}

// TestTrickResolution verifies that a completed trick advances TrickNum.
func TestTrickResolution(t *testing.T) {
	g := setupFixedHands()

	holder := findHolder(g, twoOfClubs)
	if err := g.PlayCard(holder, twoOfClubs); err != nil {
		t.Fatalf("PlayCard 2♣ error: %v", err)
	}

	for g.TrickNum == 0 {
		playAnyValid(g, g.Turn)
	}

	if g.TrickNum != 1 {
		t.Errorf("trick number = %d, want 1", g.TrickNum)
	}
}

// TestHeartsBroken verifies that hearts are not broken initially and become
// broken when a heart is played.
func TestHeartsBroken(t *testing.T) {
	g := setupFixedHands()

	if g.HeartsBroken {
		t.Fatal("hearts should not be broken initially")
	}

	// Play through the first trick.
	for g.TrickNum == 0 {
		playAnyValid(g, g.Turn)
	}

	if g.HeartsBroken {
		t.Fatal("hearts should not be broken after first trick (no hearts playable)")
	}

	// Play cards until a heart is played.
	var breakingCard cardcore.Card
	for !g.HeartsBroken {
		seat := g.Turn
		for _, card := range g.Hands[seat].Cards {
			if err := g.PlayCard(seat, card); err == nil {
				if card.Suit == sHearts {
					breakingCard = card
				}
				break
			}
		}
	}

	if breakingCard.Suit != sHearts {
		t.Errorf("breakingCard.Suit = %v, want %v", breakingCard.Suit, sHearts)
	}
}

// TestScoring verifies that round points are added to cumulative scores.
func TestScoring(t *testing.T) {
	g := New()

	g.Scores = [NumPlayers]int{10, 5, 4, 7}
	g.RoundPts = [NumPlayers]int{5, 8, 0, 13}
	g.Phase = PhasePlay
	g.TrickNum = HandSize
	g.scoreRound()

	want := [NumPlayers]int{15, 13, 4, 20}
	if g.Scores != want {
		t.Errorf("scores = %v, want %v", g.Scores, want)
	}
}

// TestShootTheMoon verifies that shooting the moon adds 26 to all opponents instead of the shooter.
func TestShootTheMoon(t *testing.T) {
	g := New()

	g.Scores = [NumPlayers]int{10, 5, 4, 7}
	g.RoundPts = [NumPlayers]int{0, 0, MoonPoints, 0}
	g.Phase = PhasePlay
	g.TrickNum = HandSize
	g.scoreRound()

	want := [NumPlayers]int{36, 31, 4, 33}
	if g.Scores != want {
		t.Errorf("scores = %v, want %v", g.Scores, want)
	}
}

// TestGameEnd verifies that the game transitions to PhaseEnd when a player reaches MaxScore.
func TestGameEnd(t *testing.T) {
	g := New()
	g.Scores = [NumPlayers]int{94, 52, 36, 26}

	g.RoundPts = [NumPlayers]int{10, 3, 8, 5}
	g.Phase = PhasePlay
	g.TrickNum = HandSize
	g.scoreRound()

	if g.Phase != PhaseScore {
		t.Fatalf("phase = %d, want PhaseScore", g.Phase)
	}

	if err := g.EndRound(); err != nil {
		t.Fatalf("EndRound error: %v", err)
	}

	if g.Phase != PhaseEnd {
		t.Fatalf("phase = %d, want PhaseEnd", g.Phase)
	}

	winner, err := g.Winner()
	if err != nil {
		t.Fatalf("Winner error: %v", err)
	}
	if winner != East {
		t.Errorf("winner = %d, want East (%d)", winner, East)
	}
}

// TestPassDirectionRotation verifies that pass direction cycles Left→Right→Across→Hold.
func TestPassDirectionRotation(t *testing.T) {
	g := New()

	want := []PassDirection{PassLeft, PassRight, PassAcross, PassHold, PassLeft}
	for i, dir := range want {
		if g.PassDir != dir {
			t.Errorf("round %d: passDir = %d, want %d", i, g.PassDir, dir)
		}

		g.RoundPts = [NumPlayers]int{3, 5, 8, 10}
		g.Phase = PhasePlay
		g.TrickNum = HandSize
		g.scoreRound()
		if err := g.EndRound(); err != nil {
			t.Fatalf("EndRound error on round %d: %v", i, err)
		}
	}
}

// TestWrongTurn verifies that PlayCard returns an error when played out of turn.
func TestWrongTurn(t *testing.T) {
	g := newHoldGame(t)
	wrongSeat := nextSeat(g.Turn)

	card := g.Hands[wrongSeat].Cards[0]
	err := g.PlayCard(wrongSeat, card)
	if err == nil {
		t.Fatal("PlayCard out of turn returned nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "turn") {
		t.Errorf("err = %v, want to contain %q", err, "turn")
	}
}

// TestTrickPoints verifies that trickPoints sums hearts (1 each) and Q♠ (13).
func TestTrickPoints(t *testing.T) {
	g := New()
	g.Trick = Trick{
		Cards: [NumPlayers]cardcore.Card{
			c(rTwo, sHearts),
			c(rFive, sHearts),
			queenOfSpades,
			c(rThree, sClubs),
		},
	}
	pts := g.trickPoints()
	if pts != 15 {
		t.Errorf("trick points = %d, want 15 (2 hearts + Q♠)", pts)
	}
}

// TestPassHistoryAccuracy verifies that PassHistory records the exact cards each player passed.
func TestPassHistoryAccuracy(t *testing.T) {
	g := newPassGame(t)

	passedCards := [NumPlayers][PassCount]cardcore.Card{}
	for i := Seat(0); i < NumPlayers; i++ {
		copy(passedCards[i][:], g.Hands[i].Cards[:PassCount])
	}

	for i := Seat(0); i < NumPlayers; i++ {
		if err := g.SetPass(i, passedCards[i]); err != nil {
			t.Fatalf("SetPass(%d) error: %v", i, err)
		}
	}

	if g.PassHistory != passedCards {
		t.Fatalf("PassHistory = %v, want %v", g.PassHistory, passedCards)
	}
}

// TestPassHistoryResetOnDeal verifies that PassHistory is zeroed on a new deal.
func TestPassHistoryResetOnDeal(t *testing.T) {
	g := New()
	playRandomRound(t, g)

	// PassHistory should have data from the round just played.
	// After a new Deal, it should be zeroed.
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	zero := [NumPlayers][PassCount]cardcore.Card{}
	if g.PassHistory != zero {
		t.Fatalf("PassHistory after Deal is not zeroed: %v", g.PassHistory)
	}
}

// TestPassHistoryHoldRound verifies that PassHistory is zeroed on hold rounds.
func TestPassHistoryHoldRound(t *testing.T) {
	g := New()
	g.PassDir = PassHold
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}

	zero := [NumPlayers][PassCount]cardcore.Card{}
	if g.PassHistory != zero {
		t.Fatalf("PassHistory on hold round is not zeroed: %v", g.PassHistory)
	}
}

// TestTrickHistoryMidTrick verifies that TrickHistory only grows when a trick is completed.
func TestTrickHistoryMidTrick(t *testing.T) {
	g := setupFixedHands()

	// Play first trick to completion.
	for g.TrickNum == 0 {
		playAnyValid(g, g.Turn)
	}
	if len(g.TrickHistory) != 1 {
		t.Fatalf("len(TrickHistory) after trick 1 = %d, want 1", len(g.TrickHistory))
	}

	// Play 2 of 4 cards in the second trick.
	playAnyValid(g, g.Turn)
	playAnyValid(g, g.Turn)

	if len(g.TrickHistory) != 1 {
		t.Fatalf("len(TrickHistory) mid-trick = %d, want 1 (trick not yet complete)",
			len(g.TrickHistory))
	}

	// Complete the second trick.
	playAnyValid(g, g.Turn)
	playAnyValid(g, g.Turn)

	if len(g.TrickHistory) != 2 {
		t.Fatalf("len(TrickHistory) after trick 2 = %d, want 2", len(g.TrickHistory))
	}
}

// TestTrickHistoryCloneIndependence verifies that cloned TrickHistory is
// independent from the original.
func TestTrickHistoryCloneIndependence(t *testing.T) {
	g := setupFixedHands()

	// Play 5 tricks.
	for g.TrickNum < 5 {
		playAnyValid(g, g.Turn)
	}
	if len(g.TrickHistory) != 5 {
		t.Fatalf("len(TrickHistory) = %d, want 5", len(g.TrickHistory))
	}

	clone := g.Clone()

	if len(clone.TrickHistory) != 5 {
		t.Fatalf("clone len(TrickHistory) = %d, want 5", len(clone.TrickHistory))
	}

	// Mutate clone's TrickHistory.
	clone.TrickHistory[0].Leader = East
	clone.TrickHistory[0].Count = 99

	// Original must be unchanged.
	if g.TrickHistory[0].Leader == East {
		t.Error("original TrickHistory[0].Leader changed after clone mutation")
	}
	if g.TrickHistory[0].Count == 99 {
		t.Error("original TrickHistory[0].Count changed after clone mutation")
	}
}

// TestTrickHistoryResetOnDeal verifies that TrickHistory is cleared on a new deal.
func TestTrickHistoryResetOnDeal(t *testing.T) {
	g := New()
	playRandomRound(t, g)

	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	if len(g.TrickHistory) != 0 {
		t.Fatalf("len(TrickHistory) after Deal = %d, want 0", len(g.TrickHistory))
	}
}

// TestTrickHistoryAccumulation verifies that all 13 tricks are recorded with correct point totals.
func TestTrickHistoryAccumulation(t *testing.T) {
	g := setupFixedHands()

	for g.Phase == PhasePlay {
		playAnyValid(g, g.Turn)
	}

	if len(g.TrickHistory) != HandSize {
		t.Fatalf("len(TrickHistory) = %d, want %d", len(g.TrickHistory), HandSize)
	}

	for i, tr := range g.TrickHistory {
		if tr.Count != NumPlayers {
			t.Errorf("trick %d: Count = %d, want %d", i, tr.Count, NumPlayers)
		}
		ledSuit := tr.Cards[tr.Leader].Suit
		if tr.LedSuit() != ledSuit {
			t.Errorf("trick %d: LedSuit() = %v, want %v", i, tr.LedSuit(), ledSuit)
		}
	}
	if pts := trickHistoryPoints(g.TrickHistory); pts != MoonPoints {
		t.Errorf("total points across TrickHistory = %d, want %d", pts, MoonPoints)
	}
}

// TestFullGameIntegration verifies that 5 random games run to completion with valid winners.
func TestFullGameIntegration(t *testing.T) {
	const (
		numGames = 5
		// Safety cap to prevent infinite loops if the engine has a bug.
		// Worst case is ~17 rounds (26 pts/round across 4 players, game
		// ends when any single player reaches 100).
		maxRounds = 20
	)

	for game := range numGames {
		g := New()

		for round := range maxRounds {
			wantDir := PassDirection(round % NumPassDirections)
			if g.PassDir != wantDir {
				t.Fatalf("game %d, round %d: PassDir = %d, want %d",
					game, round, g.PassDir, wantDir)
			}

			playRandomRound(t, g)

			if g.Phase == PhaseEnd {
				break
			}
		}

		if g.Phase != PhaseEnd {
			t.Fatalf("game %d: did not end within %d rounds", game, maxRounds)
		}

		// Verify someone reached MaxScore.
		maxScore := 0
		for i := range NumPlayers {
			if g.Scores[i] > maxScore {
				maxScore = g.Scores[i]
			}
		}
		if maxScore < MaxScore {
			t.Fatalf("game %d: no player reached %d, max score = %d", game, MaxScore, maxScore)
		}

		verifyWinner(t, g, game)
	}
}

// TestShootTheMoonIntegration verifies end-to-end shoot-the-moon scoring with scripted hands.
func TestShootTheMoonIntegration(t *testing.T) {
	const (
		numGames  = 5
		maxRounds = 20
		moonable  = MaxScore - MoonPoints
	)

	type play struct {
		seat Seat
		card cardcore.Card
	}

	type trick [NumPlayers]play

	moonHands := [NumPlayers][]cardcore.Card{
		{ // South — the shooter
			twoOfClubs, c(rJack, sClubs), c(rQueen, sClubs), c(rKing, sClubs),
			c(rQueen, sDiamonds), c(rKing, sDiamonds), c(rAce, sDiamonds),
			c(rQueen, sHearts), c(rKing, sHearts), c(rAce, sHearts),
			c(rJack, sSpades), c(rKing, sSpades), c(rAce, sSpades),
		},
		{ // West
			c(rNine, sClubs), c(rTen, sClubs), c(rAce, sClubs),
			c(rTwo, sDiamonds), c(rFive, sDiamonds), c(rEight, sDiamonds), c(rJack, sDiamonds),
			c(rFour, sHearts), c(rSeven, sHearts), c(rTen, sHearts),
			c(rFour, sSpades), c(rSeven, sSpades), c(rTen, sSpades),
		},
		{ // North
			c(rThree, sClubs), c(rFive, sClubs), c(rEight, sClubs),
			c(rFour, sDiamonds), c(rSeven, sDiamonds), c(rTen, sDiamonds),
			c(rTwo, sHearts), c(rFive, sHearts), c(rEight, sHearts), c(rJack, sHearts),
			c(rThree, sSpades), c(rSix, sSpades), c(rNine, sSpades),
		},
		{ // East
			c(rFour, sClubs), c(rSix, sClubs), c(rSeven, sClubs),
			c(rThree, sDiamonds), c(rSix, sDiamonds), c(rNine, sDiamonds),
			c(rThree, sHearts), c(rSix, sHearts), c(rNine, sHearts),
			c(rTwo, sSpades), c(rFive, sSpades), c(rEight, sSpades), queenOfSpades,
		},
	}

	script := [HandSize]trick{
		// Trick 1: South leads 2♣, West wins with A♣. No penalty cards.
		{
			{South, twoOfClubs},
			{West, c(rAce, sClubs)},
			{North, c(rThree, sClubs)},
			{East, c(rFour, sClubs)},
		},
		// Trick 2: West leads 2♦, South wins with A♦.
		{
			{West, c(rTwo, sDiamonds)},
			{North, c(rFour, sDiamonds)},
			{East, c(rThree, sDiamonds)},
			{South, c(rAce, sDiamonds)},
		},
		// Trick 3: South leads K♣, all follow clubs.
		{
			{South, c(rKing, sClubs)},
			{West, c(rTen, sClubs)},
			{North, c(rFive, sClubs)},
			{East, c(rSix, sClubs)},
		},
		// Trick 4: South leads Q♣, all follow clubs.
		{
			{South, c(rQueen, sClubs)},
			{West, c(rNine, sClubs)},
			{North, c(rEight, sClubs)},
			{East, c(rSeven, sClubs)},
		},
		// Trick 5: South leads J♣, all void in clubs. Hearts broken.
		// North sloughs 2♥, East sloughs Q♠. South takes 14 pts.
		{
			{South, c(rJack, sClubs)},
			{West, c(rJack, sDiamonds)},
			{North, c(rTwo, sHearts)},
			{East, queenOfSpades},
		},
		// Trick 6: South leads K♦, all follow diamonds.
		{
			{South, c(rKing, sDiamonds)},
			{West, c(rFive, sDiamonds)},
			{North, c(rSeven, sDiamonds)},
			{East, c(rSix, sDiamonds)},
		},
		// Trick 7: South leads Q♦, all follow diamonds.
		{
			{South, c(rQueen, sDiamonds)},
			{West, c(rEight, sDiamonds)},
			{North, c(rTen, sDiamonds)},
			{East, c(rNine, sDiamonds)},
		},
		// Trick 8: South leads A♠, all follow spades.
		{
			{South, c(rAce, sSpades)},
			{West, c(rFour, sSpades)},
			{North, c(rThree, sSpades)},
			{East, c(rTwo, sSpades)},
		},
		// Trick 9: South leads K♠, all follow spades.
		{
			{South, c(rKing, sSpades)},
			{West, c(rSeven, sSpades)},
			{North, c(rSix, sSpades)},
			{East, c(rFive, sSpades)},
		},
		// Trick 10: South leads J♠, all follow spades.
		{
			{South, c(rJack, sSpades)},
			{West, c(rTen, sSpades)},
			{North, c(rNine, sSpades)},
			{East, c(rEight, sSpades)},
		},
		// Trick 11: South leads A♥. 4 pts.
		{
			{South, c(rAce, sHearts)},
			{West, c(rFour, sHearts)},
			{North, c(rFive, sHearts)},
			{East, c(rThree, sHearts)},
		},
		// Trick 12: South leads K♥. 4 pts.
		{
			{South, c(rKing, sHearts)},
			{West, c(rSeven, sHearts)},
			{North, c(rEight, sHearts)},
			{East, c(rSix, sHearts)},
		},
		// Trick 13: South leads Q♥. 4 pts.
		{
			{South, c(rQueen, sHearts)},
			{West, c(rTen, sHearts)},
			{North, c(rJack, sHearts)},
			{East, c(rNine, sHearts)},
		},
	}

	for game := range numGames {
		g := New()

		// Play random rounds until a non-South player is moonable.
		foundMoonTarget := false
		for range maxRounds {
			playRandomRound(t, g)

			if g.Phase == PhaseEnd {
				break
			}
			for i := Seat(1); i < NumPlayers; i++ {
				if g.Scores[i] >= moonable {
					foundMoonTarget = true
					break
				}
			}
			if foundMoonTarget {
				break
			}
		}

		if !foundMoonTarget {
			// South can hit MaxScore and end the game before any
			// opponent reaches the moonable threshold.
			continue
		}

		// Inject scripted moon round.
		scoresBeforeMoon := g.Scores

		g.PassDir = PassHold
		if err := g.Deal(); err != nil {
			t.Fatalf("game %d: Deal error: %v", game, err)
		}

		for i := range NumPlayers {
			g.Hands[i] = cardcore.NewHand(moonHands[i])
		}
		g.Turn = South
		g.Trick = Trick{Leader: South}

		for i, tr := range script {
			for _, p := range tr {
				if err := g.PlayCard(p.seat, p.card); err != nil {
					t.Fatalf("game %d, trick %d: PlayCard(%d, %v): %v",
						game, i+1, p.seat, p.card, err)
				}
			}
		}

		if g.Phase != PhaseScore {
			t.Fatalf("game %d: phase = %d after scripted round, want PhaseScore", game, g.Phase)
		}

		if g.RoundPts[South] != MoonPoints {
			t.Fatalf("game %d: South RoundPts = %d, want %d", game, g.RoundPts[South], MoonPoints)
		}

		roundTotal := 0
		for i := range NumPlayers {
			roundTotal += g.RoundPts[i]
		}
		if roundTotal != MoonPoints {
			t.Fatalf("game %d: sum(RoundPts) = %d, want %d", game, roundTotal, MoonPoints)
		}

		if err := g.EndRound(); err != nil {
			t.Fatalf("game %d: EndRound error: %v", game, err)
		}

		// Verify moon scoring: shooter gets +0, others get +MoonPoints.
		if g.Scores[South] != scoresBeforeMoon[South] {
			t.Errorf("game %d: South score changed from %d to %d, want no change",
				game, scoresBeforeMoon[South], g.Scores[South])
		}
		for i := Seat(1); i < NumPlayers; i++ {
			want := scoresBeforeMoon[i] + MoonPoints
			if g.Scores[i] != want {
				t.Errorf("game %d: player %d score = %d, want %d", game, i, g.Scores[i], want)
			}
		}

		if g.Phase != PhaseEnd {
			t.Fatalf("game %d: phase = %d after moon round, want PhaseEnd", game, g.Phase)
		}

		verifyWinner(t, g, game)
	}
}

// c is a shorthand constructor for cardcore.Card.
func c(rank cardcore.Rank, suit cardcore.Suit) cardcore.Card {
	return cardcore.Card{Rank: rank, Suit: suit}
}

// verifyWinner checks that the declared winner has the lowest score.
func verifyWinner(t *testing.T, g *Game, game int) {
	t.Helper()

	winner, err := g.Winner()
	if err != nil {
		t.Fatalf("game %d: Winner error: %v", game, err)
	}
	for i := Seat(0); i < NumPlayers; i++ {
		if g.Scores[i] < g.Scores[winner] {
			t.Errorf("game %d: player %d has score %d, lower than winner %d with %d",
				game, i, g.Scores[i], winner, g.Scores[winner])
		}
	}
}

// newPassGame creates a dealt game in PhasePass for pass-related tests.
func newPassGame(t *testing.T) *Game {
	t.Helper()
	g := New()
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	return g
}

// playRandomRound plays one full round with random legal moves and validates invariants.
func playRandomRound(t *testing.T, g *Game) {
	t.Helper()

	if err := g.Deal(); err != nil {
		t.Fatalf("round %d: Deal error: %v", g.Round, err)
	}

	for i := Seat(0); i < NumPlayers; i++ {
		if g.Hands[i].Len() != HandSize {
			t.Fatalf("round %d: player %d hand size = %d, want %d",
				g.Round, i, g.Hands[i].Len(), HandSize)
		}
	}

	if g.PassDir != PassHold {
		if g.Phase != PhasePass {
			t.Fatalf("round %d: phase = %d after deal, want PhasePass", g.Round, g.Phase)
		}
		for i := Seat(0); i < NumPlayers; i++ {
			var cards [PassCount]cardcore.Card
			copy(cards[:], g.Hands[i].Cards[:PassCount])
			if err := g.SetPass(i, cards); err != nil {
				t.Fatalf("round %d: SetPass(%d) error: %v", g.Round, i, err)
			}
		}
	}

	if g.Phase != PhasePlay {
		t.Fatalf("round %d: phase = %d, want PhasePlay", g.Round, g.Phase)
	}

	for trick := range HandSize {
		for range NumPlayers {
			playAnyValid(g, g.Turn)
		}
		if g.TrickNum != trick+1 && g.Phase != PhaseScore {
			t.Fatalf("round %d, trick %d: TrickNum = %d, Phase = %d",
				g.Round, trick, g.TrickNum, g.Phase)
		}
	}

	if g.Phase != PhaseScore {
		t.Fatalf("round %d: phase = %d after all tricks, want PhaseScore", g.Round, g.Phase)
	}

	roundTotal := 0
	for i := range NumPlayers {
		roundTotal += g.RoundPts[i]
	}
	if roundTotal != MoonPoints {
		t.Fatalf("round %d: sum(RoundPts) = %d, want %d", g.Round, roundTotal, MoonPoints)
	}

	if len(g.TrickHistory) != HandSize {
		t.Fatalf("round %d: len(TrickHistory) = %d, want %d",
			g.Round, len(g.TrickHistory), HandSize)
	}
	if pts := trickHistoryPoints(g.TrickHistory); pts != MoonPoints {
		t.Fatalf("round %d: sum(TrickHistory points) = %d, want %d", g.Round, pts, MoonPoints)
	}

	for i := Seat(0); i < NumPlayers; i++ {
		if g.Hands[i].Len() != 0 {
			t.Fatalf("round %d: player %d hand not empty (%d cards)", g.Round, i, g.Hands[i].Len())
		}
	}

	if err := g.EndRound(); err != nil {
		t.Fatalf("round %d: EndRound error: %v", g.Round, err)
	}
}

// newHoldGame creates a dealt hold-round game in PhasePlay.
func newHoldGame(t *testing.T) *Game {
	t.Helper()
	g := New()
	g.PassDir = PassHold
	if err := g.Deal(); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	return g
}

// findHolder returns the seat holding the given card.
func findHolder(g *Game, card cardcore.Card) Seat {
	for i := Seat(0); i < NumPlayers; i++ {
		if g.Hands[i].Contains(card) {
			return i
		}
	}
	panic("card not found in any hand")
}

// findAnyOtherCard returns any card in the seat's hand other than the excluded card.
func findAnyOtherCard(g *Game, seat Seat, exclude cardcore.Card) cardcore.Card {
	for _, card := range g.Hands[seat].Cards {
		if !card.Equal(exclude) {
			return card
		}
	}
	panic("no other card found")
}

// trickHistoryPoints sums penalty points across all completed tricks.
func trickHistoryPoints(history []Trick) int {
	pts := 0
	for _, tr := range history {
		for _, card := range tr.Cards {
			if card.Suit == sHearts {
				pts++
			}
			if card == queenOfSpades {
				pts += 13
			}
		}
	}
	return pts
}

// playAnyValid plays a random legal card for the given seat.
func playAnyValid(g *Game, seat Seat) {
	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic(fmt.Sprintf("LegalMoves error: %v", err))
	}
	if len(legal) == 0 {
		panic("no legal moves")
	}
	pick := legal[rand.IntN(len(legal))]
	if err := g.PlayCard(seat, pick); err != nil {
		panic(fmt.Sprintf("PlayCard rejected legal move %v: %v", pick, err))
	}
}

// setupFixedHands creates a game with deterministic hands for rule testing.
func setupFixedHands() *Game {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 0
	g.HeartsBroken = false

	g.Hands[South] = cardcore.NewHand([]cardcore.Card{
		twoOfClubs, c(rFive, sClubs), c(rJack, sClubs),
		c(rThree, sDiamonds), c(rSeven, sDiamonds), c(rQueen, sDiamonds),
		c(rFour, sHearts), c(rEight, sHearts), c(rKing, sHearts),
		c(rThree, sSpades), c(rSix, sSpades), c(rTen, sSpades),
		c(rAce, sSpades),
	})
	g.Hands[West] = cardcore.NewHand([]cardcore.Card{
		c(rThree, sClubs), c(rSeven, sClubs), c(rQueen, sClubs),
		c(rTwo, sDiamonds), c(rSix, sDiamonds), c(rJack, sDiamonds),
		c(rTwo, sHearts), c(rSix, sHearts), c(rTen, sHearts),
		c(rFour, sSpades), c(rEight, sSpades), queenOfSpades,
		c(rKing, sSpades),
	})
	g.Hands[North] = cardcore.NewHand([]cardcore.Card{
		c(rFour, sClubs), c(rEight, sClubs), c(rKing, sClubs),
		c(rFour, sDiamonds), c(rEight, sDiamonds), c(rKing, sDiamonds), c(rAce, sDiamonds),
		c(rThree, sHearts), c(rSeven, sHearts), c(rJack, sHearts),
		c(rTwo, sSpades), c(rSeven, sSpades), c(rJack, sSpades),
	})
	g.Hands[East] = cardcore.NewHand([]cardcore.Card{
		c(rSix, sClubs), c(rNine, sClubs), c(rTen, sClubs),
		c(rAce, sClubs), c(rFive, sDiamonds), c(rNine, sDiamonds),
		c(rTen, sDiamonds), c(rFive, sHearts), c(rNine, sHearts),
		c(rQueen, sHearts), c(rAce, sHearts), c(rFive, sSpades),
		c(rNine, sSpades),
	})

	g.Turn = South
	g.Trick = Trick{Leader: South}

	return g
}

// setupVoidClubs creates a game where West is void in clubs but has
// hearts, Q♠, and non-penalty cards.
func setupVoidClubs() *Game {
	g := New()
	g.Phase = PhasePlay
	g.TrickNum = 0
	g.HeartsBroken = false

	g.Hands[South] = cardcore.NewHand([]cardcore.Card{
		twoOfClubs, c(rThree, sClubs), c(rFour, sClubs),
		c(rFive, sClubs), c(rSix, sClubs), c(rSeven, sClubs),
		c(rEight, sClubs), c(rNine, sClubs), c(rTen, sClubs),
		c(rJack, sClubs), c(rQueen, sClubs), c(rKing, sClubs),
		c(rAce, sClubs),
	})
	g.Hands[West] = cardcore.NewHand([]cardcore.Card{
		c(rTwo, sDiamonds), c(rThree, sDiamonds), c(rFour, sDiamonds),
		c(rTwo, sHearts), c(rThree, sHearts), c(rFour, sHearts),
		c(rFive, sHearts), c(rSix, sHearts), c(rSeven, sHearts),
		c(rEight, sHearts), c(rNine, sHearts),
		c(rTwo, sSpades), queenOfSpades,
	})
	g.Hands[North] = cardcore.NewHand([]cardcore.Card{
		c(rFive, sDiamonds), c(rSix, sDiamonds), c(rSeven, sDiamonds),
		c(rEight, sDiamonds), c(rNine, sDiamonds), c(rTen, sDiamonds),
		c(rJack, sDiamonds), c(rQueen, sDiamonds), c(rKing, sDiamonds),
		c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades),
		c(rSix, sSpades),
	})
	g.Hands[East] = cardcore.NewHand([]cardcore.Card{
		c(rAce, sDiamonds), c(rTen, sHearts), c(rJack, sHearts),
		c(rQueen, sHearts), c(rKing, sHearts), c(rAce, sHearts),
		c(rSeven, sSpades), c(rEight, sSpades), c(rNine, sSpades),
		c(rTen, sSpades), c(rJack, sSpades), c(rKing, sSpades),
		c(rAce, sSpades),
	})

	g.Turn = South
	g.Trick = Trick{Leader: South}

	return g
}
