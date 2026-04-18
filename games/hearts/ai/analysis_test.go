package ai

import (
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// TestAnalyzePlayedCards verifies that cards from completed tricks are marked as played in the analysis.
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
			t.Errorf("expected %v to be marked played", card)
		}
	}

	if a.played[sDiamonds][rTwo] {
		t.Error("2♦ should not be marked played")
	}
}

// TestAnalyzeVoidDetection verifies that failing to follow suit marks a player as void. West played a diamond when clubs were led.
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

// TestAnalyzeVoidFromOwnHand verifies that suits missing from the player's own hand are marked as void.
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

// TestAnalyzeQueenPlayed verifies that Q♠ appearing in trick history sets queen to queenPlayed for all observers.
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

// TestAnalyzeQueenPassed verifies that passing Q♠ sets queen to queenPassed with the correct queenHolder (pass target).
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

// TestAnalyzeQueenPassedThenPlayed verifies that queenPlayed takes priority over queenPassed when Q♠ appears in both pass history and trick history.
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
		t.Errorf("queen = %d, want queenPlayed (%d) — played overrides passed", a.queen, queenPlayed)
	}
}

// TestAnalyzeQueenReceivedViaPass verifies that receiving Q♠ via a pass results in queenInHand (the hand check runs before pass history).
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

// TestAnalyzeQueenUnknown verifies that Q♠ defaults to queenUnknown when not in hand, not passed, and not played.
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

// TestAnalyzeQueenHoldRound verifies that Q♠ remains queenUnknown during a hold round (no passing occurs).
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

// TestAnalyzeHeartsPlayedAcrossMultipleTricks verifies that heartsPlayed accumulates across multiple tricks. Three hearts across tricks 1-2.
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

// TestAnalyzePointsTaken verifies that pointsTaken tracks penalty points per seat. South wins Q♠ (13) and 2♥ (1) = 14 points.
func TestAnalyzePointsTaken(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 3
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
		// South leads A♠, West plays Q♠. South wins (highest spade).
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
		// South leads K♦, West discards 2♥. South wins.
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
	}

	a := analyze(g, hearts.North)

	if a.pointsTaken[hearts.South] != 14 {
		t.Errorf("pointsTaken[South] = %d, want 14", a.pointsTaken[hearts.South])
	}
	if a.pointsTaken[hearts.West] != 0 {
		t.Errorf("pointsTaken[West] = %d, want 0", a.pointsTaken[hearts.West])
	}
}

// TestAnalyzeMoonThreat verifies that moonThreat identifies the seat holding all distributed penalty points. East wins both point tricks.
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
				hearts.South: c(rTwo, sSpades),
				hearts.West:  queenOfSpades,
				hearts.North: c(rFive, sSpades),
				hearts.East:  aceOfSpades,
			},
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rThree, sDiamonds),
				hearts.West:  c(rTwo, sHearts),
				hearts.North: c(rFive, sDiamonds),
				hearts.East:  c(rAce, sDiamonds),
			},
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	if a.moonThreat != int(hearts.East) {
		t.Errorf("moonThreat = %d, want %d (East)", a.moonThreat, hearts.East)
	}
}

// TestAnalyzeMoonThreatSplit verifies that moonThreat is -1 when penalty points are split between multiple players.
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
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(rThree, sDiamonds),
				hearts.West:  c(rFive, sDiamonds),
				hearts.North: c(rTwo, sHearts),
				hearts.East:  c(rAce, sDiamonds),
			},
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	if a.moonThreat != -1 {
		t.Errorf("moonThreat = %d, want -1 (no threat)", a.moonThreat)
	}
}

// TestAnalyzeNoTrickHistory verifies the zero-value analysis state with no trick history: no hearts played, no moon threat, queen unknown.
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

// TestCurrentWinnerLeaderWins verifies that the leader wins when they play the highest card of the led suit.
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

// TestCurrentWinnerNonLeaderWins verifies that a non-leader wins when they play the highest card of the led suit.
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
