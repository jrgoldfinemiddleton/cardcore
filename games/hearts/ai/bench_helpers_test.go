package ai

import (
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// benchFixture pairs a fixture name with its builder. Used by
// TestFixturesAreLegalIntegration and every per-decision benchmark
// to share a single source of truth for the fixture set.
type benchFixture struct {
	name  string
	build func() (*hearts.Game, hearts.Seat)
}

// TestFixturesAreLegalIntegration verifies that every benchmark fixture
// produces a structurally valid game state in which the chosen seat has
// at least one legal move.
func TestFixturesAreLegalIntegration(t *testing.T) {
	for _, f := range benchFixtures() {
		g, seat := f.build()
		legal := assertLegal(g, seat)
		t.Logf("fixture %s: %d legal moves", f.name, len(legal))
	}
}

// buildLeadTrickOne returns a game in PhasePlay at TrickNum 0 where South
// is on lead and holds 2♣ (the mandatory opening card). LegalMoves must
// return exactly {2♣}.
func buildLeadTrickOne() (*hearts.Game, hearts.Seat) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 0
	g.Turn = hearts.South
	g.Trick = hearts.Trick{Leader: hearts.South}
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		twoOfClubs,
		c(rFive, sClubs),
		c(rSix, sClubs),
		c(rJack, sClubs),
		c(rNine, sDiamonds),
		c(rJack, sDiamonds),
		c(rThree, sHearts),
		c(rEight, sHearts),
		c(rKing, sHearts),
		c(rFour, sSpades),
		c(rSeven, sSpades),
		c(rTen, sSpades),
		queenOfSpades,
	})

	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rThree, sClubs),
		c(rSeven, sClubs),
		c(rNine, sClubs),
		c(rTwo, sDiamonds),
		c(rFive, sDiamonds),
		c(rSeven, sDiamonds),
		c(rTen, sDiamonds),
		c(rTwo, sHearts),
		c(rFive, sHearts),
		c(rNine, sHearts),
		c(rTwo, sSpades),
		c(rFive, sSpades),
		c(rNine, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rFour, sClubs),
		c(rEight, sClubs),
		c(rQueen, sClubs),
		c(rThree, sDiamonds),
		c(rSix, sDiamonds),
		c(rEight, sDiamonds),
		c(rQueen, sDiamonds),
		c(rFour, sHearts),
		c(rSix, sHearts),
		c(rTen, sHearts),
		c(rThree, sSpades),
		c(rSix, sSpades),
		kingOfSpades,
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rTen, sClubs),
		c(rKing, sClubs),
		c(rAce, sClubs),
		c(rFour, sDiamonds),
		c(rKing, sDiamonds),
		c(rAce, sDiamonds),
		c(rSeven, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
		c(rAce, sHearts),
		c(rEight, sSpades),
		c(rJack, sSpades),
		aceOfSpades,
	})
	return g, hearts.South
}

// buildFollowClean returns a game where West must follow a club lead in
// trick 2 with no points on the table. West holds clubs, so LegalMoves
// returns the club subset of West's hand.
func buildFollowClean() (*hearts.Game, hearts.Seat) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 1
	g.HeartsBroken = false
	g.TrickHistory = []hearts.Trick{validFirstTrick()}
	g.Turn = hearts.West
	g.Trick = hearts.Trick{
		Leader: hearts.East,
		Count:  2,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rSeven, sClubs),
			hearts.South: c(rEight, sClubs),
		},
	}
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rNine, sClubs),
		c(rJack, sClubs),
		c(rKing, sClubs),
		c(rThree, sDiamonds),
		c(rFive, sDiamonds),
		c(rTen, sDiamonds),
		c(rTwo, sHearts),
		c(rSix, sHearts),
		c(rJack, sHearts),
		c(rFour, sSpades),
		c(rEight, sSpades),
		c(rJack, sSpades),
	})

	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rSix, sClubs),
		c(rTen, sClubs),
		c(rQueen, sClubs),
		c(rTwo, sDiamonds),
		c(rFour, sDiamonds),
		c(rSeven, sDiamonds),
		c(rThree, sHearts),
		c(rFive, sHearts),
		c(rEight, sHearts),
		c(rTwo, sSpades),
		c(rThree, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rAce, sClubs),
		c(rSix, sDiamonds),
		c(rEight, sDiamonds),
		c(rNine, sDiamonds),
		c(rJack, sDiamonds),
		c(rFour, sHearts),
		c(rSeven, sHearts),
		c(rNine, sHearts),
		c(rQueen, sHearts),
		c(rFive, sSpades),
		c(rSix, sSpades),
		c(rSeven, sSpades),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sDiamonds),
		c(rKing, sDiamonds),
		c(rAce, sDiamonds),
		c(rTen, sHearts),
		c(rKing, sHearts),
		c(rAce, sHearts),
		c(rNine, sSpades),
		c(rTen, sSpades),
		queenOfSpades,
		kingOfSpades,
		aceOfSpades,
	})
	return g, hearts.West
}

