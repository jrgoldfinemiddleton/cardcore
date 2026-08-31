package tienlen

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// Variant identifies a Tiến Lên ruleset (ADR-010).
type Variant uint8

// Tiến Lên variants.
const (
	// Killer is the San Diego "Killer" ruleset.
	Killer Variant = iota
)

// ComboKind identifies the type of a Tiến Lên combination.
type ComboKind uint8

// Combination kinds. (The order of kinds has no game meaning; chops are the
// only cross-kind relation.)
const (
	// Single is any one card.
	Single ComboKind = iota
	// Pair is two cards of the same rank.
	Pair
	// Triple is three cards of the same rank.
	Triple
	// Quad is four cards of the same rank (Killer rules: a "Killer";
	// Standard rules: "tứ quý").
	Quad
	// Straight is three or more cards of consecutive rank; no 2s.
	Straight
	// PairSequence is three or more pairs of consecutive rank; no 2s
	// (Killer rules: a "Bomb"; Standard rules: "đôi thông").
	PairSequence
)

// Combo is a classified Tiến Lên combination: the cards played, their kind,
// and the combination's highest card under Tiến Lên ordering. Construct with
// Classify.
type Combo struct {
	// Kind is the combination's classification.
	Kind ComboKind
	// Cards are the combination's cards, sorted ascending by Tiến Lên
	// ordering (rank first, then suit).
	Cards []cardcore.Card
	// Top is the highest card in the combination under Tiến Lên ordering
	// (rank first, then suit); comparisons between combinations of the
	// same kind and length compare Tops.
	Top cardcore.Card
}

// rankKey maps a cardcore.Rank to its Tiến Lên order: Three=0, Four=1, …,
// Ace=11, Two=12. All game-logic comparisons use these mapped keys, never
// raw cardcore.Rank values. Because non-2 ranks are assigned gap-free
// ascending keys, straights and pair sequences are detected by sorting the
// cards' mapped keys and checking that each rises by exactly 1.
var rankKey = [cardcore.NumRanks]int{
	cardcore.Two:   12,
	cardcore.Three: 0,
	cardcore.Four:  1,
	cardcore.Five:  2,
	cardcore.Six:   3,
	cardcore.Seven: 4,
	cardcore.Eight: 5,
	cardcore.Nine:  6,
	cardcore.Ten:   7,
	cardcore.Jack:  8,
	cardcore.Queen: 9,
	cardcore.King:  10,
	cardcore.Ace:   11,
}

// suitKey maps a cardcore.Suit to its Tiến Lên order: Spades=0, Clubs=1,
// Diamonds=2, Hearts=3.
var suitKey = [cardcore.NumSuits]int{
	cardcore.Spades:   0,
	cardcore.Clubs:    1,
	cardcore.Diamonds: 2,
	cardcore.Hearts:   3,
}

// Beats reports whether c ordinarily beats other: same kind, same length,
// and higher Top (rank first, then suit).
func (c Combo) Beats(other Combo) bool {
	return c.Kind == other.Kind &&
		len(c.Cards) == len(other.Cards) &&
		compareCards(c.Top, other.Top) > 0
}

// Chops reports whether c chops other under the given variant's chop
// table. Chops are the only exception to same-type, same-length matching.
// Bomb-class same-kind comparisons (e.g., a higher Quad over a lower Quad)
// are also chops — the rules list them in the chop tables and they earn
// chop payments. The Killer and eight-card Bomb pair chops in both
// directions at any height; which of the two stands is decided by play
// order in the state machine, not by this predicate.
func (c Combo) Chops(other Combo, v Variant) bool {
	switch v {
	case Killer:
		return killerChops(c, other)
	}
	return false
}

// CanBeat reports whether c could have beaten other under the given
// variant — by ordinary matching or a chop. This is the predicate the
// self-beat continuation and finisher-protection rules need.
func (c Combo) CanBeat(other Combo, v Variant) bool {
	return c.Beats(other) || c.Chops(other, v)
}

// Classify returns the Tiến Lên combination formed by cards. It returns an
// error wrapping ErrIllegalMove if cards do not form a valid combination.
// The input slice is not modified; the returned Combo holds a sorted copy.
func Classify(cards []cardcore.Card) (Combo, error) {
	sorted := slices.Clone(cards)
	slices.SortFunc(sorted, compareCards)
	for i := 1; i < len(sorted); i++ {
		if compareCards(sorted[i-1], sorted[i]) == 0 {
			return Combo{}, fmt.Errorf(
				"cannot classify combination with duplicate card %v: %w",
				sorted[i], ErrIllegalMove,
			)
		}
	}
	kind, ok := classifyShape(sorted)
	if !ok {
		return Combo{}, fmt.Errorf(
			"cannot classify %v: no legal combination: %w", sorted, ErrIllegalMove,
		)
	}
	return Combo{Kind: kind, Cards: sorted, Top: sorted[len(sorted)-1]}, nil
}

