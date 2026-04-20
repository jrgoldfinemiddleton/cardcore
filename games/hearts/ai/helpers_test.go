package ai

import (
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
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

// c is a shorthand constructor for cardcore.Card.
func c(r cardcore.Rank, s cardcore.Suit) cardcore.Card {
	return cardcore.Card{Rank: r, Suit: s}
}

// pointTrick is a shorthand constructor for a complete (4-card) hearts.Trick.
// Used to keep large TrickHistory literals readable.
func pointTrick(leader hearts.Seat, cards [hearts.NumPlayers]cardcore.Card) hearts.Trick {
	return hearts.Trick{Leader: leader, Count: hearts.NumPlayers, Cards: cards}
}

// playRoundWithPlayer plays one complete round using a single Player for all
// four seats. Convenience wrapper around playRoundWithPlayers.
func playRoundWithPlayer(t *testing.T, g *hearts.Game, p hearts.Player, seed uint64) {
	t.Helper()
	playRoundWithPlayers(t, g, [hearts.NumPlayers]hearts.Player{p, p, p, p}, seed)
}

// playRoundWithPlayers plays one complete round (deal, pass, play, score) using
// a distinct Player per seat and verifies point conservation.
func playRoundWithPlayers(
	t *testing.T,
	g *hearts.Game,
	players [hearts.NumPlayers]hearts.Player,
	seed uint64,
) {
	t.Helper()

	if err := g.Deal(); err != nil {
		t.Fatalf("seed %d: Deal error: %v", seed, err)
	}

	if g.Phase == hearts.PhasePass {
		for i := hearts.Seat(0); i < hearts.NumPlayers; i++ {
			cards := players[i].ChoosePass(g.Clone(), i)
			if err := g.SetPass(i, cards); err != nil {
				t.Fatalf("seed %d: SetPass(%d) error: %v", seed, i, err)
			}
		}
	}

	for g.Phase == hearts.PhasePlay {
		seat := g.Turn
		card := players[seat].ChoosePlay(g.Clone(), seat)
		if err := g.PlayCard(seat, card); err != nil {
			t.Fatalf("seed %d: PlayCard(%d, %v) error: %v", seed, seat, card, err)
		}
	}

	roundTotal := 0
	for i := range hearts.NumPlayers {
		roundTotal += g.RoundPts[i]
	}
	if roundTotal != hearts.MoonPoints {
		t.Fatalf("seed %d: sum(RoundPts) = %d, want %d", seed, roundTotal, hearts.MoonPoints)
	}

	if err := g.EndRound(); err != nil {
		t.Fatalf("seed %d: EndRound error: %v", seed, err)
	}
}

// setupShootActiveEarlyGame builds a PhasePlay game at TrickNum 2 with the
// given hand for South (other seats empty), TrickHistory = validFirstTrick()
// followed by secondTrick. Used by TestDeriveShootActive* tests that exercise
// the early-game shoot-activation gate.
func setupShootActiveEarlyGame(southHand []cardcore.Card, secondTrick hearts.Trick) *hearts.Game {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.TrickNum = 2
	g.Hands[hearts.South] = cardcore.NewHand(southHand)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)
	g.TrickHistory = []hearts.Trick{validFirstTrick(), secondTrick}
	return g
}

// setupShootCandidateSouth builds a PhasePass game with the given hand for
// South (other seats empty). Used by TestDetectShootCandidate* tests that
// exercise the considerShoot gate.
func setupShootCandidateSouth(southHand []cardcore.Card) *hearts.Game {
	g := hearts.New()
	g.Phase = hearts.PhasePass
	g.Hands[hearts.South] = cardcore.NewHand(southHand)
	g.Hands[hearts.West] = cardcore.NewHand(nil)
	g.Hands[hearts.North] = cardcore.NewHand(nil)
	g.Hands[hearts.East] = cardcore.NewHand(nil)
	return g
}
