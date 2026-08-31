package tienlen

import (
	"errors"
	"slices"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// Shared combination fixtures for the comparison and chop tables. Cards
// within a fixture are listed in ascending Tiến Lên order for readability;
// Classify sorts anyway.
var (
	// Singles.
	single3Spades   = []cardcore.Card{c(rThree, sSpades)}
	single7Hearts   = []cardcore.Card{c(rSeven, sHearts)}
	single8Spades   = []cardcore.Card{c(rEight, sSpades)}
	singleAceSpades = []cardcore.Card{c(rAce, sSpades)}
	single2Spades   = []cardcore.Card{c(rTwo, sSpades)}
	single2Hearts   = []cardcore.Card{c(rTwo, sHearts)}

	// Pairs.
	pair7ClubsDiamonds = []cardcore.Card{c(rSeven, sClubs), c(rSeven, sDiamonds)}
	pair7SpadesHearts  = []cardcore.Card{c(rSeven, sSpades), c(rSeven, sHearts)}
	pairAces           = []cardcore.Card{c(rAce, sDiamonds), c(rAce, sHearts)}
	pairKings          = []cardcore.Card{c(rKing, sSpades), c(rKing, sHearts)}
	pair2s             = []cardcore.Card{c(rTwo, sSpades), c(rTwo, sClubs)}

	// Triples.
	triple3s = []cardcore.Card{c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds)}
	triple5s = []cardcore.Card{c(rFive, sClubs), c(rFive, sDiamonds), c(rFive, sHearts)}
	triple7s = []cardcore.Card{c(rSeven, sSpades), c(rSeven, sClubs), c(rSeven, sHearts)}
	triple9s = []cardcore.Card{c(rNine, sSpades), c(rNine, sDiamonds), c(rNine, sHearts)}
	triple2s = []cardcore.Card{c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds)}

	// Quads.
	quad4s = []cardcore.Card{
		c(rFour, sSpades), c(rFour, sClubs), c(rFour, sDiamonds), c(rFour, sHearts),
	}
	quad9s = []cardcore.Card{
		c(rNine, sSpades), c(rNine, sClubs), c(rNine, sDiamonds), c(rNine, sHearts),
	}
	quadKings = []cardcore.Card{
		c(rKing, sSpades), c(rKing, sClubs), c(rKing, sDiamonds), c(rKing, sHearts),
	}
	quad2s = []cardcore.Card{
		c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
	}

	// Straights.
	straight345  = []cardcore.Card{c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds)}
	straight4567 = []cardcore.Card{
		c(rFour, sSpades), c(rFive, sClubs), c(rSix, sDiamonds), c(rSeven, sHearts),
	}
	straight34567 = []cardcore.Card{
		c(rThree, sSpades), c(rFour, sDiamonds), c(rFive, sClubs),
		c(rSix, sSpades), c(rSeven, sHearts),
	}
	straight89TenDiamonds = []cardcore.Card{
		c(rEight, sSpades), c(rNine, sSpades), c(rTen, sDiamonds),
	}
	straight89TenClubs = []cardcore.Card{
		c(rEight, sHearts), c(rNine, sHearts), c(rTen, sClubs),
	}
	straightQKA = []cardcore.Card{
		c(rQueen, sSpades), c(rKing, sDiamonds), c(rAce, sHearts),
	}

	// Six-card pair sequences (three consecutive pairs).
	ps6ThreeToFive = []cardcore.Card{
		c(rThree, sSpades), c(rThree, sClubs), c(rFour, sDiamonds),
		c(rFour, sHearts), c(rFive, sSpades), c(rFive, sHearts),
	}
	ps6SevenToNine = []cardcore.Card{
		c(rSeven, sSpades), c(rSeven, sHearts), c(rEight, sClubs),
		c(rEight, sDiamonds), c(rNine, sSpades), c(rNine, sClubs),
	}
	ps6NineToJack = []cardcore.Card{
		c(rNine, sDiamonds), c(rNine, sHearts), c(rTen, sSpades),
		c(rTen, sClubs), c(rJack, sSpades), c(rJack, sHearts),
	}
	ps6QueenToAce = []cardcore.Card{
		c(rQueen, sSpades), c(rQueen, sClubs), c(rKing, sDiamonds),
		c(rKing, sHearts), c(rAce, sSpades), c(rAce, sHearts),
	}

	// Eight-card pair sequences (four consecutive pairs).
	ps8ThreeToSix = []cardcore.Card{
		c(rThree, sSpades), c(rThree, sHearts), c(rFour, sSpades), c(rFour, sHearts),
		c(rFive, sSpades), c(rFive, sHearts), c(rSix, sSpades), c(rSix, sHearts),
	}
	ps8FiveToEight = []cardcore.Card{
		c(rFive, sSpades), c(rFive, sHearts), c(rSix, sSpades), c(rSix, sHearts),
		c(rSeven, sSpades), c(rSeven, sHearts), c(rEight, sSpades), c(rEight, sHearts),
	}
	ps8EightToJack = []cardcore.Card{
		c(rEight, sClubs), c(rEight, sDiamonds), c(rNine, sClubs), c(rNine, sDiamonds),
		c(rTen, sClubs), c(rTen, sDiamonds), c(rJack, sClubs), c(rJack, sDiamonds),
	}
	ps8NineToQueen = []cardcore.Card{
		c(rNine, sSpades), c(rNine, sHearts), c(rTen, sSpades), c(rTen, sHearts),
		c(rJack, sSpades), c(rJack, sHearts), c(rQueen, sSpades), c(rQueen, sHearts),
	}

	// Ten-card pair sequences (five consecutive pairs).
	ps10ThreeToSeven = []cardcore.Card{
		c(rThree, sSpades), c(rThree, sHearts), c(rFour, sSpades), c(rFour, sHearts),
		c(rFive, sSpades), c(rFive, sHearts), c(rSix, sSpades), c(rSix, sHearts),
		c(rSeven, sSpades), c(rSeven, sHearts),
	}
	ps10FiveToNine = []cardcore.Card{
		c(rFive, sClubs), c(rFive, sDiamonds), c(rSix, sClubs), c(rSix, sDiamonds),
		c(rSeven, sClubs), c(rSeven, sDiamonds), c(rEight, sClubs), c(rEight, sDiamonds),
		c(rNine, sClubs), c(rNine, sDiamonds),
	}

	// Twelve-card pair sequences (six consecutive pairs).
	ps12ThreeToEight = []cardcore.Card{
		c(rThree, sSpades), c(rThree, sClubs), c(rFour, sSpades), c(rFour, sClubs),
		c(rFive, sSpades), c(rFive, sClubs), c(rSix, sSpades), c(rSix, sClubs),
		c(rSeven, sSpades), c(rSeven, sClubs), c(rEight, sSpades), c(rEight, sClubs),
	}
	ps12SevenToQueen = []cardcore.Card{
		c(rSeven, sDiamonds), c(rSeven, sHearts), c(rEight, sDiamonds), c(rEight, sHearts),
		c(rNine, sDiamonds), c(rNine, sHearts), c(rTen, sDiamonds), c(rTen, sHearts),
		c(rJack, sDiamonds), c(rJack, sHearts), c(rQueen, sDiamonds), c(rQueen, sHearts),
	}
)

