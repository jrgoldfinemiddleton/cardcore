package tienlen

import (
	"fmt"
	"slices"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// LegalMoves returns every combination seat may currently play through
// the in-turn channel — a Play call on their turn. (Out-of-turn chops
// are the interrupt channel: see InterruptChop.) A valid seat that
// cannot act receives an empty slice without an error — queries answer,
// Play enforces. A locked-out seat has no in-turn plays under Killer
// rules; a variant whose locked-out players may chop (see rulesFor)
// yields exactly those chops instead.
//
// The result order is API-stable: Kind declaration order, card count,
// Top under Tiến Lên card order, then the full Cards slice
// lexicographically. Each result is freshly built; mutating it cannot
// affect the game.
func (g *Game) LegalMoves(seat Seat) ([]Combo, error) {
	if err := g.validateMoveQuery(seat); err != nil {
		return nil, err
	}
	if seat != g.Turn || !g.Active[seat] || g.Hands[seat] == nil || g.Hands[seat].Len() == 0 {
		return []Combo{}, nil
	}
	if g.Pile.Open && g.Pile.Locked[seat] && !g.rules.lockedMayChop {
		return []Combo{}, nil
	}

	candidates := enumerateCombinations(g.Hands[seat].Cards)

	// Return all candidates if the pile is open and the seat can self-beat, or if
	// the pile is closed and the seat is not required to lead with the opening card.
	if g.SelfBeatContinuation || (!g.Pile.Open && !g.OpeningLeadRequired) {
		return candidates, nil
	}
	moves := make([]Combo, 0, len(candidates))
	for _, candidate := range candidates {
		switch {
		case !g.Pile.Open && containsCard(candidate.Cards, g.OpeningCard):
			moves = append(moves, candidate)
		case g.Pile.Open && g.Pile.Locked[seat] && candidate.Chops(g.Pile.Top, g.Config.Variant):
			// A locked-out player may still chop. Killer rules never
			// reach this branch (see the early return above).
			moves = append(moves, candidate)
		case g.Pile.Open && !g.Pile.Locked[seat] && candidate.CanBeat(g.Pile.Top, g.Config.Variant):
			moves = append(moves, candidate)
		}
	}
	return moves, nil
}

// CanPass reports whether seat may pass through the in-turn channel. A valid
// seat that cannot act receives false without an error.
func (g *Game) CanPass(seat Seat) (bool, error) {
	if err := g.validateMoveQuery(seat); err != nil {
		return false, err
	}
	if seat != g.Turn || !g.Active[seat] || g.Hands[seat] == nil || g.Hands[seat].Len() == 0 {
		return false, nil
	}
	if !g.Pile.Open || g.SelfBeatContinuation || g.Pile.Locked[seat] {
		return false, nil
	}
	return true, nil
}

// validateMoveQuery checks the state gates shared by move and pass queries.
func (g *Game) validateMoveQuery(seat Seat) error {
	if g.Phase != PhasePlay {
		return fmt.Errorf("cannot query moves in phase %d: %w", g.Phase, ErrWrongPhase)
	}
	if g.AutoWinWindowOpen {
		return fmt.Errorf(
			"the declaration window is open; call StartPlay first: %w", ErrIllegalMove,
		)
	}
	if g.PilePendingResolution {
		return fmt.Errorf(
			"pile pending resolution; call ResolvePile to advance: %w", ErrIllegalMove,
		)
	}
	if !g.validSeat(seat) {
		return fmt.Errorf("invalid seat %d: %w", seat, ErrIllegalMove)
	}
	return nil
}

// enumerateCombinations returns every classifiable combination in cards in
// the stable LegalMoves order.
func enumerateCombinations(cards []cardcore.Card) []Combo {
	groups := groupCardsByRank(cards)
	moves := make([]Combo, 0, len(cards))
	for _, card := range cards {
		moves = append(moves, newCombination(Single, []cardcore.Card{card}))
	}
	for _, group := range groups {
		for _, choice := range chooseCards(group, 2) {
			moves = append(moves, newCombination(Pair, choice))
		}
	}
	for _, group := range groups {
		for _, choice := range chooseCards(group, 3) {
			moves = append(moves, newCombination(Triple, choice))
		}
	}
	for _, group := range groups {
		for _, choice := range chooseCards(group, 4) {
			moves = append(moves, newCombination(Quad, choice))
		}
	}
	moves = appendStraights(moves, groups)
	moves = appendPairSequences(moves, groups)
	slices.SortFunc(moves, compareCombinations)
	return moves
}

// groupCardsByRank groups cloned, sorted cards by their Tiến Lên rank key.
func groupCardsByRank(cards []cardcore.Card) [cardcore.NumRanks][]cardcore.Card {
	sorted := slices.Clone(cards)
	slices.SortFunc(sorted, compareCards)
	var groups [cardcore.NumRanks][]cardcore.Card
	for _, card := range sorted {
		key := rankKey[card.Rank]
		groups[key] = append(groups[key], card)
	}
	return groups
}

// chooseCards returns every size-card subset of cards in input order.
//
// For example, chooseCards([A♠, A♥, A♦], 2) returns [[A♠, A♥], [A♠, A♦],
// [A♥, A♦]].
func chooseCards(cards []cardcore.Card, size int) [][]cardcore.Card {
	if size > len(cards) {
		return nil
	}
	var choices [][]cardcore.Card
	var walk func(int, []cardcore.Card)
	// start is the index of the next card to consider, and selected is the
	// current subset of cards being built. The walk function recursively
	// builds every size-card subset of cards.
	walk = func(start int, selected []cardcore.Card) {
		if len(selected) == size {
			choices = append(choices, slices.Clone(selected))
			return
		}
		remaining := size - len(selected)
		for i := start; i <= len(cards)-remaining; i++ {
			walk(i+1, append(selected, cards[i]))
		}
	}
	walk(0, nil)
	return choices
}

// appendStraights appends every straight from maximal non-2 rank runs.
func appendStraights(moves []Combo, groups [cardcore.NumRanks][]cardcore.Card) []Combo {
	// run is a [2]int range of rank keys, inclusive, representing a maximal
	// run of consecutive ranks with at least one card each. run[0] is the
	// start rank key, and run[1] is the end rank key.
	for _, run := range consecutiveRuns(groups, 1) {
		for start := run[0]; start <= run[1]-2; start++ {
			// end is the inclusive end rank key of the straight. The minimum length
			// of a straight is 3, so end must be at least start+2. The maximum
			// length of a straight is the length of the run, so end cannot exceed
			// run[1].
			for end := start + 2; end <= run[1]; end++ {
				options := make([][][]cardcore.Card, 0, end-start+1)
				for rank := start; rank <= end; rank++ {
					rankOptions := make([][]cardcore.Card, 0, len(groups[rank]))
					for _, card := range groups[rank] {
						rankOptions = append(rankOptions, []cardcore.Card{card})
					}
					options = append(options, rankOptions)
				}
				moves = append(moves, cartesianCombinations(Straight, options)...)
			}
		}
	}
	return moves
}

// appendPairSequences appends every pair sequence from maximal non-2 rank
// runs containing at least two cards per rank.
func appendPairSequences(
	moves []Combo,
	groups [cardcore.NumRanks][]cardcore.Card,
) []Combo {
	for _, run := range consecutiveRuns(groups, 2) {
		for start := run[0]; start <= run[1]-2; start++ {
			for end := start + 2; end <= run[1]; end++ {
				options := make([][][]cardcore.Card, 0, end-start+1)
				for rank := start; rank <= end; rank++ {
					options = append(options, chooseCards(groups[rank], 2))
				}
				moves = append(moves, cartesianCombinations(PairSequence, options)...)
			}
		}
	}
	return moves
}

// consecutiveRuns returns inclusive maximal rank-key ranges before the rank
// of 2 whose groups meet the minimum multiplicity. To be used for straights
// and pair sequences, which cannot include 2s.
//
// Example: if the groups have 3 cards of rank 3, 2 cards of rank 4, and 1
// card of rank 5, then consecutiveRuns(groups, 2) returns [[3, 4]].
func consecutiveRuns(
	groups [cardcore.NumRanks][]cardcore.Card,
	minimum int,
) [][2]int {
	var runs [][2]int
	start := -1
	for rank := 0; rank < rankKey[cardcore.Two]; rank++ {
		// Iterate through the ranks in order, tracking the start of a run
		// when we encounter a rank with enough cards, and closing the run
		// when we encounter a rank without enough cards.
		if len(groups[rank]) >= minimum {
			if start < 0 {
				start = rank
			}
			continue
		}
		if start >= 0 {
			runs = append(runs, [2]int{start, rank - 1})
			start = -1
		}
	}
	// If we ended the loop in a run, close it now. This is possible if the
	// Ace rank has enough cards.
	if start >= 0 {
		runs = append(runs, [2]int{start, rankKey[cardcore.Two] - 1})
	}
	return runs
}

// cartesianCombinations returns combinations formed by selecting one card
// group from each rank's options.
//
// options is a slice of rank options, where each rank option is a slice of
// card groups, and each card group is a slice of cards. The returned
// combinations are sorted by the stable LegalMoves order.
func cartesianCombinations(kind ComboKind, options [][][]cardcore.Card) []Combo {
	var combinations []Combo
	var walk func(int, []cardcore.Card)
	walk = func(index int, cards []cardcore.Card) {
		// Recursively build combinations by selecting one card group from
		// each rank's options. Example for options = [[[3♠, 3♥], [3♠, 3♦]],
		// [[4♠, 4♥], [4♦, 4♣]]]: the walk function will first select [3♠, 3♥]
		// from the first rank, then [4♠, 4♥] from the second rank, and
		// [4♦, 4♣] from the second rank. Then it will backtrack and select
		// [3♠, 3♦] from the first rank, and repeat the process. This will
		// generate all combinations of card groups from the ranks' options.
		if index == len(options) {
			combinations = append(combinations, newCombination(kind, cards))
			return
		}
		for _, choice := range options[index] {
			walk(index+1, append(cards, choice...))
		}
	}
	walk(0, nil)
	return combinations
}

// newCombination constructs a combination from its known kind and a fresh,
// sorted copy of its cards. It panics on an empty slice: its callers only
// ever generate non-empty combinations.
func newCombination(kind ComboKind, cards []cardcore.Card) Combo {
	if len(cards) == 0 {
		panic("tienlen: cannot construct an empty combination")
	}
	sorted := slices.Clone(cards)
	slices.SortFunc(sorted, compareCards)
	return Combo{Kind: kind, Cards: sorted, Top: sorted[len(sorted)-1]}
}

// compareCombinations compares combinations in the stable LegalMoves order.
func compareCombinations(a, b Combo) int {
	if a.Kind != b.Kind {
		if a.Kind < b.Kind {
			return -1
		}
		return 1
	}
	if len(a.Cards) != len(b.Cards) {
		return len(a.Cards) - len(b.Cards)
	}
	if order := compareCards(a.Top, b.Top); order != 0 {
		return order
	}
	for i := range a.Cards {
		if order := compareCards(a.Cards[i], b.Cards[i]); order != 0 {
			return order
		}
	}
	return 0
}

// containsCard reports whether cards contains target.
func containsCard(cards []cardcore.Card, target cardcore.Card) bool {
	return slices.ContainsFunc(cards, target.Equal)
}
