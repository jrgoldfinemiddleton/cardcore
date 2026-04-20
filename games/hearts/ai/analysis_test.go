package ai

import (
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// TestAnalyzePlayedCards verifies that cards from completed tricks are
// marked as played in the analysis.
func TestAnalyzePlayedCards(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 1
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	g.TrickHistory = []hearts.Trick{
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rTwo, sClubs),
				hearts.West:  c(rFive, sClubs),
				hearts.North: c(rJack, sClubs),
				hearts.East:  c(rAce, sClubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	want := []cardcore.Card{
		c(rTwo, sClubs),
		c(rFive, sClubs),
		c(rJack, sClubs),
		c(rAce, sClubs),
	}
	for _, card := range want {
		if !a.played[card.Suit][card.Rank] {
			t.Errorf("a.played[%v] = false, want true", card)
		}
	}

	if a.played[sDiamonds][rTwo] {
		t.Error("2♦ should not be marked played")
	}
}

// TestAnalyzeVoidDetection verifies that failing to follow suit marks a
// player as void. West played a diamond when clubs were led.
func TestAnalyzeVoidDetection(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 1
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	// West played a diamond when clubs were led — void in clubs.
	g.TrickHistory = []hearts.Trick{
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rTwo, sClubs),
				hearts.West:  c(rThree, sDiamonds),
				hearts.North: c(rFive, sClubs),
				hearts.East:  c(rNine, sClubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	if !a.voids[hearts.West][sClubs] {
		t.Error("West should be detected as void in clubs")
	}

	if a.voids[hearts.North][sClubs] {
		t.Error("North followed suit, should not be marked void")
	}

	if a.voids[hearts.West][sDiamonds] {
		t.Error("West played a diamond, should not be marked void in diamonds")
	}
}

// TestAnalyzeVoidFromOwnHand verifies that suits missing from the player's
// own hand are marked as void.
func TestAnalyzeVoidFromOwnHand(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0

	// South has only clubs and diamonds — void in hearts and spades.
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sDiamonds),
	})
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.South)

	if !a.voids[hearts.South][sHearts] {
		t.Error("South has no hearts, should be void in hearts")
	}
	if !a.voids[hearts.South][sSpades] {
		t.Error("South has no spades, should be void in spades")
	}
	if a.voids[hearts.South][sClubs] {
		t.Error("South has clubs, should not be void in clubs")
	}
	if a.voids[hearts.South][sDiamonds] {
		t.Error("South has diamonds, should not be void in diamonds")
	}
}

// TestAnalyzeQueenInHand verifies that holding Q♠ sets queen to queenInHand.
func TestAnalyzeQueenInHand(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{queenOfSpades})
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.West)

	if a.queen != queenInHand {
		t.Errorf("queen = %d, want queenInHand (%d)", a.queen, queenInHand)
	}
}

