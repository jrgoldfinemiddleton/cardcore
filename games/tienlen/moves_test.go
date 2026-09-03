package tienlen

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
)

// TestEnumerateSameRankCombinations verifies every suit selection for pairs,
// triples, and quads from a four-card rank.
func TestEnumerateSameRankCombinations(t *testing.T) {
	cards := []cardcore.Card{
		c(rSeven, sSpades), c(rSeven, sClubs),
		c(rSeven, sDiamonds), c(rSeven, sHearts),
	}
	moves := enumerateCombinations(cards)
	// C(4,1) singles, C(4,2) pairs, C(4,3) triples, C(4,4) quads.
	want := map[ComboKind]int{Single: 4, Pair: 6, Triple: 4, Quad: 1}
	assertKindCounts(t, moves, want)
}

// TestEnumerateStraights verifies every sub-run and suit selection from a
// maximal four-rank run.
func TestEnumerateStraights(t *testing.T) {
	cards := []cardcore.Card{
		c(rThree, sSpades), c(rThree, sClubs),
		c(rFour, sDiamonds),
		c(rFive, sSpades), c(rFive, sHearts),
		c(rSix, sClubs),
	}
	moves := enumerateCombinations(cards)
	lengths := map[int]int{}
	for _, move := range moves {
		if move.Kind == Straight {
			lengths[len(move.Cards)]++
		}
	}
	// The run is 3-4-5-6 with multiplicities 2,1,2,1. Three-card windows:
	// 3-4-5 gives 2·1·2 = 4, and 4-5-6 gives 1·2·1 = 2. The four-card
	// window 3-4-5-6 gives 2·1·2·1 = 4.
	want := map[int]int{3: 6, 4: 4}
	if !mapsEqual(lengths, want) {
		t.Errorf("got straight lengths %v, want %v", lengths, want)
	}
}

// TestEnumeratePairSequences verifies the Cartesian product of pair choices
// from triples and a quad.
func TestEnumeratePairSequences(t *testing.T) {
	cards := []cardcore.Card{
		c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds),
		c(rFour, sSpades), c(rFour, sClubs),
		c(rFour, sDiamonds), c(rFour, sHearts),
		c(rFive, sSpades), c(rFive, sClubs), c(rFive, sDiamonds),
	}
	moves := enumerateCombinations(cards)
	// One three-pair window over ranks 3-4-5 with multiplicities 3,4,3:
	// C(3,2)·C(4,2)·C(3,2) = 3·6·3 = 54 pair choices.
	want := 54
	got := countKind(moves, PairSequence)
	if got != want {
		t.Errorf("got %d pair sequences, want %d", got, want)
	}
	for _, move := range moves {
		if move.Kind == PairSequence && len(move.Cards) != 6 {
			t.Errorf("got pair sequence length %d, want 6", len(move.Cards))
		}
	}
}

// TestEnumerateExcludesTwosAndStopsAtGaps verifies that 2s never extend runs
// and missing ranks divide maximal runs.
func TestEnumerateExcludesTwosAndStopsAtGaps(t *testing.T) {
	cards := []cardcore.Card{
		c(rThree, sSpades), c(rThree, sClubs),
		c(rFour, sSpades), c(rFour, sClubs),
		c(rFive, sSpades), c(rFive, sClubs),
		c(rSeven, sSpades), c(rSeven, sClubs),
		c(rEight, sSpades), c(rEight, sClubs),
		c(rTwo, sSpades), c(rTwo, sClubs),
	}
	moves := enumerateCombinations(cards)
	// Two maximal runs: 3-4-5 (multiplicity 2 each) and 7-8. The first
	// yields one straight window with 2·2·2 = 8 suit selections and one
	// pair sequence with C(2,2)³ = 1 selection; the second is too short
	// for either. The pair of 2s contributes to neither.
	if got := countKind(moves, Straight); got != 8 {
		t.Errorf("got %d straights, want 8", got)
	}
	if got := countKind(moves, PairSequence); got != 1 {
		t.Errorf("got %d pair sequences, want 1", got)
	}
	for _, move := range moves {
		if move.Kind != Straight && move.Kind != PairSequence {
			continue
		}
		for _, card := range move.Cards {
			if card.Equal(c(rTwo, card.Suit)) {
				t.Errorf("run combination contains a 2: %v", move.Cards)
			}
		}
	}

	// A 2 can never close a run either: from Q-K-A-2 the only straight
	// is Q-K-A; there is no four-card straight ending in a 2.
	runCombos := 0
	for _, move := range enumerateCombinations([]cardcore.Card{
		c(rQueen, sSpades), c(rKing, sSpades), c(rAce, sSpades), c(rTwo, sHearts),
	}) {
		if move.Kind != Straight && move.Kind != PairSequence {
			continue
		}
		runCombos++
		for _, card := range move.Cards {
			if card.Equal(c(rTwo, card.Suit)) {
				t.Errorf("run combination contains a 2: %v", move.Cards)
			}
		}
	}
	if runCombos != 1 {
		t.Errorf("got %d run combinations, want 1 (Q-K-A only)", runCombos)
	}
}

