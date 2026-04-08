package cardcore

import "testing"

func TestNewHand(t *testing.T) {
	cards := []Card{{Ace, Spades}, {Queen, Hearts}}
	h := NewHand(cards)

	if h.Len() != 2 {
		t.Fatalf("hand size = %d, want 2", h.Len())
	}

	// NewHand should copy, not alias the input slice.
	cards[0] = Card{Two, Clubs}
	if h.Cards[0].Suit != Spades {
		t.Error("NewHand should copy cards, not alias")
	}
}

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

func TestHandRemove(t *testing.T) {
	h := NewHand([]Card{
		{Ace, Spades},
		{Queen, Hearts},
		{Two, Clubs},
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

func TestHandContains(t *testing.T) {
	h := NewHand([]Card{{Ace, Spades}, {Queen, Hearts}})

	if !h.Contains(Card{Ace, Spades}) {
		t.Error("hand should contain A♠")
	}
	if h.Contains(Card{Two, Clubs}) {
		t.Error("hand should not contain 2♣")
	}
}

func TestHandHasSuit(t *testing.T) {
	h := NewHand([]Card{{Ace, Spades}, {Queen, Hearts}})

	if !h.HasSuit(Spades) {
		t.Error("hand should have spades")
	}
	if h.HasSuit(Clubs) {
		t.Error("hand should not have clubs")
	}
}

func TestHandCardsOfSuit(t *testing.T) {
	h := NewHand([]Card{
		{Ace, Spades},
		{King, Spades},
		{Queen, Hearts},
		{Two, Clubs},
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

	expected := []Card{
		{Two, Clubs},
		{King, Clubs},
		{Five, Diamonds},
		{King, Diamonds},
		{Queen, Hearts},
		{Ace, Spades},
	}

	for i, want := range expected {
		if !h.Cards[i].Equal(want) {
			t.Errorf("sorted[%d] = %v, want %v", i, h.Cards[i], want)
		}
	}
}