// TestClassify verifies that Classify recognizes every valid combination
// shape, returns the cards sorted ascending by Tiến Lên ordering, sets Top
// to the highest card (rank first, then suit), and never mutates its input.
func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		cards     []cardcore.Card
		wantKind  ComboKind
		wantCards []cardcore.Card
		wantTop   cardcore.Card
	}{
		{
			name:      "single three of spades (lowest card)",
			cards:     []cardcore.Card{c(rThree, sSpades)},
			wantKind:  Single,
			wantCards: []cardcore.Card{c(rThree, sSpades)},
			wantTop:   c(rThree, sSpades),
		},
		{
			name:      "single two of hearts (highest card)",
			cards:     []cardcore.Card{c(rTwo, sHearts)},
			wantKind:  Single,
			wantCards: []cardcore.Card{c(rTwo, sHearts)},
			wantTop:   c(rTwo, sHearts),
		},
		{
			name:      "pair of sevens (top by suit)",
			cards:     []cardcore.Card{c(rSeven, sDiamonds), c(rSeven, sClubs)},
			wantKind:  Pair,
			wantCards: []cardcore.Card{c(rSeven, sClubs), c(rSeven, sDiamonds)},
			wantTop:   c(rSeven, sDiamonds),
		},
		{
			name:      "pair of twos",
			cards:     []cardcore.Card{c(rTwo, sHearts), c(rTwo, sSpades)},
			wantKind:  Pair,
			wantCards: []cardcore.Card{c(rTwo, sSpades), c(rTwo, sHearts)},
			wantTop:   c(rTwo, sHearts),
		},
		{
			name:      "triple of fives",
			cards:     []cardcore.Card{c(rFive, sHearts), c(rFive, sClubs), c(rFive, sDiamonds)},
			wantKind:  Triple,
			wantCards: []cardcore.Card{c(rFive, sClubs), c(rFive, sDiamonds), c(rFive, sHearts)},
			wantTop:   c(rFive, sHearts),
		},
		{
			name:      "triple of twos (unbeatable but classifiable)",
			cards:     []cardcore.Card{c(rTwo, sDiamonds), c(rTwo, sHearts), c(rTwo, sClubs)},
			wantKind:  Triple,
			wantCards: []cardcore.Card{c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts)},
			wantTop:   c(rTwo, sHearts),
		},
		{
			name: "quad of nines",
			cards: []cardcore.Card{
				c(rNine, sHearts), c(rNine, sSpades), c(rNine, sDiamonds), c(rNine, sClubs),
			},
			wantKind: Quad,
			wantCards: []cardcore.Card{
				c(rNine, sSpades), c(rNine, sClubs), c(rNine, sDiamonds), c(rNine, sHearts),
			},
			wantTop: c(rNine, sHearts),
		},
		{
			name: "quad of twos",
			cards: []cardcore.Card{
				c(rTwo, sHearts), c(rTwo, sDiamonds), c(rTwo, sClubs), c(rTwo, sSpades),
			},
			wantKind: Quad,
			wantCards: []cardcore.Card{
				c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
			},
			wantTop: c(rTwo, sHearts),
		},
		{
			name:      "lowest straight 3-4-5",
			cards:     []cardcore.Card{c(rFour, sHearts), c(rThree, sSpades), c(rFive, sClubs)},
			wantKind:  Straight,
			wantCards: []cardcore.Card{c(rThree, sSpades), c(rFour, sHearts), c(rFive, sClubs)},
			wantTop:   c(rFive, sClubs),
		},
		{
			name:      "highest straight Q-K-A",
			cards:     []cardcore.Card{c(rAce, sHearts), c(rQueen, sSpades), c(rKing, sDiamonds)},
			wantKind:  Straight,
			wantCards: []cardcore.Card{c(rQueen, sSpades), c(rKing, sDiamonds), c(rAce, sHearts)},
			wantTop:   c(rAce, sHearts),
		},
		{
			name:      "straight 8-9-10",
			cards:     []cardcore.Card{c(rTen, sDiamonds), c(rEight, sSpades), c(rNine, sSpades)},
			wantKind:  Straight,
			wantCards: []cardcore.Card{c(rEight, sSpades), c(rNine, sSpades), c(rTen, sDiamonds)},
			wantTop:   c(rTen, sDiamonds),
		},
		{
			name: "four-card straight",
			cards: []cardcore.Card{
				c(rSeven, sHearts), c(rFour, sSpades), c(rFive, sClubs), c(rSix, sDiamonds),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rFour, sSpades), c(rFive, sClubs), c(rSix, sDiamonds), c(rSeven, sHearts),
			},
			wantTop: c(rSeven, sHearts),
		},
		{
			name: "five-card straight",
			cards: []cardcore.Card{
				c(rFive, sClubs), c(rThree, sSpades), c(rSeven, sHearts),
				c(rFour, sDiamonds), c(rSix, sSpades),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sDiamonds), c(rFive, sClubs),
				c(rSix, sSpades), c(rSeven, sHearts),
			},
			wantTop: c(rSeven, sHearts),
		},
		{
			name: "six-card straight",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds),
				c(rSix, sHearts), c(rSeven, sSpades), c(rEight, sClubs),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds),
				c(rSix, sHearts), c(rSeven, sSpades), c(rEight, sClubs),
			},
			wantTop: c(rEight, sClubs),
		},
		{
			name: "seven-card straight",
			cards: []cardcore.Card{
				c(rThree, sClubs), c(rFour, sDiamonds), c(rFive, sHearts), c(rSix, sSpades),
				c(rSeven, sClubs), c(rEight, sDiamonds), c(rNine, sHearts),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sClubs), c(rFour, sDiamonds), c(rFive, sHearts), c(rSix, sSpades),
				c(rSeven, sClubs), c(rEight, sDiamonds), c(rNine, sHearts),
			},
			wantTop: c(rNine, sHearts),
		},
		{
			name: "eight-card straight",
			cards: []cardcore.Card{
				c(rThree, sDiamonds), c(rFour, sHearts), c(rFive, sSpades), c(rSix, sClubs),
				c(rSeven, sDiamonds), c(rEight, sHearts), c(rNine, sSpades), c(rTen, sClubs),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sDiamonds), c(rFour, sHearts), c(rFive, sSpades), c(rSix, sClubs),
				c(rSeven, sDiamonds), c(rEight, sHearts), c(rNine, sSpades), c(rTen, sClubs),
			},
			wantTop: c(rTen, sClubs),
		},
		{
			name: "nine-card straight",
			cards: []cardcore.Card{
				c(rThree, sHearts), c(rFour, sSpades), c(rFive, sClubs), c(rSix, sDiamonds),
				c(rSeven, sHearts), c(rEight, sSpades), c(rNine, sClubs), c(rTen, sDiamonds),
				c(rJack, sHearts),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sHearts), c(rFour, sSpades), c(rFive, sClubs), c(rSix, sDiamonds),
				c(rSeven, sHearts), c(rEight, sSpades), c(rNine, sClubs), c(rTen, sDiamonds),
				c(rJack, sHearts),
			},
			wantTop: c(rJack, sHearts),
		},
		{
			name: "ten-card straight",
			cards: []cardcore.Card{
				c(rFour, sClubs), c(rFive, sDiamonds), c(rSix, sHearts), c(rSeven, sSpades),
				c(rEight, sClubs), c(rNine, sDiamonds), c(rTen, sHearts), c(rJack, sSpades),
				c(rQueen, sClubs), c(rKing, sDiamonds),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rFour, sClubs), c(rFive, sDiamonds), c(rSix, sHearts), c(rSeven, sSpades),
				c(rEight, sClubs), c(rNine, sDiamonds), c(rTen, sHearts), c(rJack, sSpades),
				c(rQueen, sClubs), c(rKing, sDiamonds),
			},
			wantTop: c(rKing, sDiamonds),
		},
		{
			name: "eleven-card straight",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds), c(rSix, sHearts),
				c(rSeven, sSpades), c(rEight, sClubs), c(rNine, sDiamonds), c(rTen, sHearts),
				c(rJack, sSpades), c(rQueen, sClubs), c(rKing, sHearts),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds), c(rSix, sHearts),
				c(rSeven, sSpades), c(rEight, sClubs), c(rNine, sDiamonds), c(rTen, sHearts),
				c(rJack, sSpades), c(rQueen, sClubs), c(rKing, sHearts),
			},
			wantTop: c(rKing, sHearts),
		},
		{
			name: "twelve-card dragon straight",
			cards: []cardcore.Card{
				c(rAce, sHearts), c(rThree, sSpades), c(rKing, sDiamonds), c(rFour, sClubs),
				c(rQueen, sClubs), c(rFive, sDiamonds), c(rJack, sSpades), c(rSix, sHearts),
				c(rTen, sHearts), c(rSeven, sSpades), c(rNine, sDiamonds), c(rEight, sClubs),
			},
			wantKind: Straight,
			wantCards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sClubs), c(rFive, sDiamonds), c(rSix, sHearts),
				c(rSeven, sSpades), c(rEight, sClubs), c(rNine, sDiamonds), c(rTen, sHearts),
				c(rJack, sSpades), c(rQueen, sClubs), c(rKing, sDiamonds), c(rAce, sHearts),
			},
			wantTop: c(rAce, sHearts),
		},
		{
			name: "six-card pair sequence",
			cards: []cardcore.Card{
				c(rFive, sHearts), c(rThree, sSpades), c(rFour, sDiamonds),
				c(rThree, sClubs), c(rFive, sSpades), c(rFour, sHearts),
			},
			wantKind: PairSequence,
			wantCards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFour, sDiamonds),
				c(rFour, sHearts), c(rFive, sSpades), c(rFive, sHearts),
			},
			wantTop: c(rFive, sHearts),
		},
		{
			name: "pair sequence top by suit",
			cards: []cardcore.Card{
				c(rSeven, sSpades), c(rSeven, sHearts), c(rEight, sClubs),
				c(rEight, sDiamonds), c(rNine, sSpades), c(rNine, sClubs),
			},
			wantKind: PairSequence,
			wantCards: []cardcore.Card{
				c(rSeven, sSpades), c(rSeven, sHearts), c(rEight, sClubs),
				c(rEight, sDiamonds), c(rNine, sSpades), c(rNine, sClubs),
			},
			wantTop: c(rNine, sClubs),
		},
		{
			name: "eight-card pair sequence",
			cards: []cardcore.Card{
				c(rFive, sSpades), c(rFive, sHearts), c(rSix, sSpades), c(rSix, sHearts),
				c(rSeven, sSpades), c(rSeven, sHearts), c(rEight, sSpades), c(rEight, sHearts),
			},
			wantKind: PairSequence,
			wantCards: []cardcore.Card{
				c(rFive, sSpades), c(rFive, sHearts), c(rSix, sSpades), c(rSix, sHearts),
				c(rSeven, sSpades), c(rSeven, sHearts), c(rEight, sSpades), c(rEight, sHearts),
			},
			wantTop: c(rEight, sHearts),
		},
		{
			name: "ten-card pair sequence",
			cards: []cardcore.Card{
				c(rFour, sSpades), c(rFour, sClubs), c(rFive, sDiamonds), c(rFive, sHearts),
				c(rSix, sSpades), c(rSix, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
				c(rEight, sSpades), c(rEight, sHearts),
			},
			wantKind: PairSequence,
			wantCards: []cardcore.Card{
				c(rFour, sSpades), c(rFour, sClubs), c(rFive, sDiamonds), c(rFive, sHearts),
				c(rSix, sSpades), c(rSix, sClubs), c(rSeven, sDiamonds), c(rSeven, sHearts),
				c(rEight, sSpades), c(rEight, sHearts),
			},
			wantTop: c(rEight, sHearts),
		},
		{
			name: "twelve-card pair sequence",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFour, sDiamonds), c(rFour, sHearts),
				c(rFive, sSpades), c(rFive, sClubs), c(rSix, sDiamonds), c(rSix, sHearts),
				c(rSeven, sSpades), c(rSeven, sClubs), c(rEight, sDiamonds), c(rEight, sHearts),
			},
			wantKind: PairSequence,
			wantCards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFour, sDiamonds), c(rFour, sHearts),
				c(rFive, sSpades), c(rFive, sClubs), c(rSix, sDiamonds), c(rSix, sHearts),
				c(rSeven, sSpades), c(rSeven, sClubs), c(rEight, sDiamonds), c(rEight, sHearts),
			},
			wantTop: c(rEight, sHearts),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := slices.Clone(tt.cards)
			got, err := Classify(tt.cards)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.cards, err)
			}
			if got.Kind != tt.wantKind {
				t.Errorf("got Kind %v, want %v", got.Kind, tt.wantKind)
			}
			if !slices.Equal(got.Cards, tt.wantCards) {
				t.Errorf("got Cards %v, want %v", got.Cards, tt.wantCards)
			}
			if got.Top != tt.wantTop {
				t.Errorf("got Top %v, want %v", got.Top, tt.wantTop)
			}
			if !slices.Equal(tt.cards, input) {
				t.Errorf("Classify mutated its input: got %v, want %v", tt.cards, input)
			}
		})
	}
}