// TestEnumerateMaximum verifies the 486-combination worst-case hand and its
// per-kind counts.
func TestEnumerateMaximum(t *testing.T) {
	moves := enumerateCombinations(maximumCombinationHand())
	// The fixture holds ranks 3-4-5-6 with multiplicities 3,3,4,3.
	// Singles: 3+3+4+3 = 13. Pairs: C(3,2)+C(3,2)+C(4,2)+C(3,2) =
	// 3+3+6+3 = 15. Triples: C(3,3)+C(3,3)+C(4,3)+C(3,3) = 1+1+4+1 = 7.
	// Quads: C(4,4) = 1. Straights over the windows 3-4-5, 4-5-6, and
	// 3-4-5-6: 3·3·4 + 3·4·3 + 3·3·4·3 = 36+36+108 = 180. Pair
	// sequences with 3,3,6,3 pair choices per rank:
	// 3·3·6 + 3·6·3 + 3·3·6·3 = 54+54+162 = 270. Total: 486.
	want := map[ComboKind]int{
		Single: 13, Pair: 15, Triple: 7, Quad: 1,
		Straight: 180, PairSequence: 270,
	}
	assertKindCounts(t, moves, want)
	if got := len(moves); got != 486 {
		t.Errorf("got %d combinations, want 486", got)
	}
}

// TestEnumerateMatchesClassifyOracle verifies that direct generation equals
// exhaustive subset classification for representative hands.
func TestEnumerateMatchesClassifyOracle(t *testing.T) {
	fixtures := []struct {
		name  string
		cards []cardcore.Card
	}{
		{"maximum", maximumCombinationHand()},
		{"twos and gaps", []cardcore.Card{
			c(rThree, sSpades), c(rThree, sClubs),
			c(rFour, sDiamonds), c(rFive, sHearts),
			c(rSeven, sSpades), c(rSeven, sClubs),
			c(rEight, sDiamonds), c(rTwo, sSpades), c(rTwo, sHearts),
		}},
		{"mixed shapes", []cardcore.Card{
			c(rNine, sSpades), c(rNine, sClubs), c(rNine, sDiamonds),
			c(rTen, sSpades), c(rJack, sClubs), c(rQueen, sDiamonds),
			c(rKing, sHearts),
		}},
	}
	for _, tt := range fixtures {
		t.Run(tt.name, func(t *testing.T) {
			got := combinationSet(enumerateCombinations(tt.cards))
			want := classifiedSubsetSet(tt.cards)
			if !mapsEqual(got, want) {
				t.Errorf("got %d generated sets, want %d classified sets", len(got), len(want))
			}
		})
	}
}

// TestEnumerateOrderIndependent verifies that hand storage order cannot
// change the ordered move list.
func TestEnumerateOrderIndependent(t *testing.T) {
	cards := maximumCombinationHand()
	want := serializeCombinations(enumerateCombinations(cards))
	shuffled := slices.Clone(cards)
	testRNG().Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	got := serializeCombinations(enumerateCombinations(shuffled))
	if got != want {
		t.Error("got different ordered combinations after shuffling the hand")
	}
}

// TestEnumerateFreshUniqueClassifiable verifies uniqueness, classification,
// and ownership independence of generated card slices.
func TestEnumerateFreshUniqueClassifiable(t *testing.T) {
	cards := maximumCombinationHand()
	moves := enumerateCombinations(cards)
	seen := map[string]bool{}
	for _, move := range moves {
		key := cardSetKey(move.Cards)
		if seen[key] {
			t.Errorf("duplicate combination %s", key)
		}
		seen[key] = true
		classified, err := Classify(move.Cards)
		if err != nil {
			t.Errorf("Classify(%v): %v", move.Cards, err)
			continue
		}
		if classified.Kind != move.Kind || !classified.Top.Equal(move.Top) {
			t.Errorf("got classified kind/top %d/%v, want %d/%v",
				classified.Kind, classified.Top, move.Kind, move.Top)
		}
	}
	original := slices.Clone(cards)
	second := slices.Clone(moves[1].Cards)
	moves[0].Cards[0] = c(rTwo, sHearts)
	if !slices.Equal(cards, original) {
		t.Errorf("got mutated hand %v, want %v", cards, original)
	}
	// Mutating one result must not leak into another: results share no
	// backing storage with the hand or with each other.
	if !slices.Equal(moves[1].Cards, second) {
		t.Errorf("got mutated second result %v, want %v", moves[1].Cards, second)
	}
}

