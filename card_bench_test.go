package cardcore

import "testing"

// BenchmarkNewStandardDeck measures the cost of constructing a fresh
// sorted 52-card deck.
func BenchmarkNewStandardDeck(b *testing.B) {
	for b.Loop() {
		_ = NewStandardDeck()
	}
}

// BenchmarkDeckShuffle measures the cost of in-place Fisher-Yates
// shuffle on a 52-card deck. Shuffle uses the package-level
// math/rand/v2 source (no RNG arg), so we cannot seed
// deterministically. Acceptable: we measure cost, not output.
func BenchmarkDeckShuffle(b *testing.B) {
	d := NewStandardDeck()
	for b.Loop() {
		d.Shuffle()
	}
}

// BenchmarkHandRemove measures the cost of constructing a 13-card hand
// and removing one card from the middle (O(n) append-shift). Hand.Remove
// mutates, so each iteration rebuilds the hand; the measurement therefore
// includes NewHand. Target index 6 approximates worst-case shift.
func BenchmarkHandRemove(b *testing.B) {
	// One suit's worth of cards = 13, matching a dealt Hearts hand size.
	source := make([]Card, 0, NumRanks)
	for _, r := range AllRanks() {
		source = append(source, Card{Rank: r, Suit: Clubs})
	}
	target := source[6]

	for b.Loop() {
		h := NewHand(source)
		_ = h.Remove(target)
	}
}
