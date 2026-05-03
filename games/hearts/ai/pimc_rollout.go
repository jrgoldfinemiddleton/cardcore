package ai

import (
	"fmt"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// rollout plays out the remainder of the round on a clone of g and
// returns the leaf score for seat. seat plays candidate as its next
// card, then policy drives every subsequent decision for all four
// seats (including seat itself) until PhasePlay ends.
//
// g is never mutated. The deal's entries for seats other than seat
// replace those seats' hands in the clone before play resumes; the entry
// for seat must equal seat's real hand (caller's contract, not
// re-validated here).
//
// policy carries its own RNG; the caller constructs it via
// rolloutFactory with a per-(sample, candidate) derived RNG so that
// stochastic policies produce independent streams across candidates.
// rollout itself is RNG-agnostic.
//
// Panics if g is not in PhasePlay, if g.Turn != seat, if candidate is
// not a legal move, or if any engine call fails during rollout (engine
// invariant violation; programmer error).
func rollout(
	g *hearts.Game,
	seat hearts.Seat,
	candidate cardcore.Card,
	deal sampledDeal,
	policy hearts.Player,
) int {
	if g.Phase != hearts.PhasePlay {
		panic(fmt.Sprintf("ai: rollout requires PhasePlay, got phase %d", g.Phase))
	}
	if g.Turn != seat {
		panic(fmt.Sprintf("ai: rollout requires Turn == seat (%d), got Turn %d", seat, g.Turn))
	}
	assertDealMatchesSeatHand(deal, seat, g.Hands[seat])

	clone := g.Clone()
	for s := range hearts.NumPlayers {
		if hearts.Seat(s) == seat {
			continue
		}
		clone.Hands[s] = cardcore.NewHand(deal[s].Cards)
	}

	if err := clone.PlayCard(seat, candidate); err != nil {
		panic(fmt.Errorf("ai: rollout candidate play failed: %w", err))
	}

	for clone.Phase == hearts.PhasePlay {
		turn := clone.Turn
		card := policy.ChoosePlay(clone.Clone(), turn)
		if err := clone.PlayCard(turn, card); err != nil {
			panic(fmt.Errorf("ai: rollout policy play failed: %w", err))
		}
	}

	return leafScore(clone.RoundPts, seat)
}

// leafScore computes the round score for seat from the final round
// point totals. It re-implements the moon flip locally rather than
// reading g.Scores so that PIMC's leaf evaluation is self-contained
// and insulated from any future engine scoring changes.
//
// If some seat shot the moon (RoundPts == MoonPoints), the leaf score
// is 0 if seat is the shooter, else MoonPoints. Otherwise the leaf
// score is RoundPts[seat]. Lower is better.
func leafScore(roundPts [hearts.NumPlayers]int, seat hearts.Seat) int {
	for s, pts := range roundPts {
		if pts == hearts.MoonPoints {
			if hearts.Seat(s) == seat {
				return 0
			}
			return hearts.MoonPoints
		}
	}
	return roundPts[seat]
}

// assertDealMatchesSeatHand panics if deal[seat] does not exactly match
// seat's real hand. A sampled deal must preserve seat's known hand and
// only invent the opponents'. A mismatch means a buggy caller; turning
// it into a loud panic prevents silent contract violations (rollout
// would otherwise discard deal[seat] and use g.Hands[seat] from the
// clone, masking the bug).
func assertDealMatchesSeatHand(deal sampledDeal, seat hearts.Seat, real *cardcore.Hand) {
	got := deal[seat].Cards
	want := real.Cards
	if len(got) != len(want) {
		panic(fmt.Sprintf(
			"ai: rollout deal[seat=%d] has %d cards, want %d (caller contract violation)",
			seat, len(got), len(want),
		))
	}
	gotSet := make(map[cardcore.Card]bool, len(got))
	for _, c := range got {
		gotSet[c] = true
	}
	for _, c := range want {
		if !gotSet[c] {
			panic(fmt.Sprintf(
				"ai: rollout deal[seat=%d] missing card %v (caller contract violation)",
				seat, c,
			))
		}
	}
}