// TestLegalMovesOpeningAndLaterLead verifies the opening-card filter and an
// unrestricted later lead.
func TestLegalMovesOpeningAndLaterLead(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{
			c(rThree, sSpades), c(rThree, sClubs), c(rFour, sDiamonds),
			c(rFive, sHearts), c(rSeven, sSpades),
		},
		[]cardcore.Card{c(rSix, sHearts)},
	)
	moves, err := g.LegalMoves(0)
	if err != nil {
		t.Fatalf("LegalMoves opening lead: %v", err)
	}
	if len(moves) == 0 {
		t.Fatal("got no opening moves, want moves containing the opening card")
	}
	for _, move := range moves {
		if !slices.ContainsFunc(move.Cards, g.OpeningCard.Equal) {
			t.Errorf("opening move omits %v: %v", g.OpeningCard, move.Cards)
		}
	}
	all := enumerateCombinations(g.Hands[0].Cards)
	if len(moves) >= len(all) {
		t.Errorf("got %d opening moves, want fewer than all %d moves", len(moves), len(all))
	}
	g.OpeningLeadRequired = false
	moves, err = g.LegalMoves(0)
	if err != nil {
		t.Fatalf("LegalMoves later lead: %v", err)
	}
	if got, want := serializeCombinations(moves), serializeCombinations(all); got != want {
		t.Error("got incomplete later-lead move list")
	}
}

// TestKillerLegalMovesFollowingAndSelfBeat verifies ordinary beat
// filtering, single-2 chops, and Killer's unrestricted self-beat
// continuation.
func TestKillerLegalMovesFollowingAndSelfBeat(t *testing.T) {
	// The fixture opens mid-pile: seat 0 has led a single 2♠ (no longer
	// in its hand) and seat 1 is to answer.
	g := newKillerGame(t,
		[]cardcore.Card{c(rAce, sHearts)},
		[]cardcore.Card{
			c(rFour, sSpades),
			c(rSix, sSpades), c(rSix, sClubs), c(rSix, sDiamonds), c(rSix, sHearts),
		},
		[]cardcore.Card{c(rFive, sSpades)},
	)
	top := mustClassifyCards(t, c(rTwo, sSpades))
	g.OpeningLeadRequired = false
	g.Pile = Pile{Open: true, Top: top, Holder: 0, Locked: make([]bool, len(g.Hands))}
	g.Turn = 1
	moves, err := g.LegalMoves(1)
	if err != nil {
		t.Fatalf("LegalMoves following: %v", err)
	}
	if len(moves) != 1 || moves[0].Kind != Quad {
		t.Errorf("got responses %v, want the quad chop", moves)
	}
	for _, move := range moves {
		if !move.CanBeat(top, Killer) {
			t.Errorf("got non-beating response %v", move.Cards)
		}
	}

	// Seat 1 now holds the pile top itself (the A♦, a card no hand
	// contains) and must self-beat: every combination is legal, even one
	// that cannot beat the top (such a play would open a new pile).
	g.Pile.Top = mustClassifyCards(t, c(rAce, sDiamonds))
	g.Pile.Holder = 1
	g.SelfBeatContinuation = true
	moves, err = g.LegalMoves(1)
	if err != nil {
		t.Fatalf("LegalMoves self-beat: %v", err)
	}
	all := enumerateCombinations(g.Hands[1].Cards)
	if got, want := serializeCombinations(moves), serializeCombinations(all); got != want {
		t.Error("got filtered self-beat moves, want every combination")
	}
	// The lowest move in the stable order (the single 4♠) cannot beat
	// the A♦ top, proving the self-beat list is unfiltered.
	if moves[0].CanBeat(g.Pile.Top, Killer) {
		t.Errorf("got first self-beat move %v beating %v, want a non-beating move",
			moves[0].Cards, g.Pile.Top.Cards)
	}
}