// killerChops reports whether c chops o under Killer rules. Only Quad and
// PairSequence ever chop. Every chopper chops a single 2; all but the
// six-card Bomb also chop a pair of 2s; no other single or pair is a chop
// target, and no combination chops a triple.
func killerChops(c, o Combo) bool {
	if c.Kind != Quad && c.Kind != PairSequence {
		return false
	}
	if isSingle2(o) {
		return true
	}
	if isPairOf2s(o) {
		return c.Kind == Quad || len(c.Cards) >= 8
	}
	if o.Kind == Quad {
		if c.Kind == PairSequence {
			return len(c.Cards) >= 8
		}
		return compareCards(c.Top, o.Top) > 0
	}
	if o.Kind == PairSequence {
		return killerChopsPairSequence(c, o)
	}
	return false
}

// killerChopsPairSequence reports whether c chops the pair sequence o under
// Killer rules: a Quad chops six- and eight-card Bombs at any height, and a
// Bomb chops any shorter Bomb as well as a lower Bomb of its own length.
// The caller (killerChops) guarantees that c is a Quad or PairSequence and
// that o is a PairSequence.
func killerChopsPairSequence(c, o Combo) bool {
	if c.Kind == Quad {
		return len(o.Cards) <= 8
	}
	if len(o.Cards) < len(c.Cards) {
		return true
	}
	return len(o.Cards) == len(c.Cards) && compareCards(c.Top, o.Top) > 0
}

// isSingle2 reports whether o is a single 2.
func isSingle2(o Combo) bool {
	return o.Kind == Single && o.Top.Rank == cardcore.Two
}

// isPairOf2s reports whether o is a pair of 2s.
func isPairOf2s(o Combo) bool {
	return o.Kind == Pair && o.Top.Rank == cardcore.Two
}

// compareCards compares two cards by Tiến Lên ordering (rank first, then
// suit), returning -1, 0, or +1.
func compareCards(a, b cardcore.Card) int {
	if rankKey[a.Rank] != rankKey[b.Rank] {
		return cmp.Compare(rankKey[a.Rank], rankKey[b.Rank])
	}
	return cmp.Compare(suitKey[a.Suit], suitKey[b.Suit])
}

// classifyShape determines the kind of a sorted slice of cards, reporting
// false if the cards form no legal combination. No combination exceeds
// twelve cards — there are only twelve non-2 ranks — so other lengths
// report false.
func classifyShape(cards []cardcore.Card) (ComboKind, bool) {
	switch len(cards) {
	case 1:
		return Single, true
	case 2:
		if sameRank(cards) {
			return Pair, true
		}
	case 3:
		if sameRank(cards) {
			return Triple, true
		}
		if isStraight(cards) {
			return Straight, true
		}
	case 4:
		if sameRank(cards) {
			return Quad, true
		}
		if isStraight(cards) {
			return Straight, true
		}
	case 5, 6, 7, 8, 9, 10, 11, 12:
		if isStraight(cards) {
			return Straight, true
		}
		if isPairSequence(cards) {
			return PairSequence, true
		}
	}
	return Single, false
}

// sameRank reports whether every card in the slice has the same rank.
func sameRank(cards []cardcore.Card) bool {
	for i := 1; i < len(cards); i++ {
		if rankKey[cards[i].Rank] != rankKey[cards[0].Rank] {
			return false
		}
	}
	return true
}

// isStraight reports whether the sorted cards form a straight: three or more
// cards of consecutive rank containing no 2. The rank keys must rise by
// exactly one per card; a 2 has the maximum key and so would sort last, so
// rejecting a final-card 2 excludes 2s entirely.
func isStraight(cards []cardcore.Card) bool {
	if len(cards) < 3 {
		return false
	}
	for i := 1; i < len(cards); i++ {
		if rankKey[cards[i].Rank] != rankKey[cards[i-1].Rank]+1 {
			return false
		}
	}
	return cards[len(cards)-1].Rank != cardcore.Two
}

// isPairSequence reports whether the sorted cards form a pair sequence:
// three or more pairs of consecutive rank containing no 2s. A 2 has the
// maximum rank key and so would sort into the final pair, so rejecting a
// final-pair 2 excludes 2s entirely.
func isPairSequence(cards []cardcore.Card) bool {
	if len(cards) < 6 || len(cards)%2 != 0 {
		return false
	}
	for i := 0; i+1 < len(cards); i += 2 {
		if rankKey[cards[i].Rank] != rankKey[cards[i+1].Rank] {
			return false
		}
	}
	for i := 2; i+1 < len(cards); i += 2 {
		if rankKey[cards[i].Rank] != rankKey[cards[i-2].Rank]+1 {
			return false
		}
	}
	return cards[len(cards)-1].Rank != cardcore.Two
}