// TestClassifyInvalid verifies that Classify rejects every invalid shape —
// empty and oversized inputs, duplicate cards, mismatched sets, straights
// and pair sequences containing 2s, wrap-around straights, and malformed
// pair sequences — with an error wrapping ErrIllegalMove.
func TestClassifyInvalid(t *testing.T) {
	tests := []struct {
		name  string
		cards []cardcore.Card
	}{
		{
			name:  "empty",
			cards: nil,
		},
		{
			name: "thirteen cards, one of each rank",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades), c(rSix, sSpades),
				c(rSeven, sSpades), c(rEight, sSpades), c(rNine, sSpades), c(rTen, sSpades),
				c(rJack, sSpades), c(rQueen, sSpades), c(rKing, sSpades), c(rAce, sSpades),
				c(rTwo, sSpades),
			},
		},
		{
			name: "fourteen cards",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades), c(rSix, sSpades),
				c(rSeven, sSpades), c(rEight, sSpades), c(rNine, sSpades), c(rTen, sSpades),
				c(rJack, sSpades), c(rQueen, sSpades), c(rKing, sSpades), c(rAce, sSpades),
				c(rTwo, sSpades), c(rTwo, sClubs),
			},
		},
		{
			name: "fourteen-card pair sequence (too long)",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFour, sSpades), c(rFour, sClubs),
				c(rFive, sSpades), c(rFive, sClubs), c(rSix, sSpades), c(rSix, sClubs),
				c(rSeven, sSpades), c(rSeven, sClubs), c(rEight, sSpades), c(rEight, sClubs),
				c(rNine, sSpades), c(rNine, sClubs),
			},
		},
		{
			name:  "mismatched pair",
			cards: []cardcore.Card{c(rSeven, sSpades), c(rEight, sClubs)},
		},
		{
			name:  "mismatched triple",
			cards: []cardcore.Card{c(rFive, sClubs), c(rFive, sDiamonds), c(rSix, sHearts)},
		},
		{
			name: "mismatched quad",
			cards: []cardcore.Card{
				c(rNine, sSpades), c(rNine, sClubs), c(rNine, sDiamonds), c(rTen, sHearts),
			},
		},
		{
			name:  "duplicate card",
			cards: []cardcore.Card{c(rSeven, sSpades), c(rSeven, sSpades)},
		},
		{
			name:  "duplicate card in longer combination",
			cards: []cardcore.Card{c(rFive, sSpades), c(rFive, sSpades), c(rSix, sClubs)},
		},
		{
			name: "straight containing a 2",
			cards: []cardcore.Card{
				c(rJack, sSpades), c(rQueen, sClubs), c(rKing, sDiamonds),
				c(rAce, sHearts), c(rTwo, sSpades),
			},
		},
		{
			name:  "three-card straight containing a 2 (K-A-2)",
			cards: []cardcore.Card{c(rKing, sSpades), c(rAce, sClubs), c(rTwo, sDiamonds)},
		},
		{
			name: "four-card straight containing a 2 (Q-K-A-2)",
			cards: []cardcore.Card{
				c(rQueen, sSpades), c(rKing, sSpades), c(rAce, sSpades), c(rTwo, sSpades),
			},
		},
		{
			name:  "straight containing a 2 in the middle (A-2-3)",
			cards: []cardcore.Card{c(rAce, sClubs), c(rTwo, sDiamonds), c(rThree, sSpades)},
		},
		{
			name: "pair sequence containing 2s",
			cards: []cardcore.Card{
				c(rQueen, sSpades), c(rQueen, sClubs), c(rKing, sSpades), c(rKing, sClubs),
				c(rAce, sSpades), c(rAce, sClubs), c(rTwo, sDiamonds), c(rTwo, sHearts),
			},
		},
		{
			name: "non-consecutive pair sequence",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFive, sSpades),
				c(rFive, sClubs), c(rSeven, sSpades), c(rSeven, sClubs),
			},
		},
		{
			name: "two-pair sequence (too short)",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFour, sSpades), c(rFour, sClubs),
			},
		},
		{
			name: "five-card pair-sequence attempt",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rThree, sClubs), c(rFour, sSpades),
				c(rFour, sClubs), c(rFive, sSpades),
			},
		},
		{
			name: "six cards, neither straight nor pair sequence",
			cards: []cardcore.Card{
				c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades),
				c(rSix, sSpades), c(rSeven, sSpades), c(rNine, sSpades),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Classify(tt.cards)
			if err == nil {
				t.Fatalf("Classify(%v) succeeded, want error", tt.cards)
			}
			if !errors.Is(err, ErrIllegalMove) {
				t.Errorf("got error %v, want it to wrap ErrIllegalMove", err)
			}
		})
	}
}