// TestAnalyzeQueenPlayed verifies that Q♠ appearing in trick history sets
// queen to queenPlayed for all observers.
func TestAnalyzeQueenPlayed(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 2
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	g.TrickHistory = []hearts.Trick{
		// Trick 0: valid 2♣ lead.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rTwo, sClubs),
				hearts.West:  c(rFive, sClubs),
				hearts.North: c(rJack, sClubs),
				hearts.East:  c(rAce, sClubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// Trick 1: spade lead, West plays Q♠. South wins with A♠.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: aceOfSpades,
				hearts.West:  queenOfSpades,
				hearts.North: c(rFive, sSpades),
				hearts.East:  c(rNine, sSpades),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	// Both the observer and the player who played Q♠ should see queenPlayed.
	a := analyze(g, hearts.South)
	if a.queen != queenPlayed {
		t.Errorf("South: queen = %d, want queenPlayed (%d)", a.queen, queenPlayed)
	}

	a = analyze(g, hearts.West)
	if a.queen != queenPlayed {
		t.Errorf("West (played Q♠): queen = %d, want queenPlayed (%d)", a.queen, queenPlayed)
	}
}

// TestAnalyzeQueenPassed verifies that passing Q♠ sets queen to queenPassed
// with the correct queenHolder (pass target).
func TestAnalyzeQueenPassed(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassLeft
	g.TrickNum = 0
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	g.PassHistory[hearts.South] = [hearts.PassCount]cardcore.Card{
		queenOfSpades,
		kingOfSpades,
		c(rAce, sHearts),
	}

	a := analyze(g, hearts.South)

	if a.queen != queenPassed {
		t.Errorf("queen = %d, want queenPassed (%d)", a.queen, queenPassed)
	}
	if a.queenHolder != hearts.West {
		t.Errorf("queenHolder = %d, want West (%d)", a.queenHolder, hearts.West)
	}
}

// TestAnalyzeQueenPassedThenPlayed verifies that queenPlayed takes priority
// over queenPassed when Q♠ appears in both pass history and trick history.
func TestAnalyzeQueenPassedThenPlayed(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassLeft
	g.TrickNum = 2
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	// South passed Q♠ to West.
	g.PassHistory[hearts.South] = [hearts.PassCount]cardcore.Card{
		queenOfSpades,
		kingOfSpades,
		c(rAce, sHearts),
	}

	// Q♠ later appeared in trick history.
	g.TrickHistory = []hearts.Trick{
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rTwo, sClubs),
				hearts.West:  c(rFive, sClubs),
				hearts.North: c(rJack, sClubs),
				hearts.East:  c(rAce, sClubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: aceOfSpades,
				hearts.West:  queenOfSpades,
				hearts.North: c(rFive, sSpades),
				hearts.East:  c(rNine, sSpades),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	if a.queen != queenPlayed {
		t.Errorf("queen = %d, want queenPlayed (%d) — played overrides passed",
			a.queen, queenPlayed)
	}
}

// TestAnalyzeQueenReceivedViaPass verifies that receiving Q♠ via a pass
// results in queenInHand (the hand check runs before pass history).
func TestAnalyzeQueenReceivedViaPass(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassLeft
	g.TrickNum = 0

	// West received Q♠ (it's in their hand now).
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{queenOfSpades})
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.West)

	if a.queen != queenInHand {
		t.Errorf("queen = %d, want queenInHand (%d)", a.queen, queenInHand)
	}
}

// TestAnalyzeQueenUnknown verifies that Q♠ defaults to queenUnknown when
// not in hand, not passed, and not played.
func TestAnalyzeQueenUnknown(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.South)

	if a.queen != queenUnknown {
		t.Errorf("queen = %d, want queenUnknown (%d)", a.queen, queenUnknown)
	}
}

// TestAnalyzeQueenHoldRound verifies that Q♠ remains queenUnknown during
// a hold round (no passing occurs).
func TestAnalyzeQueenHoldRound(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassHold
	g.TrickNum = 0
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.South)

	if a.queen != queenUnknown {
		t.Errorf("queen = %d, want queenUnknown (%d) on hold round", a.queen, queenUnknown)
	}
}

