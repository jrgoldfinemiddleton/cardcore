package cardcore

import "slices"

// Hand represents a player's hand of cards.
type Hand struct {
	Cards []Card
}

// NewHand creates a hand from the given cards.
func NewHand(cards []Card) *Hand {
	c := make([]Card, len(cards))
	copy(c, cards)
	return &Hand{Cards: c}
}

// Len returns the number of cards in the hand.
func (h *Hand) Len() int {
	return len(h.Cards)
}

// Add adds a card to the hand.
func (h *Hand) Add(c Card) {
	h.Cards = append(h.Cards, c)
}

// Remove removes the first occurrence of a card from the hand.
// Returns false if the card was not found.
func (h *Hand) Remove(c Card) bool {
	for i, card := range h.Cards {
		if card.Equal(c) {
			h.Cards = append(h.Cards[:i], h.Cards[i+1:]...)
			return true
		}
	}
	return false
}

// Contains reports whether the hand contains the given card.
func (h *Hand) Contains(c Card) bool {
	for _, card := range h.Cards {
		if card.Equal(c) {
			return true
		}
	}
	return false
}

// HasSuit reports whether the hand contains any card of the given suit.
func (h *Hand) HasSuit(s Suit) bool {
	for _, c := range h.Cards {
		if c.Suit == s {
			return true
		}
	}
	return false
}

// CardsOfSuit returns all cards of the given suit in the hand.
func (h *Hand) CardsOfSuit(s Suit) []Card {
	var result []Card
	for _, c := range h.Cards {
		if c.Suit == s {
			result = append(result, c)
		}
	}
	return result
}

// Sort orders cards by suit then rank (ascending).
func (h *Hand) Sort() {
	slices.SortFunc(h.Cards, func(a, b Card) int {
		if a.Suit != b.Suit {
			return int(a.Suit) - int(b.Suit)
		}
		return int(a.Rank) - int(b.Rank)
	})
}