// buildFollowWithPoints returns a game where East must follow a heart
// lead in trick 5; hearts are already broken and points are on the table
// (Q♠ taken in an earlier trick). East holds hearts, so LegalMoves
// returns the heart subset of East's hand.
func buildFollowWithPoints() (*hearts.Game, hearts.Seat) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 4
	g.HeartsBroken = true
	g.TrickHistory = []hearts.Trick{
		// Trick 1: validated 2♣ opener.
		validFirstTrick(),
		// Trick 2: East leads A♣, all follow clubs, East wins.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rAce, sClubs),
			hearts.South: c(rSix, sClubs),
			hearts.West:  c(rTen, sClubs),
			hearts.North: c(rKing, sClubs),
		}),
		// Trick 3: East leads 2♦, North void in diamonds dumps Q♠, West wins.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rTwo, sDiamonds),
			hearts.South: c(rFive, sDiamonds),
			hearts.West:  c(rKing, sDiamonds),
			hearts.North: queenOfSpades,
		}),
		// Trick 4: West leads A♦, North sloughs 4♥ (still void) → hearts break.
		pointTrick(hearts.West, [hearts.NumPlayers]cardcore.Card{
			hearts.West:  c(rAce, sDiamonds),
			hearts.North: c(rFour, sHearts),
			hearts.East:  c(rSix, sDiamonds),
			hearts.South: c(rNine, sDiamonds),
		}),
	}
	g.Turn = hearts.East
	g.Trick = hearts.Trick{
		Leader: hearts.West,
		Count:  2,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.West:  c(rKing, sHearts),
			hearts.North: c(rFive, sHearts),
		},
	}
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rEight, sClubs),
		c(rNine, sClubs),
		c(rThree, sDiamonds),
		c(rJack, sDiamonds),
		c(rTen, sHearts),
		c(rQueen, sHearts),
		c(rTwo, sSpades),
		c(rNine, sSpades),
		c(rJack, sSpades),
	})

	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rSeven, sClubs),
		c(rFour, sDiamonds),
		c(rSeven, sDiamonds),
		c(rEight, sDiamonds),
		c(rSix, sHearts),
		c(rEight, sHearts),
		c(rThree, sSpades),
		c(rFive, sSpades),
		c(rSix, sSpades),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sClubs),
		c(rTen, sDiamonds),
		c(rQueen, sDiamonds),
		c(rNine, sHearts),
		c(rAce, sHearts),
		c(rEight, sSpades),
		c(rTen, sSpades),
		kingOfSpades,
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rJack, sClubs),
		c(rTwo, sHearts),
		c(rThree, sHearts),
		c(rSeven, sHearts),
		c(rJack, sHearts),
		c(rFour, sSpades),
		c(rSeven, sSpades),
		aceOfSpades,
	})
	return g, hearts.East
}

// buildVoidDiscard returns a game where South must follow a diamond lead
// in trick 3 but holds no diamonds (void). LegalMoves returns South's
// off-suit cards (the discard scenario), exercising scanHandVoids.
func buildVoidDiscard() (*hearts.Game, hearts.Seat) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 2
	g.HeartsBroken = true
	g.TrickHistory = []hearts.Trick{
		validFirstTrick(),
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rAce, sClubs),
			hearts.South: c(rSix, sClubs),
			hearts.West:  c(rTen, sClubs),
			hearts.North: c(rKing, sClubs),
		}),
	}
	g.Turn = hearts.South
	g.Trick = hearts.Trick{
		Leader: hearts.East,
		Count:  1,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.East: c(rAce, sDiamonds),
		},
	}
	// South's hand contains zero diamonds — required to trip the
	// void-discard branch in validatePlay/scanHandVoids.
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rSeven, sClubs),
		c(rEight, sClubs),
		c(rJack, sClubs),
		c(rThree, sHearts),
		c(rSeven, sHearts),
		c(rJack, sHearts),
		c(rKing, sHearts),
		c(rTwo, sSpades),
		c(rFive, sSpades),
		c(rNine, sSpades),
		queenOfSpades,
	})

	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rNine, sClubs),
		c(rTwo, sDiamonds),
		c(rFour, sDiamonds),
		c(rSix, sDiamonds),
		c(rEight, sDiamonds),
		c(rTwo, sHearts),
		c(rFive, sHearts),
		c(rEight, sHearts),
		c(rThree, sSpades),
		c(rSix, sSpades),
		c(rTen, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sClubs),
		c(rThree, sDiamonds),
		c(rFive, sDiamonds),
		c(rSeven, sDiamonds),
		c(rNine, sDiamonds),
		c(rFour, sHearts),
		c(rSix, sHearts),
		c(rNine, sHearts),
		c(rFour, sSpades),
		c(rSeven, sSpades),
		c(rJack, sSpades),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rTen, sDiamonds),
		c(rJack, sDiamonds),
		c(rQueen, sDiamonds),
		c(rKing, sDiamonds),
		c(rTen, sHearts),
		c(rQueen, sHearts),
		c(rAce, sHearts),
		c(rEight, sSpades),
		kingOfSpades,
		aceOfSpades,
	})
	return g, hearts.South
}