// TestAnalyzeHeartsPlayedAcrossMultipleTricks verifies that heartsPlayed
// accumulates across multiple tricks. Three hearts across tricks 1-2.
func TestAnalyzeHeartsPlayedAcrossMultipleTricks(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 3
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	g.TrickHistory = []hearts.Trick{
		// Trick 0: no hearts.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rTwo, sClubs),
				hearts.West:  c(rFive, sClubs),
				hearts.North: c(rJack, sClubs),
				hearts.East:  c(rAce, sClubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// Trick 1: one heart discarded.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rKing, sDiamonds),
				hearts.West:  c(rTwo, sHearts),
				hearts.North: c(rThree, sDiamonds),
				hearts.East:  c(rFive, sDiamonds),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// Trick 2: two more hearts.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rAce, sDiamonds),
				hearts.West:  c(rThree, sHearts),
				hearts.North: c(rFour, sHearts),
				hearts.East:  c(rSix, sDiamonds),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	if a.heartsPlayed != 3 {
		t.Errorf("heartsPlayed = %d, want 3", a.heartsPlayed)
	}
}

// TestAnalyzePointsTaken verifies that pointsTaken tracks penalty points
// per seat. South wins Q♠ (13) and 2♥ (1) = 14 points.
func TestAnalyzePointsTaken(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 3
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	g.TrickHistory = []hearts.Trick{
		pointTrick(hearts.South, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rTwo, sClubs),
			hearts.West:  c(rFive, sClubs),
			hearts.North: c(rJack, sClubs),
			hearts.East:  c(rAce, sClubs),
		}),
		// South leads A♠, West plays Q♠. South wins (highest spade).
		pointTrick(hearts.South, [hearts.NumPlayers]cardcore.Card{
			hearts.South: aceOfSpades,
			hearts.West:  queenOfSpades,
			hearts.North: c(rFive, sSpades),
			hearts.East:  c(rNine, sSpades),
		}),
		// South leads K♦, West discards 2♥. South wins.
		pointTrick(hearts.South, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rKing, sDiamonds),
			hearts.West:  c(rTwo, sHearts),
			hearts.North: c(rThree, sDiamonds),
			hearts.East:  c(rFive, sDiamonds),
		}),
	}

	a := analyze(g, hearts.North)

	if a.pointsTaken[hearts.South] != 14 {
		t.Errorf("pointsTaken[South] = %d, want 14", a.pointsTaken[hearts.South])
	}
	if a.pointsTaken[hearts.West] != 0 {
		t.Errorf("pointsTaken[West] = %d, want 0", a.pointsTaken[hearts.West])
	}
}

// TestAnalyzeMoonThreat verifies that moonThreat identifies the seat
// holding all distributed penalty points. East wins both point tricks.
func TestAnalyzeMoonThreat(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 3
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	// East wins both point tricks and collects all distributed points.
	g.TrickHistory = []hearts.Trick{
		pointTrick(hearts.South, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rTwo, sClubs),
			hearts.West:  c(rFive, sClubs),
			hearts.North: c(rJack, sClubs),
			hearts.East:  c(rAce, sClubs),
		}),
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rTwo, sSpades),
			hearts.West:  queenOfSpades,
			hearts.North: c(rFive, sSpades),
			hearts.East:  aceOfSpades,
		}),
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rThree, sDiamonds),
			hearts.West:  c(rTwo, sHearts),
			hearts.North: c(rFive, sDiamonds),
			hearts.East:  c(rAce, sDiamonds),
		}),
	}

	a := analyze(g, hearts.South)

	if a.moonThreat != int(hearts.East) {
		t.Errorf("moonThreat = %d, want %d (East)", a.moonThreat, hearts.East)
	}
}

// TestAnalyzeMoonThreatSplit verifies that moonThreat is -1 when penalty
// points are split between multiple players.
func TestAnalyzeMoonThreatSplit(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 3
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	// Points split between South and East — no moon threat.
	g.TrickHistory = []hearts.Trick{
		pointTrick(hearts.South, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rTwo, sClubs),
			hearts.West:  c(rFive, sClubs),
			hearts.North: c(rJack, sClubs),
			hearts.East:  c(rAce, sClubs),
		}),
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.South: aceOfSpades,
			hearts.West:  queenOfSpades,
			hearts.North: c(rFive, sSpades),
			hearts.East:  c(rNine, sSpades),
		}),
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rThree, sDiamonds),
			hearts.West:  c(rFive, sDiamonds),
			hearts.North: c(rTwo, sHearts),
			hearts.East:  c(rAce, sDiamonds),
		}),
	}

	a := analyze(g, hearts.South)

	if a.moonThreat != -1 {
		t.Errorf("moonThreat = %d, want -1 (no threat)", a.moonThreat)
	}
}

