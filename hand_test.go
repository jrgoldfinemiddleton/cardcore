package cardcore

import "testing"

// TestNewHand verifies that NewHand creates a hand with the correct size and
// copies the input slice.
func TestNewHand(t *testing.T) {
	cards := []Card{{Queen, Hearts}, {Ace, Spades}}
	h := NewHand(cards)

	if h.Len() != 2 {
		t.Fatalf("hand size = %d, want 2", h.Len())
	}

	// NewHand should copy, not alias the input slice.
	cards[0] = Card{Two, Clubs}
	if h.Cards[0].Suit != Hearts {
		t.Error("NewHand should copy cards, not alias")
	}
}

// TestHandAdd verifies that Add appends cards and they are retrievable via Contains.
func TestHandAdd(t *testing.T) {
	h := NewHand(nil)
	h.Add(Card{Ace, Spades})
	h.Add(Card{Queen, Hearts})

	if h.Len() != 2 {
		t.Fatalf("hand size = %d, want 2", h.Len())
	}
	if !h.Contains(Card{Ace, Spades}) {
		t.Error("hand should contain A♠")
	}
	if !h.Contains(Card{Queen, Hearts}) {
		t.Error("hand should contain Q♥")
	}
}

// TestHandRemove verifies that Remove deletes a card and returns false for absent cards.
func TestHandRemove(t *testing.T) {
	h := NewHand([]Card{
		{Two, Clubs},
		{Queen, Hearts},
		{Ace, Spades},
	})

	if !h.Remove(Card{Queen, Hearts}) {
		t.Error("Remove should return true for card in hand")
	}
	if h.Len() != 2 {
		t.Fatalf("hand size after remove = %d, want 2", h.Len())
	}
	if h.Contains(Card{Queen, Hearts}) {
		t.Error("removed card should not be in hand")
	}

	if h.Remove(Card{Queen, Hearts}) {
		t.Error("Remove should return false for card not in hand")
	}
}

// TestHandContains verifies that Contains reports presence and absence correctly.
func TestHandContains(t *testing.T) {
	h := NewHand([]Card{{Queen, Hearts}, {Ace, Spades}})

	if !h.Contains(Card{Ace, Spades}) {
		t.Error("hand should contain A♠")
	}
	if h.Contains(Card{Two, Clubs}) {
		t.Error("hand should not contain 2♣")
	}
}

// TestHandHasSuit verifies that HasSuit detects suits present in the hand.
func TestHandHasSuit(t *testing.T) {
	h := NewHand([]Card{{Queen, Hearts}, {Ace, Spades}})

	if !h.HasSuit(Spades) {
		t.Error("hand should have spades")
	}
	if h.HasSuit(Clubs) {
		t.Error("hand should not have clubs")
	}
}

// TestHandCardsOfSuit verifies that CardsOfSuit returns matching cards and an
// empty slice for absent suits.
func TestHandCardsOfSuit(t *testing.T) {
	h := NewHand([]Card{
		{Two, Clubs},
		{Queen, Hearts},
		{King, Spades},
		{Ace, Spades},
	})

	spades := h.CardsOfSuit(Spades)
	if len(spades) != 2 {
		t.Fatalf("spades count = %d, want 2", len(spades))
	}

	diamonds := h.CardsOfSuit(Diamonds)
	if len(diamonds) != 0 {
		t.Fatalf("diamonds count = %d, want 0", len(diamonds))
	}
}

// TestHandSort verifies that Sort orders cards by suit then rank ascending.
func TestHandSort(t *testing.T) {
	h := NewHand([]Card{
		{Ace, Spades},
		{Two, Clubs},
		{Queen, Hearts},
		{King, Diamonds},
		{King, Clubs},
		{Five, Diamonds},
	})

	h.Sort()

	want := []Card{
		{Two, Clubs},
		{King, Clubs},
		{Five, Diamonds},
		{King, Diamonds},
		{Queen, Hearts},
		{Ace, Spades},
	}

	for i, c := range want {
		if !h.Cards[i].Equal(c) {
			t.Errorf("sorted[%d] = %v, want %v", i, h.Cards[i], c)
		}
	}
}
