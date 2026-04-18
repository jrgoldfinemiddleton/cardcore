package cardcore

import "testing"

// TestAllSuits verifies that AllSuits returns all four suits in iota order.
func TestAllSuits(t *testing.T) {
	suits := AllSuits()
	if len(suits) != NumSuits {
		t.Fatalf("expected %d suits, got %d", NumSuits, len(suits))
	}
	for i, s := range suits {
		if s != Suit(i) {
			t.Errorf("suits[%d] = %v, want %v", i, s, Suit(i))
		}
	}
}

// TestSuitString verifies the human-readable name for each suit.
func TestSuitString(t *testing.T) {
	tests := []struct {
		suit Suit
		want string
	}{
		{Clubs, "Clubs"},
		{Diamonds, "Diamonds"},
		{Hearts, "Hearts"},
		{Spades, "Spades"},
	}
	for _, tt := range tests {
		if got := tt.suit.String(); got != tt.want {
			t.Errorf("Suit(%d).String() = %q, want %q", tt.suit, got, tt.want)
		}
	}
}

// TestSuitSymbol verifies the Unicode symbol for each suit.
func TestSuitSymbol(t *testing.T) {
	tests := []struct {
		suit Suit
		want string
	}{
		{Clubs, "♣"},
		{Diamonds, "♦"},
		{Hearts, "♥"},
		{Spades, "♠"},
	}
	for _, tt := range tests {
		if got := tt.suit.Symbol(); got != tt.want {
			t.Errorf("Suit(%d).Symbol() = %q, want %q", tt.suit, got, tt.want)
		}
	}
}

// TestInvalidSuit verifies that out-of-range suit values produce fallback strings.
func TestInvalidSuit(t *testing.T) {
	s := Suit(99)
	if got := s.String(); got != "Suit(99)" {
		t.Errorf("invalid suit string = %q, want %q", got, "Suit(99)")
	}
	if got := s.Symbol(); got != "?" {
		t.Errorf("invalid suit symbol = %q, want %q", got, "?")
	}
}

// TestAllRanks verifies that AllRanks returns all thirteen ranks in iota order.
func TestAllRanks(t *testing.T) {
	ranks := AllRanks()
	if len(ranks) != NumRanks {
		t.Fatalf("expected %d ranks, got %d", NumRanks, len(ranks))
	}
	for i, r := range ranks {
		if r != Rank(i) {
			t.Errorf("ranks[%d] = %v, want %v", i, r, Rank(i))
		}
	}
}

// TestRankString verifies the short display string for a sample of ranks.
func TestRankString(t *testing.T) {
	tests := []struct {
		rank Rank
		want string
	}{
		{Two, "2"},
		{Ten, "10"},
		{Jack, "J"},
		{Queen, "Q"},
		{King, "K"},
		{Ace, "A"},
	}
	for _, tt := range tests {
		if got := tt.rank.String(); got != tt.want {
			t.Errorf("Rank(%d).String() = %q, want %q", tt.rank, got, tt.want)
		}
	}
}

// TestInvalidRank verifies that an out-of-range rank value produces a fallback string.
func TestInvalidRank(t *testing.T) {
	r := Rank(99)
	if got := r.String(); got != "Rank(99)" {
		t.Errorf("invalid rank string = %q, want %q", got, "Rank(99)")
	}
}

// TestCardString verifies the combined rank+suit display string for cards across all four suits.
func TestCardString(t *testing.T) {
	tests := []struct {
		card Card
		want string
	}{
		{Card{Two, Clubs}, "2♣"},
		{Card{Ten, Diamonds}, "10♦"},
		{Card{Queen, Hearts}, "Q♥"},
		{Card{Ace, Spades}, "A♠"},
	}
	for _, tt := range tests {
		if got := tt.card.String(); got != tt.want {
			t.Errorf("Card.String() = %q, want %q", got, tt.want)
		}
	}
}

// TestCardEqual verifies that Equal returns true for identical cards and false for different ones.
func TestCardEqual(t *testing.T) {
	a := Card{Ace, Spades}
	b := Card{Ace, Spades}
	c := Card{Ace, Hearts}

	if !a.Equal(b) {
		t.Error("identical cards should be equal")
	}
	if a.Equal(c) {
		t.Error("different cards should not be equal")
	}
}