// TestAnalyzeNoTrickHistory verifies the zero-value analysis state with no
// trick history: no hearts played, no moon threat, queen unknown.
func TestAnalyzeNoTrickHistory(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
	})
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.South)

	if a.heartsPlayed != 0 {
		t.Errorf("heartsPlayed = %d, want 0", a.heartsPlayed)
	}
	if a.moonThreat != -1 {
		t.Errorf("moonThreat = %d, want -1", a.moonThreat)
	}
	if a.queen != queenUnknown {
		t.Errorf("queen = %d, want queenUnknown (%d)", a.queen, queenUnknown)
	}
	for s := range hearts.NumPlayers {
		if a.pointsTaken[s] != 0 {
			t.Errorf("pointsTaken[%d] = %d, want 0", s, a.pointsTaken[s])
		}
	}
}

// TestCurrentWinnerLeaderWins verifies that the leader wins when they play
// the highest card of the led suit.
func TestCurrentWinnerLeaderWins(t *testing.T) {
	g := hearts.New()
	g.Trick = hearts.Trick{
		Leader: hearts.South,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rAce, sDiamonds),
			hearts.West:  c(rFive, sDiamonds),
			hearts.North: c(rNine, sDiamonds),
		},
	}

	if seat, _ := currentWinner(g); seat != hearts.South {
		t.Errorf("currentWinner seat = %d, want %d (South)", seat, hearts.South)
	}
}

// TestCurrentWinnerNonLeaderWins verifies that a non-leader wins when
// they play the highest card of the led suit.
func TestCurrentWinnerNonLeaderWins(t *testing.T) {
	g := hearts.New()
	g.Trick = hearts.Trick{
		Leader: hearts.South,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFive, sDiamonds),
			hearts.West:  c(rKing, sDiamonds),
			hearts.North: c(rNine, sDiamonds),
		},
	}

	if seat, _ := currentWinner(g); seat != hearts.West {
		t.Errorf("currentWinner seat = %d, want %d (West)", seat, hearts.West)
	}
}

// TestCurrentWinnerOffSuitIgnored verifies that off-suit cards are ignored
// when determining the trick winner.
func TestCurrentWinnerOffSuitIgnored(t *testing.T) {
	g := hearts.New()
	g.Trick = hearts.Trick{
		Leader: hearts.South,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFive, sDiamonds),
			hearts.West:  c(rAce, sSpades),
			hearts.North: c(rNine, sDiamonds),
		},
	}

	if seat, _ := currentWinner(g); seat != hearts.North {
		t.Errorf("currentWinner seat = %d, want %d (North)", seat, hearts.North)
	}
}

// --- Shoot detection tests ---

// TestDetectShootCandidateFullChecklist verifies that considerShoot is true
// when the hand has A♥, K♥, Q♥, 4+ more hearts, and a side ace. Hand: 7
// hearts (A♥ K♥ Q♥ J♥ 10♥ 9♥ 8♥) + A♣ + 5 low clubs.
func TestDetectShootCandidateFullChecklist(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePass
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
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
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.South)

	if !a.considerShoot {
		t.Error("considerShoot should be true: A♥ K♥ Q♥ + 4 more hearts + A♣")
	}
}

// TestDetectShootCandidateTooFewHearts verifies that considerShoot is false
// when the hand has fewer than 7 hearts. Hand: 6 hearts (A♥ K♥ Q♥ J♥
// 10♥ 9♥) + A♣ + 6 low clubs.
func TestDetectShootCandidateTooFewHearts(t *testing.T) {
	g := setupShootCandidateSouth([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rAce, sClubs),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	})

	a := analyze(g, hearts.South)

	if a.considerShoot {
		t.Error("considerShoot should be false: only 6 hearts")
	}
}