// TestComboBeats verifies ordinary comparison: same kind, same length, and
// higher Top (rank first, then suit). Different kinds or lengths never
// ordinarily beat, and equal Tops never beat (no self-beat).
func TestComboBeats(t *testing.T) {
	tests := []struct {
		name string
		c    []cardcore.Card
		o    []cardcore.Card
		want bool
	}{
		{
			name: "higher single beats lower single (rank dominates suit)",
			c:    single8Spades,
			o:    single7Hearts,
			want: true,
		},
		{
			name: "lower single does not beat higher single",
			c:    single7Hearts,
			o:    single8Spades,
			want: false,
		},
		{
			name: "suit tie-break: two of hearts beats two of spades",
			c:    single2Hearts,
			o:    single2Spades,
			want: true,
		},
		{
			name: "suit tie-break reversed",
			c:    single2Spades,
			o:    single2Hearts,
			want: false,
		},
		{
			name: "lowest single does not beat highest single",
			c:    single3Spades,
			o:    single2Hearts,
			want: false,
		},
		{
			name: "no self-beat on equal single",
			c:    single8Spades,
			o:    single8Spades,
			want: false,
		},
		{
			name: "rules example: 7♠-7♥ beats 7♣-7♦",
			c:    pair7SpadesHearts,
			o:    pair7ClubsDiamonds,
			want: true,
		},
		{
			name: "rules example pair reversed",
			c:    pair7ClubsDiamonds,
			o:    pair7SpadesHearts,
			want: false,
		},
		{
			name: "pair of twos beats pair of aces",
			c:    pair2s,
			o:    pairAces,
			want: true,
		},
		{
			name: "pair of aces does not beat pair of twos",
			c:    pairAces,
			o:    pair2s,
			want: false,
		},
		{
			name: "rules example: 8♠-9♠-10♦ beats 8♥-9♥-10♣",
			c:    straight89TenDiamonds,
			o:    straight89TenClubs,
			want: true,
		},
		{
			name: "rules example straight reversed",
			c:    straight89TenClubs,
			o:    straight89TenDiamonds,
			want: false,
		},
		{
			name: "no self-beat on equal-top straight",
			c:    straight345,
			o:    straight345,
			want: false,
		},
		{
			name: "different kinds never beat",
			c:    triple5s,
			o:    pair7ClubsDiamonds,
			want: false,
		},
		{
			name: "four-card straight does not beat five-card straight",
			c:    straight4567,
			o:    straight34567,
			want: false,
		},
		{
			name: "five-card straight does not beat four-card straight",
			c:    straight34567,
			o:    straight4567,
			want: false,
		},
		{
			name: "six-card pair sequence does not beat eight-card pair sequence",
			c:    ps6SevenToNine,
			o:    ps8FiveToEight,
			want: false,
		},
		{
			name: "eight-card pair sequence chops but does not beat six-card pair sequence",
			c:    ps8FiveToEight,
			o:    ps6SevenToNine,
			want: false,
		},
		{
			name: "higher triple beats lower triple",
			c:    triple9s,
			o:    triple7s,
			want: true,
		},
		{
			name: "higher quad beats lower quad",
			c:    quad9s,
			o:    quad4s,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combo, err := Classify(tt.c)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.c, err)
			}
			other, err := Classify(tt.o)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.o, err)
			}
			if got := combo.Beats(other); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestComboChopsKiller verifies the Killer chop relation for every chopper