// TestNewStandardDeck verifies that a new deck has 52 unique cards ordered by suit then rank.
func TestNewStandardDeck(t *testing.T) {
	d := NewStandardDeck()

	if d.Len() != DeckSize {
		t.Fatalf("deck size = %d, want %d", d.Len(), DeckSize)
	}

	// Check no duplicates.
	seen := make(map[Card]bool)
	for _, c := range d.Cards {
		if seen[c] {
			t.Fatalf("duplicate card: %v", c)
		}
		seen[c] = true
	}

	// First card should be 2♣, last should be A♠.
	if first := d.Cards[0]; first.Suit != Clubs || first.Rank != Two {
		t.Errorf("first card = %v, want 2♣", first)
	}
	if last := d.Cards[DeckSize-1]; last.Suit != Spades || last.Rank != Ace {
		t.Errorf("last card = %v, want A♠", last)
	}
}

// TestDeckShuffle verifies that Shuffle randomizes card order while preserving all 52 cards.
func TestDeckShuffle(t *testing.T) {
	d := NewStandardDeck()
	original := make([]Card, DeckSize)
	copy(original, d.Cards)

	d.Shuffle()

	if d.Len() != DeckSize {
		t.Fatalf("deck size after shuffle = %d, want %d", d.Len(), DeckSize)
	}

	// Shuffled deck should (almost certainly) differ from original.
	same := 0
	for i := range d.Cards {
		if d.Cards[i].Equal(original[i]) {
			same++
		}
	}
	if same == DeckSize {
		t.Error("shuffle did not change card order (extremely unlikely)")
	}

	// All original cards should still be present.
	seen := make(map[Card]bool)
	for _, c := range d.Cards {
		seen[c] = true
	}
	if len(seen) != DeckSize {
		t.Errorf("shuffle lost cards: %d unique, want %d", len(seen), DeckSize)
	}
}

// TestDeckDeal verifies dealing cards removes them from the deck, including edge cases
// for dealing zero cards and exhausting the deck.
func TestDeckDeal(t *testing.T) {
	d := NewStandardDeck()

	hand, err := d.Deal(13)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hand) != 13 {
		t.Fatalf("dealt %d cards, want 13", len(hand))
	}
	if d.Len() != 39 {
		t.Fatalf("deck has %d cards after deal, want 39", d.Len())
	}

	// Dealt cards should not still be in the deck.
	for _, c := range hand {
		if d.Contains(c) {
			t.Errorf("dealt card %v still in deck", c)
		}
	}

	// Deal 0 from non-empty deck.
	zero, err := d.Deal(0)
	if err != nil {
		t.Fatalf("unexpected error dealing 0: %v", err)
	}
	if len(zero) != 0 {
		t.Errorf("dealt %d cards, want 0", len(zero))
	}
	if d.Len() != 39 {
		t.Errorf("deck has %d cards after dealing 0, want 39", d.Len())
	}

	// Deal remaining 39 cards.
	rest, err := d.Deal(39)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 39 {
		t.Errorf("dealt %d cards, want 39", len(rest))
	}
	if d.Len() != 0 {
		t.Errorf("deck has %d cards after full deal, want 0", d.Len())
	}

	// Deal 0 from empty deck.
	zero, err = d.Deal(0)
	if err != nil {
		t.Fatalf("unexpected error dealing 0 from empty: %v", err)
	}
	if len(zero) != 0 {
		t.Errorf("dealt %d cards from empty deck, want 0", len(zero))
	}
}

// TestDeckDealErrors verifies that Deal returns errors for negative and over-size requests.
func TestDeckDealErrors(t *testing.T) {
	d := NewStandardDeck()

	if _, err := d.Deal(-1); err == nil {
		t.Error("expected error for negative deal")
	}

	if _, err := d.Deal(53); err == nil {
		t.Error("expected error for over-deal")
	}
}

// TestDeckContains verifies that Contains reports card presence before and after dealing.
func TestDeckContains(t *testing.T) {
	d := NewStandardDeck()

	if !d.Contains(Card{Ace, Spades}) {
		t.Error("new deck should contain A♠")
	}

	if _, err := d.Deal(DeckSize); err != nil {
		t.Fatalf("Deal error: %v", err)
	}
	if d.Contains(Card{Ace, Spades}) {
		t.Error("empty deck should not contain A♠")
	}
}