// TestDetectShootCandidateMissingQueenOfHearts verifies that considerShoot
// is false when Q♥ is absent. Hand: 7 hearts
// (A♥ K♥ J♥ 10♥ 9♥ 8♥ 7♥) + A♣ + 5 clubs.
func TestDetectShootCandidateMissingQueenOfHearts(t *testing.T) {
	g := setupShootCandidateSouth([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rAce, sClubs),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	})

	a := analyze(g, hearts.South)

	if a.considerShoot {
		t.Error("considerShoot should be false: missing Q♥")
	}
}

// TestDetectShootCandidateNoSideAce verifies that considerShoot is false
// when no non-heart ace is present. Hand: 7 hearts
// (A♥ K♥ Q♥ J♥ 10♥ 9♥ 8♥) + K♣ + 5 low clubs.
func TestDetectShootCandidateNoSideAce(t *testing.T) {
	g := setupShootCandidateSouth([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rKing, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	})

	a := analyze(g, hearts.South)

	if a.considerShoot {
		t.Error("considerShoot should be false: no side ace")
	}
}

// TestDeriveShootActiveMoonThreatIsSelf verifies that shootActive is true
// when the seat is the moon threat (has all distributed penalty points),
// originally held ≥7 hearts, and holds the highest unplayed heart (A♥).
func TestDeriveShootActiveMoonThreatIsSelf(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 7
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rAce, sHearts),
	})
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.TrickHistory = moonThreatHistory()

	a := analyze(g, hearts.East)

	if !a.shootActive {
		t.Error("shootActive should be true: East is the moon threat")
	}
}

// TestDeriveShootActiveNoPointsStrongHand verifies that shootActive is true
// when no penalty points have been distributed and the hand still holds
// A♥, K♥, and Q♥.
func TestDeriveShootActiveNoPointsStrongHand(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 2
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
	})
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)
	g.TrickHistory = []hearts.Trick{
		validFirstTrick(),
		{
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rFive, sClubs),
				hearts.West:  c(rSix, sClubs),
				hearts.North: c(rSeven, sClubs),
				hearts.East:  c(rEight, sClubs),
			},
		},
	}

	a := analyze(g, hearts.South)

	if !a.shootActive {
		t.Error("shootActive should be true: no points distributed, A♥ + K♥ + Q♥ in hand")
	}
}

// TestDeriveShootActivePointsDistributed verifies that shootActive is false
// when penalty points have been distributed to another player, even with a
// strong hand. South has 7 hearts (passes the originalHearts ≥ 7 gate), but
// East took 2♥ in trick history so totalDistributed > 0 blocks activation.
// Branch: early-game path, totalDistributed > 0 exit.
func TestDeriveShootActivePointsDistributed(t *testing.T) {
	g := setupShootActiveEarlyGame([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
	}, hearts.Trick{
		Leader: hearts.East,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFive, sDiamonds),
			hearts.West:  c(rTwo, sHearts),
			hearts.North: c(rSix, sDiamonds),
			hearts.East:  c(rAce, sDiamonds),
		},
	})

	a := analyze(g, hearts.South)

	if a.shootActive {
		t.Error("shootActive should be false: East took a heart")
	}
}

// TestDeriveShootActiveNoPointsMissingTopHearts verifies that shootActive is
// false when no points are distributed but the hand lacks A♥ and K♥. South
// has 7 hearts (passes the originalHearts ≥ 7 gate) and
// totalDistributed == 0, but the A♥+K♥+Q♥ check fails because A♥
// and K♥ are missing.
// Branch: early-game path, A♥+K♥+Q♥ check.
func TestDeriveShootActiveNoPointsMissingTopHearts(t *testing.T) {
	g := setupShootActiveEarlyGame([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rAce, sClubs),
		c(rTwo, sHearts),
		c(rThree, sHearts),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
	}, hearts.Trick{
		Leader: hearts.East,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFive, sDiamonds),
			hearts.West:  c(rSix, sDiamonds),
			hearts.North: c(rSeven, sDiamonds),
			hearts.East:  c(rEight, sDiamonds),
		},
	})

	a := analyze(g, hearts.South)

	if a.shootActive {
		t.Error("shootActive should be false: missing A♥ and K♥")
	}
}