// TestLegalMovesEmptyEligibility verifies empty successful queries for
// non-turn, locked, finished, and declared seats, plus an unbeatable top.
func TestLegalMovesEmptyEligibility(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFour, sSpades)},
		[]cardcore.Card{
			c(rFive, sSpades), c(rAce, sSpades), c(rAce, sClubs),
			c(rAce, sDiamonds), c(rAce, sHearts),
		},
		[]cardcore.Card{c(rSix, sSpades)},
	)
	g.OpeningLeadRequired = false
	g.Pile = Pile{
		Open: true, Top: mustClassifyCards(t,
			c(rTwo, sSpades), c(rTwo, sClubs), c(rTwo, sDiamonds)),
		Holder: 0, Locked: make([]bool, len(g.Hands)),
	}
	g.Turn = 1
	assertNoLegalMoves(t, g)

	g.Pile.Top = mustClassifyCards(t, c(rFour, sSpades))
	g.Pile.Locked[1] = true
	assertNoLegalMoves(t, g)
	g.Pile.Locked[1] = false
	g.Turn = 0
	assertNoLegalMoves(t, g)
	g.Turn = 1
	g.Active[1] = false
	assertNoLegalMoves(t, g)
	g.DeclaredHands[1] = g.Hands[1]
	g.Hands[1] = cardcore.NewHand(nil)
	assertNoLegalMoves(t, g)
}

// TestLegalMovesErrors verifies every inapplicable-query error gate.
func TestLegalMovesErrors(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Game)
		seat Seat
		want error
	}{
		{"wrong phase", func(g *Game) { g.Phase = PhaseDeal }, 0, ErrWrongPhase},
		{"window open", func(g *Game) { g.AutoWinWindowOpen = true }, 0, ErrIllegalMove},
		{"pending resolution", func(g *Game) { g.PilePendingResolution = true }, 0, ErrIllegalMove},
		{"seat below range", func(*Game) {}, -1, ErrIllegalMove},
		{"seat above range", func(*Game) {}, 2, ErrIllegalMove},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newKillerGame(t,
				[]cardcore.Card{c(rThree, sSpades)},
				[]cardcore.Card{c(rFour, sSpades)},
			)
			tt.edit(g)
			moves, err := g.LegalMoves(tt.seat)
			if !errors.Is(err, tt.want) || moves != nil {
				t.Errorf("got moves %v error %v, want nil and %v", moves, err, tt.want)
			}
		})
	}
}

// TestCanPass verifies passing eligibility while leading, following,
// self-beating, locked, and querying a non-turn seat.
func TestCanPass(t *testing.T) {
	g := newKillerGame(t,
		[]cardcore.Card{c(rThree, sSpades), c(rFive, sSpades)},
		[]cardcore.Card{c(rFour, sSpades), c(rSix, sSpades)},
	)
	assertCanPass(t, g, 0, false)
	g.OpeningLeadRequired = false
	g.Pile = Pile{
		Open: true, Top: mustClassifyCards(t, c(rThree, sSpades)),
		Holder: 0, Locked: make([]bool, len(g.Hands)),
	}
	g.Turn = 1
	assertCanPass(t, g, 1, true)
	assertCanPass(t, g, 0, false)
	g.SelfBeatContinuation = true
	g.Pile.Holder = 1
	assertCanPass(t, g, 1, false)
	g.SelfBeatContinuation = false
	g.Pile.Locked[1] = true
	assertCanPass(t, g, 1, false)
}

// TestCanPassErrors verifies that passing queries share the LegalMoves error
// gates.
func TestCanPassErrors(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Game)
		seat Seat
		want error
	}{
		{"wrong phase", func(g *Game) { g.Phase = PhaseScore }, 0, ErrWrongPhase},
		{"window open", func(g *Game) { g.AutoWinWindowOpen = true }, 0, ErrIllegalMove},
		{"pending resolution", func(g *Game) { g.PilePendingResolution = true }, 0, ErrIllegalMove},
		{"seat out of range", func(*Game) {}, 3, ErrIllegalMove},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newKillerGame(t,
				[]cardcore.Card{c(rThree, sSpades)},
				[]cardcore.Card{c(rFour, sSpades)},
			)
			tt.edit(g)
			got, err := g.CanPass(tt.seat)
			if !errors.Is(err, tt.want) || got {
				t.Errorf("got %v, %v, want false, %v", got, err, tt.want)
			}
		})
	}
}