// and target kind: only Quad and PairSequence chop; every chopper chops a
// single 2; all but the six-card Bomb chop a pair of 2s; bomb-class
// same-kind chops compare Tops; a Quad chops six- and eight-card Bombs but
// not ten- or twelve-card ones; longer Bombs chop shorter Bombs and the
// Quad at any height; and no combination chops triple 2s.
func TestComboChopsKiller(t *testing.T) {
	tests := []struct {
		name string
		c    []cardcore.Card
		o    []cardcore.Card
		want bool
	}{
		// Quad (Killer) as chopper.
		{
			name: "Quad chops single 2",
			c:    quad9s,
			o:    single2Spades,
			want: true,
		},
		{
			name: "Quad of 2s chops single 2",
			c:    quad2s,
			o:    single2Spades,
			want: true,
		},
		{
			name: "Quad does not chop single ace",
			c:    quad9s,
			o:    singleAceSpades,
			want: false,
		},
		{
			name: "Quad chops pair of 2s",
			c:    quad9s,
			o:    pair2s,
			want: true,
		},
		{
			name: "Quad does not chop pair of kings",
			c:    quad9s,
			o:    pairKings,
			want: false,
		},
		{
			name: "Quad does not chop triple",
			c:    quad9s,
			o:    triple3s,
			want: false,
		},
		{
			name: "Quad does not chop triple 2s",
			c:    quad9s,
			o:    triple2s,
			want: false,
		},
		{
			name: "Quad does not chop straight",
			c:    quad9s,
			o:    straight345,
			want: false,
		},
		{
			name: "Quad chops lower Quad",
			c:    quad9s,
			o:    quad4s,
			want: true,
		},
		{
			name: "Quad does not chop higher Quad",
			c:    quad4s,
			o:    quad9s,
			want: false,
		},
		{
			name: "Quad does not chop equal-top Quad",
			c:    quad4s,
			o:    quad4s,
			want: false,
		},
		{
			name: "Quad chops higher six-card Bomb",
			c:    quad4s,
			o:    ps6QueenToAce,
			want: true,
		},
		{
			name: "Quad chops lower six-card Bomb",
			c:    quadKings,
			o:    ps6ThreeToFive,
			want: true,
		},
		{
			name: "Quad chops higher eight-card Bomb",
			c:    quad4s,
			o:    ps8NineToQueen,
			want: true,
		},
		{
			name: "Quad chops lower eight-card Bomb",
			c:    quadKings,
			o:    ps8ThreeToSix,
			want: true,
		},
		{
			name: "Quad does not chop ten-card Bomb",
			c:    quadKings,
			o:    ps10ThreeToSeven,
			want: false,
		},
		{
			name: "Quad does not chop twelve-card Bomb",
			c:    quadKings,
			o:    ps12ThreeToEight,
			want: false,
		},

		// Six-card Bomb as chopper.
		{
			name: "six-card Bomb chops single 2",
			c:    ps6SevenToNine,
			o:    single2Spades,
			want: true,
		},
		{
			name: "six-card Bomb does not chop single ace",
			c:    ps6SevenToNine,
			o:    singleAceSpades,
			want: false,
		},
		{
			name: "six-card Bomb does not chop pair of 2s",
			c:    ps6SevenToNine,
			o:    pair2s,
			want: false,
		},
		{
			name: "six-card Bomb does not chop pair of kings",
			c:    ps6SevenToNine,
			o:    pairKings,
			want: false,
		},
		{
			name: "six-card Bomb does not chop triple",
			c:    ps6SevenToNine,
			o:    triple3s,
			want: false,
		},
		{
			name: "six-card Bomb does not chop triple 2s",
			c:    ps6SevenToNine,
			o:    triple2s,
			want: false,
		},
		{
			name: "six-card Bomb does not chop straight",
			c:    ps6SevenToNine,
			o:    straight345,
			want: false,
		},
		{
			name: "six-card Bomb does not chop Quad",
			c:    ps6QueenToAce,
			o:    quad4s,
			want: false,
		},
		{
			name: "six-card Bomb chops lower six-card Bomb",
			c:    ps6NineToJack,
			o:    ps6SevenToNine,
			want: true,
		},
		{
			name: "six-card Bomb does not chop higher six-card Bomb",
			c:    ps6SevenToNine,
			o:    ps6NineToJack,
			want: false,
		},
		{
			name: "six-card Bomb does not chop equal-top six-card Bomb",
			c:    ps6SevenToNine,
			o:    ps6SevenToNine,
			want: false,
		},
		{
			name: "six-card Bomb does not chop eight-card Bomb",
			c:    ps6QueenToAce,
			o:    ps8ThreeToSix,
			want: false,
		},
		{
			name: "six-card Bomb does not chop ten-card Bomb",
			c:    ps6QueenToAce,
			o:    ps10ThreeToSeven,
			want: false,
		},
		{
			name: "six-card Bomb does not chop twelve-card Bomb",
			c:    ps6QueenToAce,
			o:    ps12ThreeToEight,
			want: false,
		},

		// Eight-card Bomb as chopper.
		{
			name: "eight-card Bomb chops single 2",
			c:    ps8FiveToEight,
			o:    single2Spades,
			want: true,
		},
		{
			name: "eight-card Bomb does not chop single ace",
			c:    ps8FiveToEight,
			o:    singleAceSpades,
			want: false,
		},
		{
			name: "eight-card Bomb chops pair of 2s",
			c:    ps8FiveToEight,
			o:    pair2s,
			want: true,
		},
		{
			name: "eight-card Bomb does not chop pair of kings",
			c:    ps8FiveToEight,
			o:    pairKings,
			want: false,
		},
		{
			name: "eight-card Bomb does not chop triple",
			c:    ps8FiveToEight,
			o:    triple3s,
			want: false,
		},
		{
			name: "eight-card Bomb does not chop triple 2s",
			c:    ps8FiveToEight,
			o:    triple2s,
			want: false,
		},
		{
			name: "eight-card Bomb does not chop straight",
			c:    ps8FiveToEight,
			o:    straight345,
			want: false,
		},
		{
			name: "low eight-card Bomb chops high Quad",
			c:    ps8ThreeToSix,
			o:    quadKings,
			want: true,
		},
		{
			name: "high eight-card Bomb chops low Quad",
			c:    ps8NineToQueen,
			o:    quad4s,
			want: true,
		},
		{
			name: "eight-card Bomb chops higher six-card Bomb",
			c:    ps8ThreeToSix,
			o:    ps6QueenToAce,
			want: true,
		},
		{
			name: "eight-card Bomb chops lower eight-card Bomb",
			c:    ps8EightToJack,
			o:    ps8FiveToEight,
			want: true,
		},
		{
			name: "eight-card Bomb does not chop higher eight-card Bomb",
			c:    ps8FiveToEight,
			o:    ps8EightToJack,
			want: false,
		},
		{
			name: "eight-card Bomb does not chop ten-card Bomb",
			c:    ps8NineToQueen,
			o:    ps10ThreeToSeven,
			want: false,
		},
		{
			name: "eight-card Bomb does not chop twelve-card Bomb",
			c:    ps8NineToQueen,
			o:    ps12ThreeToEight,
			want: false,
		},

		// Ten-card Bomb as chopper.
		{
			name: "ten-card Bomb chops single 2",
			c:    ps10ThreeToSeven,
			o:    single2Spades,
			want: true,
		},
		{
			name: "ten-card Bomb does not chop single ace",
			c:    ps10ThreeToSeven,
			o:    singleAceSpades,
			want: false,
		},
		{
			name: "ten-card Bomb chops pair of 2s",
			c:    ps10ThreeToSeven,
			o:    pair2s,
			want: true,
		},
		{
			name: "ten-card Bomb does not chop pair of kings",
			c:    ps10ThreeToSeven,
			o:    pairKings,
			want: false,
		},
		{
			name: "ten-card Bomb does not chop triple",
			c:    ps10ThreeToSeven,
			o:    triple3s,
			want: false,
		},
		{
			name: "ten-card Bomb does not chop triple 2s",
			c:    ps10ThreeToSeven,
			o:    triple2s,
			want: false,
		},
		{
			name: "ten-card Bomb does not chop straight",
			c:    ps10ThreeToSeven,
			o:    straight345,
			want: false,
		},
		{
			name: "ten-card Bomb chops higher Quad",
			c:    ps10ThreeToSeven,
			o:    quadKings,
			want: true,
		},
		{
			name: "ten-card Bomb chops higher six-card Bomb",
			c:    ps10ThreeToSeven,
			o:    ps6QueenToAce,
			want: true,
		},
		{
			name: "ten-card Bomb chops higher eight-card Bomb",
			c:    ps10ThreeToSeven,
			o:    ps8NineToQueen,
			want: true,
		},
		{
			name: "ten-card Bomb chops lower ten-card Bomb",
			c:    ps10FiveToNine,
			o:    ps10ThreeToSeven,
			want: true,
		},
		{
			name: "ten-card Bomb does not chop higher ten-card Bomb",
			c:    ps10ThreeToSeven,
			o:    ps10FiveToNine,
			want: false,
		},
		{
			name: "ten-card Bomb does not chop twelve-card Bomb",
			c:    ps10FiveToNine,
			o:    ps12ThreeToEight,
			want: false,
		},

		// Twelve-card Bomb as chopper.
		{
			name: "twelve-card Bomb chops single 2",
			c:    ps12ThreeToEight,
			o:    single2Spades,
			want: true,
		},
		{
			name: "twelve-card Bomb does not chop single ace",
			c:    ps12ThreeToEight,
			o:    singleAceSpades,
			want: false,
		},
		{
			name: "twelve-card Bomb chops pair of 2s",
			c:    ps12ThreeToEight,
			o:    pair2s,
			want: true,
		},
		{
			name: "twelve-card Bomb does not chop pair of kings",
			c:    ps12ThreeToEight,
			o:    pairKings,
			want: false,
		},
		{
			name: "twelve-card Bomb does not chop triple",
			c:    ps12ThreeToEight,
			o:    triple3s,
			want: false,
		},
		{
			name: "twelve-card Bomb does not chop triple 2s",
			c:    ps12ThreeToEight,
			o:    triple2s,
			want: false,
		},
		{
			name: "twelve-card Bomb does not chop straight",
			c:    ps12ThreeToEight,
			o:    straight345,
			want: false,
		},
		{
			name: "twelve-card Bomb chops higher Quad",
			c:    ps12ThreeToEight,
			o:    quadKings,
			want: true,
		},
		{
			name: "twelve-card Bomb chops higher six-card Bomb",
			c:    ps12ThreeToEight,
			o:    ps6QueenToAce,
			want: true,
		},
		{
			name: "twelve-card Bomb chops higher eight-card Bomb",
			c:    ps12ThreeToEight,
			o:    ps8NineToQueen,
			want: true,
		},
		{
			name: "twelve-card Bomb chops higher ten-card Bomb",
			c:    ps12ThreeToEight,
			o:    ps10FiveToNine,
			want: true,
		},
		{
			name: "twelve-card Bomb chops lower twelve-card Bomb",
			c:    ps12SevenToQueen,
			o:    ps12ThreeToEight,
			want: true,
		},
		{
			name: "twelve-card Bomb does not chop higher twelve-card Bomb",
			c:    ps12ThreeToEight,
			o:    ps12SevenToQueen,
			want: false,
		},

		// Non-bomb-class combinations chop nothing.
		{
			name: "single does not chop single 2",
			c:    single2Hearts,
			o:    single2Spades,
			want: false,
		},
		{
			name: "single does not chop Quad",
			c:    single2Hearts,
			o:    quad4s,
			want: false,
		},
		{
			name: "pair does not chop single 2",
			c:    pairAces,
			o:    single2Spades,
			want: false,
		},
		{
			name: "pair does not chop pair of 2s",
			c:    pairKings,
			o:    pair2s,
			want: false,
		},
		{
			name: "triple does not chop single 2",
			c:    triple9s,
			o:    single2Spades,
			want: false,
		},
		{
			name: "straight does not chop single 2",
			c:    straightQKA,
			o:    single2Spades,
			want: false,
		},
		{
			name: "straight does not chop six-card Bomb",
			c:    straightQKA,
			o:    ps6ThreeToFive,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combo, err := Classify(tt.c)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.c, err)
			}
			other, err := Classify(tt.o)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.o, err)
			}
			if got := combo.Chops(other, Killer); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestComboCanBeat verifies the combined "could have beaten" predicate:
// ordinary beats and chops both count, anything else does not, and no
// bomb-class combination of any length can beat triple 2s.
func TestComboCanBeat(t *testing.T) {
	tests := []struct {
		name string
		c    []cardcore.Card
		o    []cardcore.Card
		want bool
	}{
		{
			name: "ordinary beat counts",
			c:    single8Spades,
			o:    single7Hearts,
			want: true,
		},
		{
			name: "ordinary non-beat does not count",
			c:    single7Hearts,
			o:    single8Spades,
			want: false,
		},
		{
			name: "chop counts",
			c:    quad4s,
			o:    single2Spades,
			want: true,
		},
		{
			name: "bomb-class same-kind higher counts",
			c:    quad9s,
			o:    quad4s,
			want: true,
		},
		{
			name: "neither beat nor chop does not count",
			c:    pairKings,
			o:    pair2s,
			want: false,
		},
		{
			name: "no chop-table entry: Quad cannot beat a ten-card Bomb",
			c:    quadKings,
			o:    ps10ThreeToSeven,
			want: false,
		},
		{
			name: "single 2 cannot beat triple 2s",
			c:    single2Hearts,
			o:    triple2s,
			want: false,
		},
		{
			name: "Quad cannot beat triple 2s",
			c:    quad9s,
			o:    triple2s,
			want: false,
		},
		{
			name: "six-card Bomb cannot beat triple 2s",
			c:    ps6SevenToNine,
			o:    triple2s,
			want: false,
		},
		{
			name: "eight-card Bomb cannot beat triple 2s",
			c:    ps8FiveToEight,
			o:    triple2s,
			want: false,
		},
		{
			name: "ten-card Bomb cannot beat triple 2s",
			c:    ps10ThreeToSeven,
			o:    triple2s,
			want: false,
		},
		{
			name: "twelve-card Bomb cannot beat triple 2s",
			c:    ps12ThreeToEight,
			o:    triple2s,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combo, err := Classify(tt.c)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.c, err)
			}
			other, err := Classify(tt.o)
			if err != nil {
				t.Fatalf("Classify(%v) error: %v", tt.o, err)
			}
			if got := combo.CanBeat(other, Killer); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