// TestDeriveShootActiveNoPointsMissingQueenHearts verifies that shootActive
// is false when no points are distributed and the hand holds A♥+K♥ but
// not Q♥. South has 7 hearts (A♥, K♥, J♥, 10♥, 9♥, 8♥, 7♥) plus
// 4 clubs. Without Q♥, the early-game gate rejects shoot activation.
func TestDeriveShootActiveNoPointsMissingQueenHearts(t *testing.T) {
	g := setupShootActiveEarlyGame([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rFour, sClubs),
		c(rFive, sClubs),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rNine, sHearts),
		c(rTen, sHearts),
		c(rJack, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
	}, hearts.Trick{
		Leader: hearts.East,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rFive, sDiamonds),
			hearts.West:  c(rSix, sDiamonds),
			hearts.North: c(rSeven, sDiamonds),
			hearts.East:  c(rEight, sDiamonds),
		},
	})

	a := analyze(g, hearts.South)

	if a.shootActive {
		t.Error("shootActive should be false: missing Q♥")
	}
}

// TestDeriveShootActiveMoonThreatLowOriginalHearts verifies that shootActive
// is false when the seat is the moon threat but originally held fewer than 7
// hearts. East accidentally collected all points with only 3 hearts in hand
// (A♥, K♥, Q♥). East IS the moon threat and holds A♥ (the highest
// unplayed heart), so the moon-threat path would activate —
// but originalHearts < 7 rejects it before that check is reached.
// Branch: originalHearts < 7 early exit.
func TestDeriveShootActiveMoonThreatLowOriginalHearts(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 7
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rAce, sHearts),
		c(rKing, sHearts),
		c(rQueen, sHearts),
		c(rFour, sSpades),
		c(rTen, sSpades),
		c(rJack, sSpades),
	})
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.TrickHistory = moonThreatHistory()

	a := analyze(g, hearts.East)

	if a.shootActive {
		t.Error("shootActive should be false: originalHearts < 7 (accidental collector)")
	}
}

// TestDeriveShootActiveMoonThreatNoHighestHeart verifies that shootActive is
// false when the seat is the moon threat with ≥7 original hearts but does
// not hold the highest unplayed heart. East has 3♥–8♥ (6 in hand),
// played A♥ in trick history (originalHearts = 7). K♥ is unplayed
// and not in East's hand.
func TestDeriveShootActiveMoonThreatNoHighestHeart(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 7
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rThree, sHearts),
		c(rFour, sHearts),
		c(rFive, sHearts),
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
	})
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	// Use moonThreatHistory (East wins all points). Add a hearts-led
	// trick where East plays A♥ and wins, so heartsFromSeat = 1 and
	// originalHearts = 6 + 1 = 7. K♥ is the highest unplayed heart
	// and East doesn't hold it.
	h := moonThreatHistory()
	h = append(h, hearts.Trick{
		Leader: hearts.North,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(rNine, sHearts),
			hearts.West:  c(rTen, sHearts),
			hearts.North: c(rJack, sHearts),
			hearts.East:  c(rAce, sHearts),
		},
	})
	g.TrickHistory = h
	g.TrickNum = 8

	a := analyze(g, hearts.East)

	if a.shootActive {
		t.Error("shootActive should be false: East does not hold highest unplayed heart (K♥)")
	}
}