// maximumCombinationHand returns the 3,3,4,3 multiplicity fixture over
// consecutive ranks 3 through 6.
func maximumCombinationHand() []cardcore.Card {
	return []cardcore.Card{
		c(rThree, sSpades), c(rThree, sClubs), c(rThree, sDiamonds),
		c(rFour, sSpades), c(rFour, sClubs), c(rFour, sDiamonds),
		c(rFive, sSpades), c(rFive, sClubs), c(rFive, sDiamonds), c(rFive, sHearts),
		c(rSix, sSpades), c(rSix, sClubs), c(rSix, sDiamonds),
	}
}

// assertKindCounts compares the number of combinations of each kind.
func assertKindCounts(t *testing.T, moves []Combo, want map[ComboKind]int) {
	t.Helper()
	got := map[ComboKind]int{}
	for _, move := range moves {
		got[move.Kind]++
	}
	if !mapsEqual(got, want) {
		t.Errorf("got kind counts %v, want %v", got, want)
	}
}

// countKind returns the number of combinations of kind.
func countKind(moves []Combo, kind ComboKind) int {
	count := 0
	for _, move := range moves {
		if move.Kind == kind {
			count++
		}
	}
	return count
}

// classifiedSubsetSet returns every card set accepted by Classify.
func classifiedSubsetSet(cards []cardcore.Card) map[string]bool {
	set := map[string]bool{}
	// Enumerate every non-empty subset of the hand and classify it.
	// We use masking to generate subsets, which is efficient and
	// guarantees no duplicates. Each loop iteration corresponds to
	// a unique subset of the cards. Example: for a hand of 3 cards,
	// the masks 1, 2, 3, 4, 5, 6, 7 correspond to the subsets:
	// 001, 010, 011, 100, 101, 110, 111 (in binary), which map to:
	// {card0}, {card1}, {card0, card1}, {card2}, {card0, card2},
	// {card1, card2}, {card0, card1, card2}.
	for mask := 1; mask < 1<<len(cards); mask++ {
		var subset []cardcore.Card
		for i, card := range cards {
			// Include card i if the i-th bit of mask is set, which
			// guarantees a unique subset for each mask and no
			// duplicates.
			if mask&(1<<i) != 0 {
				subset = append(subset, card)
			}
		}
		combo, err := Classify(subset)
		if err == nil {
			set[cardSetKey(combo.Cards)] = true
		}
	}
	return set
}

// combinationSet returns the canonical card-set keys of combinations.
func combinationSet(moves []Combo) map[string]bool {
	set := make(map[string]bool, len(moves))
	for _, move := range moves {
		set[cardSetKey(move.Cards)] = true
	}
	return set
}

// cardSetKey returns a stable key for a card set regardless of input order.
func cardSetKey(cards []cardcore.Card) string {
	sorted := slices.Clone(cards)
	slices.SortFunc(sorted, compareCards)
	return fmt.Sprint(sorted)
}

// serializeCombinations returns a stable representation of an ordered move
// list.
func serializeCombinations(moves []Combo) string {
	var out strings.Builder
	for _, move := range moves {
		fmt.Fprintf(&out, "%d:%v:%v;", move.Kind, move.Top, move.Cards)
	}
	return out.String()
}

// mapsEqual reports whether two maps contain equal key-value pairs.
func mapsEqual[K comparable, V comparable](a, b map[K]V) bool {
	return len(a) == len(b) && mapsContain(a, b) && mapsContain(b, a)
}

// mapsContain reports whether every key-value pair in b is present in a.
func mapsContain[K comparable, V comparable](a, b map[K]V) bool {
	for key, value := range b {
		if got, ok := a[key]; !ok || got != value {
			return false
		}
	}
	return true
}

// mustClassifyCards classifies cards or fails the test.
func mustClassifyCards(t *testing.T, cards ...cardcore.Card) Combo {
	t.Helper()
	combo, err := Classify(cards)
	if err != nil {
		t.Fatalf("Classify(%v): %v", cards, err)
	}
	return combo
}

// assertNoLegalMoves verifies that seat 1's move query succeeds with no
// moves.
func assertNoLegalMoves(t *testing.T, g *Game) {
	t.Helper()
	moves, err := g.LegalMoves(1)
	if err != nil || len(moves) != 0 {
		t.Errorf("LegalMoves(1) = %v, %v; want empty, nil", moves, err)
	}
}

// assertCanPass verifies a passing query result without an error.
func assertCanPass(t *testing.T, g *Game, seat Seat, want bool) {
	t.Helper()
	got, err := g.CanPass(seat)
	if err != nil || got != want {
		t.Errorf("CanPass(%d) = %v, %v; want %v, nil", seat, got, err, want)
	}
}