// buildLateGameMoonThreat returns a game at TrickNum 8 where East has
// won every prior trick and collected Q♠ + 8 hearts (21 penalty points
// of 26 distributed), triggering analyze.detectMoonThreat → moonThreat
// == East. East is on lead and must choose a card.
func buildLateGameMoonThreat() (*hearts.Game, hearts.Seat) {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 8
	g.HeartsBroken = true
	g.TrickHistory = []hearts.Trick{
		// Trick 1: validated 2♣ opener.
		validFirstTrick(),
		// Trick 2: East leads A♣, all follow clubs, East wins.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rAce, sClubs),
			hearts.South: c(rSix, sClubs),
			hearts.West:  c(rTen, sClubs),
			hearts.North: c(rKing, sClubs),
		}),
		// Trick 3: East leads A♦, all follow diamonds, East wins.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rAce, sDiamonds),
			hearts.South: c(rTwo, sDiamonds),
			hearts.West:  c(rFive, sDiamonds),
			hearts.North: c(rKing, sDiamonds),
		}),
		// Trick 4: East leads A♠, all follow spades (no Q♠ yet), East wins.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  aceOfSpades,
			hearts.South: c(rTwo, sSpades),
			hearts.West:  c(rFive, sSpades),
			hearts.North: kingOfSpades,
		}),
		// Trick 5: East leads J♦, West void in diamonds sloughs Q♠, East wins.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rJack, sDiamonds),
			hearts.South: c(rThree, sDiamonds),
			hearts.West:  queenOfSpades,
			hearts.North: c(rFour, sDiamonds),
		}),
		// Trick 6: East leads Q♦, West sloughs 3♥ (still void) → hearts break.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rQueen, sDiamonds),
			hearts.South: c(rSix, sDiamonds),
			hearts.West:  c(rThree, sHearts),
			hearts.North: c(rSeven, sDiamonds),
		}),
		// Trick 7: East leads K♥ (hearts broken), all follow, East wins +3♥.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rKing, sHearts),
			hearts.South: c(rFive, sHearts),
			hearts.West:  c(rSix, sHearts),
			hearts.North: c(rSeven, sHearts),
		}),
		// Trick 8: East leads A♥, all follow, East wins +4♥.
		pointTrick(hearts.East, [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rAce, sHearts),
			hearts.South: c(rEight, sHearts),
			hearts.West:  c(rNine, sHearts),
			hearts.North: c(rTen, sHearts),
		}),
	}
	g.Turn = hearts.East
	g.Trick = hearts.Trick{Leader: hearts.East}
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rJack, sClubs),
		c(rTen, sDiamonds),
		c(rFour, sHearts),
		c(rTen, sSpades),
		c(rJack, sSpades),
	})

	// Fixture 5 South must equal fixture 6 South (buildOpponentMoonThreat
	// overrides South's hand with these same cards).
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rEight, sClubs),
		c(rNine, sDiamonds),
		c(rTwo, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rSeven, sClubs),
		c(rNine, sClubs),
		c(rThree, sSpades),
		c(rSix, sSpades),
		c(rEight, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sClubs),
		c(rEight, sDiamonds),
		c(rFour, sSpades),
		c(rSeven, sSpades),
		c(rNine, sSpades),
	})
	return g, hearts.East
}

// buildOpponentMoonThreat returns a game at TrickNum 8 where East has
// won every prior trick and collected 21 penalty points (Q♠ + 8 hearts),
// East leads T8 with 4♥, and South must decide. From South's perspective
// East is the moon threat, exercising heuristic moonBlock branches
// in followScore / voidScore / heartLeadScore.
func buildOpponentMoonThreat() (*hearts.Game, hearts.Seat) {
	g, _ := buildLateGameMoonThreat()
	g.Turn = hearts.South
	g.Trick = hearts.Trick{
		Leader: hearts.East,
		Count:  1,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.East: c(rFour, sHearts),
		},
	}
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rEight, sClubs),
		c(rNine, sDiamonds),
		c(rTwo, sHearts),
		c(rJack, sHearts),
		c(rQueen, sHearts),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rJack, sClubs),
		c(rTen, sDiamonds),
		c(rTen, sSpades),
		c(rJack, sSpades),
	})
	return g, hearts.South
}

// benchFixtures returns the canonical set of six fixtures exercised by
// the per-decision benchmarks and TestFixturesAreLegalIntegration.
func benchFixtures() []benchFixture {
	return []benchFixture{
		{"lead_trick_one", buildLeadTrickOne},
		{"follow_clean", buildFollowClean},
		{"follow_with_points", buildFollowWithPoints},
		{"void_discard", buildVoidDiscard},
		{"late_game_moon_threat", buildLateGameMoonThreat},
		{"opponent_moon_threat", buildOpponentMoonThreat},
	}
}

// assertLegal returns g.LegalMoves(seat), panicking if the call
// errors or returns no legal moves.
func assertLegal(g *hearts.Game, seat hearts.Seat) []cardcore.Card {
	legal, err := g.LegalMoves(seat)
	if err != nil {
		panic("LegalMoves error: " + err.Error())
	}
	if len(legal) == 0 {
		panic("LegalMoves returned empty for fixture")
	}
	return legal
}