// TestDeriveShootActiveHeartsFromSeatReconstruction verifies that
// originalHearts is correctly reconstructed from current hand plus hearts
// played in trick history. East has 5 hearts in hand and played 2 hearts in
// tricks, so originalHearts = 7. East holds A♥ (highest unplayed) and is
// moon threat.
func TestDeriveShootActiveHeartsFromSeatReconstruction(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 9
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rSix, sHearts),
		c(rSeven, sHearts),
		c(rEight, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
	})
	g.Hands[hearts.South] = cardcore.NewHand(nil)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	// moonThreatHistory has East winning all points (1 pt from West's 2♥).
	// Add two tricks where East played 9♥ and 10♥, so heartsFromSeat = 2.
	// originalHearts = 5 (in hand) + 2 (played) = 7.
	h := moonThreatHistory()
	h = append(h,
		hearts.Trick{
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rFive, sSpades),
				hearts.West:  c(rSix, sSpades),
				hearts.North: c(rSeven, sSpades),
				hearts.East:  c(rNine, sHearts),
			},
		},
		hearts.Trick{
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rEight, sSpades),
				hearts.West:  c(rNine, sSpades),
				hearts.North: c(rJack, sSpades),
				hearts.East:  c(rTen, sHearts),
			},
		},
	)
	g.TrickHistory = h

	a := analyze(g, hearts.East)

	if !a.shootActive {
		t.Error("shootActive should be true: originalHearts = 5 + 2 = 7, holds A♥, is moon threat")
	}
}

// TestHoldsHighestHeartMidRankPositive verifies that holdsHighestHeart
// returns true when top hearts (A♥ through 6♥) have been played and the
// seat holds 5♥ — the highest remaining heart. Constructs the analysis
// struct directly to isolate holdsHighestHeart from the rest of analyze().
// Branch: holdsHighestHeart loop finds 5♥ as first unplayed rank.
func TestHoldsHighestHeartMidRankPositive(t *testing.T) {
	a := analysis{}
	// Mark A♥ through 6♥ as played.
	for rank := rAce; rank >= rSix; rank-- {
		a.played[sHearts][rank] = true
	}

	// Seat holds 5♥ — the highest unplayed heart.
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rTwo, sHearts),
		c(rThree, sHearts),
		c(rFive, sHearts),
	})

	if !a.holdsHighestHeart(hand) {
		t.Error("holdsHighestHeart should be true: " +
			"5♥ is the highest unplayed heart and seat holds it")
	}
}

// TestHoldsHighestHeartMidRankNegative verifies that holdsHighestHeart
// returns false when top hearts (A♥ through 6♥) have been played and the
// seat does NOT hold 5♥ — the highest remaining heart. Same played state
// as the positive test, but hand has 4♥ instead of 5♥.
// Branch: holdsHighestHeart loop finds 5♥ as first unplayed rank,
// seat lacks it.
func TestHoldsHighestHeartMidRankNegative(t *testing.T) {
	a := analysis{}
	// Mark A♥ through 6♥ as played.
	for rank := rAce; rank >= rSix; rank-- {
		a.played[sHearts][rank] = true
	}

	// Seat holds 4♥ but NOT 5♥.
	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
		c(rTwo, sHearts),
		c(rThree, sHearts),
		c(rFour, sHearts),
	})

	if a.holdsHighestHeart(hand) {
		t.Error("holdsHighestHeart should be false: " +
			"5♥ is the highest unplayed heart but seat holds 4♥, not 5♥")
	}
}

// TestHoldsHighestHeartAllPlayed verifies that holdsHighestHeart returns
// false (without panicking) when every heart has already been played.
// Regression test for a uint8 underflow on the rank loop variable that
// caused an index-out-of-range panic at a.played[Hearts][255].
// Branch: holdsHighestHeart loop scans every rank without finding an
// unplayed heart and must terminate cleanly at Two.
func TestHoldsHighestHeartAllPlayed(t *testing.T) {
	a := analysis{}
	for rank := rTwo; rank <= rAce; rank++ {
		a.played[sHearts][rank] = true
	}

	hand := cardcore.NewHand([]cardcore.Card{
		c(rTwo, sClubs),
		c(rThree, sClubs),
	})

	if a.holdsHighestHeart(hand) {
		t.Error("holdsHighestHeart should be false: no hearts remain unplayed")
	}
}
