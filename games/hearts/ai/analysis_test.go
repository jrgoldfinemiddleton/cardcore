package ai

import (
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	want := []cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Five, cardcore.Clubs),
		c(cardcore.Jack, cardcore.Clubs),
		c(cardcore.Ace, cardcore.Clubs),
	}
	for _, card := range want {
		if !a.played[card.Suit][card.Rank] {
			t.Errorf("expected %v to be marked played", card)
		}
	}

	if a.played[cardcore.Diamonds][cardcore.Two] {
		t.Error("2♦ should not be marked played")
	}
}

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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Three, cardcore.Diamonds),
				hearts.North: c(cardcore.Five, cardcore.Clubs),
				hearts.East:  c(cardcore.Nine, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
	}

	a := analyze(g, hearts.South)

	if !a.voids[hearts.West][cardcore.Clubs] {
		t.Error("West should be detected as void in clubs")
	}

	if a.voids[hearts.North][cardcore.Clubs] {
		t.Error("North followed suit, should not be marked void")
	}

	if a.voids[hearts.West][cardcore.Diamonds] {
		t.Error("West played a diamond, should not be marked void in diamonds")
	}
}

func TestAnalyzeVoidFromOwnHand(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0

	// South has only clubs and diamonds — void in hearts and spades.
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
		c(cardcore.Three, cardcore.Diamonds),
	})
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)

	a := analyze(g, hearts.South)

	if !a.voids[hearts.South][cardcore.Hearts] {
		t.Error("South has no hearts, should be void in hearts")
	}
	if !a.voids[hearts.South][cardcore.Spades] {
		t.Error("South has no spades, should be void in spades")
	}
	if a.voids[hearts.South][cardcore.Clubs] {
		t.Error("South has clubs, should not be void in clubs")
	}
	if a.voids[hearts.South][cardcore.Diamonds] {
		t.Error("South has diamonds, should not be void in diamonds")
	}
}

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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// Trick 1: spade lead, West plays Q♠. South wins with A♠.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: aceOfSpades,
				hearts.West:  queenOfSpades,
				hearts.North: c(cardcore.Five, cardcore.Spades),
				hearts.East:  c(cardcore.Nine, cardcore.Spades),
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
		c(cardcore.Ace, cardcore.Hearts),
	}

	a := analyze(g, hearts.South)

	if a.queen != queenPassed {
		t.Errorf("queen = %d, want queenPassed (%d)", a.queen, queenPassed)
	}
	if a.queenHolder != hearts.West {
		t.Errorf("queenHolder = %d, want West (%d)", a.queenHolder, hearts.West)
	}
}

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
		c(cardcore.Ace, cardcore.Hearts),
	}

	// Q♠ later appeared in trick history.
	g.TrickHistory = []hearts.Trick{
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: aceOfSpades,
				hearts.West:  queenOfSpades,
				hearts.North: c(cardcore.Five, cardcore.Spades),
				hearts.East:  c(cardcore.Nine, cardcore.Spades),
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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// Trick 1: one heart discarded.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.King, cardcore.Diamonds),
				hearts.West:  c(cardcore.Two, cardcore.Hearts),
				hearts.North: c(cardcore.Three, cardcore.Diamonds),
				hearts.East:  c(cardcore.Five, cardcore.Diamonds),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// Trick 2: two more hearts.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Ace, cardcore.Diamonds),
				hearts.West:  c(cardcore.Three, cardcore.Hearts),
				hearts.North: c(cardcore.Four, cardcore.Hearts),
				hearts.East:  c(cardcore.Six, cardcore.Diamonds),
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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// South leads A♠, West plays Q♠. South wins (highest spade).
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: aceOfSpades,
				hearts.West:  queenOfSpades,
				hearts.North: c(cardcore.Five, cardcore.Spades),
				hearts.East:  c(cardcore.Nine, cardcore.Spades),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		// South leads K♦, West discards 2♥. South wins.
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.King, cardcore.Diamonds),
				hearts.West:  c(cardcore.Two, cardcore.Hearts),
				hearts.North: c(cardcore.Three, cardcore.Diamonds),
				hearts.East:  c(cardcore.Five, cardcore.Diamonds),
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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Two, cardcore.Spades),
				hearts.West:  queenOfSpades,
				hearts.North: c(cardcore.Five, cardcore.Spades),
				hearts.East:  aceOfSpades,
			},
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Three, cardcore.Diamonds),
				hearts.West:  c(cardcore.Two, cardcore.Hearts),
				hearts.North: c(cardcore.Five, cardcore.Diamonds),
				hearts.East:  c(cardcore.Ace, cardcore.Diamonds),
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
				hearts.South: c(cardcore.Two, cardcore.Clubs),
				hearts.West:  c(cardcore.Five, cardcore.Clubs),
				hearts.North: c(cardcore.Jack, cardcore.Clubs),
				hearts.East:  c(cardcore.Ace, cardcore.Clubs),
			},
			Leader: hearts.South,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: aceOfSpades,
				hearts.West:  queenOfSpades,
				hearts.North: c(cardcore.Five, cardcore.Spades),
				hearts.East:  c(cardcore.Nine, cardcore.Spades),
			},
			Leader: hearts.East,
			Count:  hearts.NumPlayers,
		},
		{
			Cards: [hearts.NumPlayers]cardcore.Card{
				hearts.South: c(cardcore.Three, cardcore.Diamonds),
				hearts.West:  c(cardcore.Five, cardcore.Diamonds),
				hearts.North: c(cardcore.Two, cardcore.Hearts),
				hearts.East:  c(cardcore.Ace, cardcore.Diamonds),
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

func TestAnalyzeNoTrickHistory(t *testing.T) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(cardcore.Two, cardcore.Clubs),
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

// Leader played A♦, others played lower diamonds. Leader wins.
func TestCurrentWinnerLeaderWins(t *testing.T) {
	g := hearts.New()
	g.Trick = hearts.Trick{
		Leader: hearts.South,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Ace, cardcore.Diamonds),
			hearts.West:  c(cardcore.Five, cardcore.Diamonds),
			hearts.North: c(cardcore.Nine, cardcore.Diamonds),
		},
	}

	if seat, _ := currentWinner(g); seat != hearts.South {
		t.Errorf("currentWinner seat = %d, want %d (South)", seat, hearts.South)
	}
}

// West played highest diamond, beats leader and North.
func TestCurrentWinnerNonLeaderWins(t *testing.T) {
	g := hearts.New()
	g.Trick = hearts.Trick{
		Leader: hearts.South,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Five, cardcore.Diamonds),
			hearts.West:  c(cardcore.King, cardcore.Diamonds),
			hearts.North: c(cardcore.Nine, cardcore.Diamonds),
		},
	}

	if seat, _ := currentWinner(g); seat != hearts.West {
		t.Errorf("currentWinner seat = %d, want %d (West)", seat, hearts.West)
	}
}

// West played A♠ off-suit (void in diamonds). Highest diamond
// is North's 9♦, so North wins.
func TestCurrentWinnerOffSuitIgnored(t *testing.T) {
	g := hearts.New()
	g.Trick = hearts.Trick{
		Leader: hearts.South,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.South: c(cardcore.Five, cardcore.Diamonds),
			hearts.West:  c(cardcore.Ace, cardcore.Spades),
			hearts.North: c(cardcore.Nine, cardcore.Diamonds),
		},
	}

	if seat, _ := currentWinner(g); seat != hearts.North {
		t.Errorf("currentWinner seat = %d, want %d (North)", seat, hearts.North)
	}
}
