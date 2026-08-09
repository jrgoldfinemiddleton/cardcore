package cardcore

import (
	"fmt"
	"math/rand/v2"
)

// Suit represents a card suit.
type Suit uint8

// Standard suits, in canonical order (Clubs, Diamonds, Hearts, Spades).
const (
	// Clubs is the clubs suit (♣).
	Clubs Suit = iota
	// Diamonds is the diamonds suit (♦).
	Diamonds
	// Hearts is the hearts suit (♥).
	Hearts
	// Spades is the spades suit (♠).
	Spades
)

// NumSuits is the number of standard suits.
const NumSuits = 4

var suitNames = [NumSuits]string{"Clubs", "Diamonds", "Hearts", "Spades"}
var suitSymbols = [NumSuits]string{"♣", "♦", "♥", "♠"}

// Rank represents a card rank (2 through Ace).
type Rank uint8

// Standard ranks, in ascending order (Two through Ace).
const (
	// Two is the two rank (2).
	Two Rank = iota
	// Three is the three rank (3).
	Three
	// Four is the four rank (4).
	Four
	// Five is the five rank (5).
	Five
	// Six is the six rank (6).
	Six
	// Seven is the seven rank (7).
	Seven
	// Eight is the eight rank (8).
	Eight
	// Nine is the nine rank (9).
	Nine
	// Ten is the ten rank (10).
	Ten
	// Jack is the jack rank (J).
	Jack
	// Queen is the queen rank (Q).
	Queen
	// King is the king rank (K).
	King
	// Ace is the ace rank (A).
	Ace
)

// NumRanks is the number of standard ranks.
const NumRanks = 13

var rankNames = [NumRanks]string{
	"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A",
}

// Card represents a single playing card as a rank and suit.
type Card struct {
	// Rank is the card's rank.
	Rank Rank
	// Suit is the card's suit.
	Suit Suit
}

// DeckSize is the number of cards in a standard deck.
const DeckSize = NumSuits * NumRanks // 52

// Deck is an ordered collection of cards.
type Deck struct {
	// Cards holds the cards in the deck; Deal removes cards from the top
	// (the front of the slice).
	Cards []Card
}

// NewStandardDeck creates a standard 52-card deck in sorted order
// (Clubs 2-A, Diamonds 2-A, Hearts 2-A, Spades 2-A).
func NewStandardDeck() *Deck {
	cards := make([]Card, 0, DeckSize)
	for _, suit := range AllSuits() {
		for _, rank := range AllRanks() {
			cards = append(cards, Card{Rank: rank, Suit: suit})
		}
	}
	return &Deck{Cards: cards}
}

// String returns the English name of the suit.
func (s Suit) String() string {
	if s < NumSuits {
		return suitNames[s]
	}
	return fmt.Sprintf("Suit(%d)", s)
}

// Symbol returns the Unicode symbol for the suit.
func (s Suit) Symbol() string {
	if s < NumSuits {
		return suitSymbols[s]
	}
	return "?"
}

// String returns the short display string for the rank.
func (r Rank) String() string {
	if r < NumRanks {
		return rankNames[r]
	}
	return fmt.Sprintf("Rank(%d)", r)
}

// String returns a human-readable card representation like "A♠" or "10♣".
func (c Card) String() string {
	return c.Rank.String() + c.Suit.Symbol()
}

// Equal reports whether two cards are the same.
func (c Card) Equal(other Card) bool {
	return c.Suit == other.Suit && c.Rank == other.Rank
}

// Len returns the number of cards in the deck.
func (d *Deck) Len() int {
	return len(d.Cards)
}

// Shuffle randomizes the order of cards in the deck using the provided random
// number generator. The caller controls seeding for reproducible shuffles.
// Panics if rng is nil.
func (d *Deck) Shuffle(rng *rand.Rand) {
	if rng == nil {
		panic("cardcore: Shuffle requires a non-nil *rand.Rand")
	}
	rng.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Deal removes n cards from the top of the deck and returns them.
// Returns an error if there aren't enough cards.
func (d *Deck) Deal(n int) ([]Card, error) {
	if n < 0 {
		return nil, fmt.Errorf("cannot deal negative cards: %d", n)
	}
	if n > len(d.Cards) {
		return nil, fmt.Errorf("cannot deal %d cards from deck of %d", n, len(d.Cards))
	}
	dealt := make([]Card, n)
	copy(dealt, d.Cards[:n])
	d.Cards = d.Cards[n:]
	return dealt, nil
}

// Contains reports whether the deck contains the given card.
func (d *Deck) Contains(c Card) bool {
	for _, card := range d.Cards {
		if card.Equal(c) {
			return true
		}
	}
	return false
}

// AllSuits returns all four suits in standard order.
func AllSuits() [NumSuits]Suit {
	return [NumSuits]Suit{Clubs, Diamonds, Hearts, Spades}
}

// AllRanks returns all thirteen ranks in ascending order.
func AllRanks() [NumRanks]Rank {
	return [NumRanks]Rank{
		Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King, Ace,
	}
}
