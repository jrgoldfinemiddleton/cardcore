package ai

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// TestBuildConstraintsOpponentOrdering verifies opponents are listed in
// cyclic order starting from (seat+1) % NumPlayers. The DP relies on a
// deterministic opponent ordering so its state keys are stable across
// samples; test-driving the order locks the contract.
func TestBuildConstraintsOpponentOrdering(t *testing.T) {
	tests := []struct {
		seat hearts.Seat
		want [hearts.NumPlayers - 1]hearts.Seat
	}{
		{hearts.South, [3]hearts.Seat{hearts.West, hearts.North, hearts.East}},
		{hearts.West, [3]hearts.Seat{hearts.North, hearts.East, hearts.South}},
		{hearts.North, [3]hearts.Seat{hearts.East, hearts.South, hearts.West}},
		{hearts.East, [3]hearts.Seat{hearts.South, hearts.West, hearts.North}},
	}
	for _, tt := range tests {
		g := freshPlayGame(t)
		got := buildConstraints(g, tt.seat).opponents
		if got != tt.want {
			t.Errorf("seat %d: opponents got %v, want %v", tt.seat, got, tt.want)
		}
	}
}

// TestBuildConstraintsHandSizesReflectPlay verifies handSizes equals
// each opponent's actual hand-card count, minus any hard-pass
// assignments. The engine has already removed played cards from
// g.Hands; buildConstraints reads those lengths, then subtracts the
// number of unplayed cards seat passed to each opponent.
func TestBuildConstraintsHandSizesReflectPlay(t *testing.T) {
	g := freshPlayGame(t)
	seat := hearts.South

	got := buildConstraints(g, seat).handSizes
	// freshPlayGame finishes the pass phase. South passed 3 cards via
	// firstLegalPolicy, all of which go to the recipient under
	// PassDir; in round 0 (PassLeft) that recipient is West, so West's
	// DP capacity drops by 3 from its 13-card hand. North and East are
	// not South's recipients, so their capacities equal their hands.
	wantWest := len(g.Hands[hearts.West].Cards) - hearts.PassCount
	wantNorth := len(g.Hands[hearts.North].Cards)
	wantEast := len(g.Hands[hearts.East].Cards)
	wants := [3]int{wantWest, wantNorth, wantEast}
	for i, opp := range [3]hearts.Seat{hearts.West, hearts.North, hearts.East} {
		if got[i] != wants[i] {
			t.Errorf("opponents[%d]=%d: handSize got %d, want %d",
				i, opp, got[i], wants[i])
		}
	}
}

// TestBuildConstraintsHandSizesReflectMidTrickPlay verifies that
// handSizes reflects an opponent's mid-trick card play. The engine
// removes a played card from g.Hands[seat] at PlayCard time (before
// trick resolution); buildConstraints reads len(g.Hands[opp].Cards)
// directly. This test deals a real game, captures the baseline
// handSizes, plays one card via the engine, then asserts that the
// leader's slot in handSizes dropped by exactly 1 and every other
// slot is unchanged. It also verifies the engine actually removed
// the card (without which the buildConstraints assertion could be
// vacuously satisfied).
func TestBuildConstraintsHandSizesReflectMidTrickPlay(t *testing.T) {
	g := freshPlayGame(t)

	leader := g.Turn
	// seat = the player to move after the leader plays. This is the
	// realistic PIMC perspective (the seat about to act) and
	// guarantees seat != leader, so the leader appears as one of
	// seat's opponents.
	seat := (leader + 1) % hearts.NumPlayers

	before := buildConstraints(g, seat).handSizes

	policy := firstLegalPolicy{}
	card := policy.ChoosePlay(g, leader)
	if err := g.PlayCard(leader, card); err != nil {
		t.Fatalf("PlayCard(%d, %v) error: %v", leader, card, err)
	}

	// Engine contract: the leader's hand must drop from 13 to 12.
	// Without this check the assertion below could be vacuously
	// satisfied if the engine ever deferred card removal until
	// trick resolution.
	if got := len(g.Hands[leader].Cards); got != 12 {
		t.Fatalf("leader hand size after PlayCard: got %d, want 12 "+
			"(engine did not remove on play)", got)
	}

	after := buildConstraints(g, seat).handSizes

	for i := range after {
		opp := (seat + hearts.Seat(i) + 1) % hearts.NumPlayers
		want := before[i]
		if opp == leader {
			want--
		}
		if after[i] != want {
			t.Errorf("opponents[%d]=%d: handSize got %d, want %d (delta from %d)",
				i, opp, after[i], want, before[i])
		}
	}
}

// TestBuildConstraintsTrickHistoryVoids verifies that voids revealed in
// completed tricks (an opponent failed to follow suit) are captured.
// This is delegated to analyze() but the test pins the wiring.
func TestBuildConstraintsTrickHistoryVoids(t *testing.T) {
	// Trick 2: East (winner of Trick 1 per validFirstTrick) leads 6♣;
	// turn order East -> South -> West -> North. South and North
	// follow with clubs; West discards 2♥ off-suit, revealing the
	// club void. North wins trick 2 with 8♣.
	clubLead := hearts.Trick{
		Leader: hearts.East,
		Count:  hearts.NumPlayers,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rSix, sClubs),
			hearts.South: c(rSeven, sClubs),
			hearts.West:  c(rTwo, sHearts),
			hearts.North: c(rEight, sClubs),
		},
	}
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassHold
	g.TrickNum = 2
	g.TrickHistory = []hearts.Trick{validFirstTrick(), clubLead}
	g.Turn = hearts.North // winner of Trick 2 leads next
	g.HeartsBroken = true
	// Realistic 44-card distribution (52 - 8 played, 11 per hand).
	// West holds zero clubs, consistent with the void revealed in
	// Trick 2. Each player's current hand excludes the cards they
	// already played in Tricks 1 and 2.
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rNine, sClubs),
		c(rTwo, sDiamonds), c(rThree, sDiamonds), c(rFour, sDiamonds),
		c(rThree, sHearts), c(rFour, sHearts), c(rFive, sHearts),
		c(rTwo, sSpades), c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rFive, sDiamonds), c(rSix, sDiamonds), c(rSeven, sDiamonds), c(rEight, sDiamonds),
		c(rSix, sHearts), c(rSeven, sHearts), c(rEight, sHearts), c(rNine, sHearts),
		c(rSix, sSpades), c(rSeven, sSpades), c(rEight, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rTen, sClubs), c(rJack, sClubs), c(rAce, sClubs),
		c(rNine, sDiamonds), c(rTen, sDiamonds), c(rJack, sDiamonds),
		c(rTen, sHearts), c(rJack, sHearts),
		c(rNine, sSpades), c(rTen, sSpades), c(rJack, sSpades),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sClubs), c(rKing, sClubs),
		c(rQueen, sDiamonds), c(rKing, sDiamonds), c(rAce, sDiamonds),
		c(rQueen, sHearts), c(rKing, sHearts), c(rAce, sHearts),
		queenOfSpades, kingOfSpades, aceOfSpades,
	})

	got := buildConstraints(g, hearts.South)
	westIdx := 0
	if !got.voids[westIdx][sClubs] {
		t.Errorf("voids[West][Clubs] got false, want true")
	}
}

// TestBuildConstraintsCurrentTrickVoids verifies that off-suit plays in
// the in-progress trick reveal voids. analyze() does not scan the
// current trick, so this is buildConstraints's own responsibility.
func TestBuildConstraintsCurrentTrickVoids(t *testing.T) {
	g := buildVoidFixture(t)

	got := buildConstraints(g, hearts.South)
	westIdx := 0
	if !got.voids[westIdx][sClubs] {
		t.Errorf("voids[West][Clubs] got false, want true after current-trick discard")
	}
}

// TestBuildConstraintsCurrentTrickPlayedCardsExcluded verifies that
// cards played so far in the in-progress trick are absent from
// unseenBySuit. Otherwise the DP would try to deal already-played cards
// to opponents.
func TestBuildConstraintsCurrentTrickPlayedCardsExcluded(t *testing.T) {
	g := buildVoidFixture(t)

	got := buildConstraints(g, hearts.South)
	if slices.Contains(got.unseenBySuit[sClubs], c(rSix, sClubs)) {
		t.Errorf("unseenBySuit[Clubs] contains current-trick card 6♣")
	}
	if slices.Contains(got.unseenBySuit[sHearts], c(rTwo, sHearts)) {
		t.Errorf("unseenBySuit[Hearts] contains current-trick card 2♥")
	}
}

// TestBuildConstraintsHardPassConstraint verifies that cards seat passed
// (and not yet played) are removed from the unseen pool, added to the
// recipient's hardAssigned slice, and the recipient's handSize is
// decremented accordingly.
func TestBuildConstraintsHardPassConstraint(t *testing.T) {
	g := buildPassFixture(t, hearts.PassLeft)
	g.PassHistory[hearts.South] = [hearts.PassCount]cardcore.Card{
		queenOfSpades, kingOfSpades, aceOfSpades,
	}

	got := buildConstraints(g, hearts.South)
	westIdx := 0 // opponents[0] from South's perspective is West

	wantHard := []cardcore.Card{
		queenOfSpades, kingOfSpades, aceOfSpades,
	}
	if !cardSlicesEqualAsSets(got.hardAssigned[westIdx], wantHard) {
		t.Errorf("hardAssigned[West] got %v, want set %v", got.hardAssigned[westIdx], wantHard)
	}

	// West's hand has 13 cards; 3 are hard-assigned, so DP capacity is 10.
	if got.handSizes[westIdx] != 10 {
		t.Errorf("handSizes[West] got %d, want 10 (13 in hand - 3 hard)",
			got.handSizes[westIdx])
	}

	for _, hardCard := range wantHard {
		if slices.Contains(got.unseenBySuit[hardCard.Suit], hardCard) {
			t.Errorf("unseenBySuit[%v] still contains hard-assigned %v", hardCard.Suit, hardCard)
		}
	}
}

// TestBuildConstraintsHoldRoundSkipsHardPass verifies that PassHold
// rounds (no passing) produce empty hardAssigned and unmodified
// handSizes.
func TestBuildConstraintsHoldRoundSkipsHardPass(t *testing.T) {
	g := buildPassFixture(t, hearts.PassHold)

	got := buildConstraints(g, hearts.South)
	for i := range got.hardAssigned {
		if len(got.hardAssigned[i]) != 0 {
			t.Errorf("hardAssigned[%d] got %v, want empty in hold round",
				i, got.hardAssigned[i])
		}
	}
	for i, opp := range [3]hearts.Seat{hearts.West, hearts.North, hearts.East} {
		if got.handSizes[i] != len(g.Hands[opp].Cards) {
			t.Errorf("handSizes[%d] got %d, want %d",
				i, got.handSizes[i], len(g.Hands[opp].Cards))
		}
		if got.handSizes[i] != 13 {
			t.Errorf("handSizes[%d] got %d, want 13 (full hand, no hard-pass decrement)",
				i, got.handSizes[i])
		}
	}
}

// TestBuildConstraintsPlayedPassCardNotHardAssigned verifies that a
// pass card already visible in trick history is NOT hard-assigned (it
// has left the recipient's hand by play), while unplayed pass cards
// remain hard-assigned.
func TestBuildConstraintsPlayedPassCardNotHardAssigned(t *testing.T) {
	// South passed 3♣, K♠, A♠ to West (PassLeft).
	// Trick 1: validFirstTrick — West played 3♣ (consumed pass card).
	// K♠ and A♠ remain unplayed in West's hand → still hard-assigned.
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassLeft
	g.TrickNum = 1
	// Trick 1: validated 2♣ opener.
	g.TrickHistory = []hearts.Trick{validFirstTrick()}
	g.Turn = hearts.East // winner of Trick 1
	g.PassHistory[hearts.South] = [hearts.PassCount]cardcore.Card{
		c(rThree, sClubs), kingOfSpades, aceOfSpades,
	}
	// Realistic 48-card distribution (52 - 4 played in Trick 1).
	// Each player's hand excludes the card they played in Trick 1.
	// West holds K♠ and A♠ (unplayed pass cards from South).
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rSix, sClubs), c(rSeven, sClubs), c(rEight, sClubs), c(rNine, sClubs),
		c(rTen, sClubs), c(rJack, sClubs), c(rQueen, sClubs), c(rKing, sClubs),
		c(rAce, sClubs),
		c(rTwo, sDiamonds), c(rThree, sDiamonds), c(rFour, sDiamonds),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rFive, sDiamonds), c(rSix, sDiamonds), c(rSeven, sDiamonds), c(rEight, sDiamonds),
		c(rNine, sDiamonds), c(rTen, sDiamonds), c(rJack, sDiamonds), c(rQueen, sDiamonds),
		c(rKing, sDiamonds),
		c(rTwo, sHearts),
		kingOfSpades, aceOfSpades,
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rAce, sDiamonds),
		c(rThree, sHearts), c(rFour, sHearts), c(rFive, sHearts), c(rSix, sHearts),
		c(rSeven, sHearts), c(rEight, sHearts), c(rNine, sHearts), c(rTen, sHearts),
		c(rJack, sHearts), c(rQueen, sHearts), c(rKing, sHearts),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rAce, sHearts),
		c(rTwo, sSpades), c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades),
		c(rSix, sSpades), c(rSeven, sSpades), c(rEight, sSpades), c(rNine, sSpades),
		c(rTen, sSpades), c(rJack, sSpades), queenOfSpades,
	})

	got := buildConstraints(g, hearts.South)
	westIdx := 0

	// 3♣ was played in Trick 1 — must NOT be hard-assigned.
	for _, hardCard := range got.hardAssigned[westIdx] {
		if hardCard == c(rThree, sClubs) {
			t.Errorf("hardAssigned[West] contains played pass card 3♣")
		}
	}

	// K♠ and A♠ were not played — must remain hard-assigned.
	wantHard := []cardcore.Card{kingOfSpades, aceOfSpades}
	if !cardSlicesEqualAsSets(got.hardAssigned[westIdx], wantHard) {
		t.Errorf("hardAssigned[West] got %v, want set %v",
			got.hardAssigned[westIdx], wantHard)
	}

	// West has 12 cards; 2 are hard-assigned, so DP capacity is 10.
	if got.handSizes[westIdx] != 10 {
		t.Errorf("handSizes[West] got %d, want 10 (12 in hand - 2 hard)",
			got.handSizes[westIdx])
	}
}

// TestNewSampleDealDPSmallVoidPosition verifies the DP root value matches
// a hand-computed 42-deal count for a small fixture with two void
// constraints.
func TestNewSampleDealDPSmallVoidPosition(t *testing.T) {
	sc := smallVoidConstraints()
	dp := newSampleDealDP(&sc)
	rootKey := makeStateKey(0, 2, 2, 2)
	got := dp.table[rootKey]
	if got != 42 {
		t.Errorf("root deal count got %d, want 42", got)
	}
}

// TestNewSampleDealDPNoVoids verifies the DP on an unconstrained
// position where all splits are legal. With 2 cards per suit across 4
// suits and capacities (3, 3, 2), the answer is 8!/(3!*3!*2!) = 560
// when no voids apply (each suit's multinomial times the next suit's
// subtree, summed).
func TestNewSampleDealDPNoVoids(t *testing.T) {
	// 8 cards, 2 per suit, capacities 3+3+2=8. No voids.
	sc := sampleConstraints{
		seat:      hearts.South,
		opponents: [3]hearts.Seat{hearts.West, hearts.North, hearts.East},
		handSizes: [3]int{3, 3, 2},
	}
	// 2 cards in each suit (using arbitrary cards).
	sc.unseenBySuit[sSpades] = []cardcore.Card{
		kingOfSpades, aceOfSpades,
	}
	sc.unseenBySuit[sHearts] = []cardcore.Card{
		c(rTwo, sHearts), c(rThree, sHearts),
	}
	sc.unseenBySuit[sDiamonds] = []cardcore.Card{
		c(rKing, sDiamonds), c(rAce, sDiamonds),
	}
	sc.unseenBySuit[sClubs] = []cardcore.Card{
		twoOfClubs, c(rThree, sClubs),
	}
	dp := newSampleDealDP(&sc)
	rootKey := makeStateKey(0, 3, 3, 2)
	got := dp.table[rootKey]
	// 8! / (3! * 3! * 2!) = 560
	if got != 560 {
		t.Errorf("root deal count got %d, want 560", got)
	}
}

// TestNewSampleDealDPAllVoidInfeasible verifies zero deals when voids
// make the position impossible. All opponents void in spades but spades
// has a card that must go somewhere.
func TestNewSampleDealDPAllVoidInfeasible(t *testing.T) {
	sc := sampleConstraints{
		seat:      hearts.South,
		opponents: [3]hearts.Seat{hearts.West, hearts.North, hearts.East},
		handSizes: [3]int{1, 1, 1},
		voids: [3][cardcore.NumSuits]bool{
			{sSpades: true},
			{sSpades: true},
			{sSpades: true},
		},
	}
	sc.unseenBySuit[sSpades] = []cardcore.Card{aceOfSpades}
	sc.unseenBySuit[sHearts] = []cardcore.Card{c(rTwo, sHearts)}
	sc.unseenBySuit[sDiamonds] = []cardcore.Card{c(rAce, sDiamonds)}
	dp := newSampleDealDP(&sc)
	rootKey := makeStateKey(0, 1, 1, 1)
	got := dp.table[rootKey]
	if got != 0 {
		t.Errorf("root deal count got %d, want 0", got)
	}
}

// TestMultinomial verifies the multinomial function against known values.
func TestMultinomial(t *testing.T) {
	tests := []struct {
		k, a, b int
		want    int64
	}{
		{0, 0, 0, 1},
		{1, 1, 0, 1},
		{1, 0, 1, 1},
		{2, 1, 1, 2},
		{3, 1, 1, 6},
		{6, 2, 2, 90},
		{6, 3, 2, 60},
	}
	for _, tt := range tests {
		got := multinomial(tt.k, tt.a, tt.b)
		if got != tt.want {
			t.Errorf("multinomial(%d,%d,%d) got %d, want %d",
				tt.k, tt.a, tt.b, got, tt.want)
		}
	}
}

// TestBinomial verifies the binomial function against known values.
func TestBinomial(t *testing.T) {
	tests := []struct {
		n, k int
		want int64
	}{
		{0, 0, 1},
		{1, 0, 1},
		{1, 1, 1},
		{5, 2, 10},
		{13, 4, 715},
		{13, 0, 1},
	}
	for _, tt := range tests {
		got := binomial(tt.n, tt.k)
		if got != tt.want {
			t.Errorf("binomial(%d,%d) got %d, want %d", tt.n, tt.k, got, tt.want)
		}
	}
}

// TestSampleDealStructure verifies that sample produces a deal
// satisfying all structural invariants: seat's hand is unchanged, every
// opponent gets the right number of cards, and all unseen + hard-assigned
// cards appear exactly once across opponent hands.
func TestSampleDealStructure(t *testing.T) {
	for _, seat := range []hearts.Seat{hearts.South, hearts.East} {
		g := freshPlayGame(t)
		sc := buildConstraints(g, seat)
		dp := newSampleDealDP(&sc)
		rng := rand.New(rand.NewPCG(1, 2))

		deal := dp.sample(g, seat, rng)

		// Seat's hand must match exactly.
		if !cardSlicesEqualAsSets(deal[seat].Cards, g.Hands[seat].Cards) {
			t.Errorf("seat %d: deal[seat] got %v, want %v",
				seat, deal[seat].Cards, g.Hands[seat].Cards)
		}

		// Each opponent must have the right total hand size (DP
		// capacity + hard-assigned).
		for i, opp := range sc.opponents {
			wantSize := len(g.Hands[opp].Cards)
			gotSize := len(deal[opp].Cards)
			if gotSize != wantSize {
				t.Errorf("seat %d: deal[%d] size got %d, want %d",
					seat, opp, gotSize, wantSize)
			}

			// No card in a void suit.
			for _, card := range deal[opp].Cards {
				if sc.voids[i][card.Suit] {
					t.Errorf("seat %d: deal[%d] contains %v but void in %v",
						seat, opp, card, card.Suit)
				}
			}
		}

		// Every card across all four hands must appear exactly once
		// (52-card conservation minus played cards).
		seen := make(map[cardcore.Card]int)
		for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
			for _, card := range deal[s].Cards {
				seen[card]++
			}
		}
		for card, count := range seen {
			if count != 1 {
				t.Errorf("seat %d: card %v appears %d times in deal",
					seat, card, count)
			}
		}
	}
}

// TestSampleDealSmallVoidUniformity verifies approximate uniformity on
// the smallVoidConstraints fixture. With 42 feasible deals and 10000
// samples, each deal should appear roughly 10000/42 ≈ 238 times. The
// test checks that at least 30 distinct deals appear (out of 42) and
// that no single deal appears more than 2× the expected frequency.
func TestSampleDealSmallVoidUniformity(t *testing.T) {
	sc := smallVoidConstraints()
	dp := newSampleDealDP(&sc)

	// Build a minimal game state for the fixture. sample
	// only reads g.Hands[seat] to copy into the deal.
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassHold
	seat := hearts.South
	g.Hands[seat] = cardcore.NewHand([]cardcore.Card{
		c(rFour, sClubs), c(rFive, sClubs),
	})

	type dealKey [hearts.NumPlayers - 1]string
	makeDealKey := func(deal sampledDeal) dealKey {
		var dk dealKey
		for i, opp := range sc.opponents {
			hand := deal[opp]
			hand.Sort()
			dk[i] = fmt.Sprint(hand.Cards)
		}
		return dk
	}

	const numSamples = 10000
	counts := make(map[dealKey]int)
	rng := rand.New(rand.NewPCG(42, 99))
	for range numSamples {
		deal := dp.sample(g, seat, rng)
		counts[makeDealKey(deal)]++
	}

	// At least 30 of the 42 feasible deals should appear.
	if len(counts) < 30 {
		t.Errorf("distinct deals got %d, want >= 30 (out of 42 feasible)",
			len(counts))
	}

	// No deal should appear more than 2× the expected frequency.
	maxExpected := 2 * numSamples / 42
	for dk, n := range counts {
		if n > maxExpected {
			t.Errorf("deal %v appeared %d times, max expected %d",
				dk, n, maxExpected)
		}
	}
}

// TestSampleDealDeterministic verifies that the same RNG seed produces
// the same deal, confirming the reproducibility contract.
func TestSampleDealDeterministic(t *testing.T) {
	g := freshPlayGame(t)
	seat := hearts.South
	sc := buildConstraints(g, seat)
	dp := newSampleDealDP(&sc)

	deal1 := dp.sample(g, seat, rand.New(rand.NewPCG(7, 13)))
	deal2 := dp.sample(g, seat, rand.New(rand.NewPCG(7, 13)))

	for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
		if !cardSlicesEqualAsSets(deal1[s].Cards, deal2[s].Cards) {
			t.Errorf("seat %d: deal1 != deal2 with same seed", s)
		}
	}
}

// TestSampleDealDifferentSeeds verifies that different RNG seeds produce
// at least one different deal (stochastic; uses tries=N pattern).
func TestSampleDealDifferentSeeds(t *testing.T) {
	g := freshPlayGame(t)
	seat := hearts.South
	sc := buildConstraints(g, seat)
	dp := newSampleDealDP(&sc)

	baseline := dp.sample(g, seat, rand.New(rand.NewPCG(1, 2)))

	const tries = 20
	for seed := uint64(3); seed < tries+3; seed++ {
		other := dp.sample(g, seat, rand.New(rand.NewPCG(seed, seed+1)))
		different := false
		for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
			if !cardSlicesEqualAsSets(baseline[s].Cards, other[s].Cards) {
				different = true
				break
			}
		}
		if different {
			return
		}
	}
	t.Errorf("all %d seeds produced the same deal", tries)
}

// buildVoidFixture constructs a shared mid-trick state used by
// TestBuildConstraintsCurrentTrickVoids and
// TestBuildConstraintsCurrentTrickPlayedCardsExcluded: trick 2 in
// progress, East (winner of Trick 1) leads 6♣, South follows 7♣,
// West discards 2♥ (revealing a club void), North has not yet played.
// PassDir = PassHold sidesteps the hard-pass machinery.
//
// Hands are realistic: every unplayed card sits in exactly one hand
// (52 = 7 played + 45 in hands; West holds zero clubs — had exactly
// one club (3♣) which was played in Trick 1).
func buildVoidFixture(t *testing.T) *hearts.Game {
	t.Helper()
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = hearts.PassHold
	g.TrickNum = 1
	// Trick 1: validated 2♣ opener.
	g.TrickHistory = []hearts.Trick{validFirstTrick()}
	// Trick 2: East (winner of Trick 1) leads 6♣; South and West have
	// played, North has not.
	g.Trick = hearts.Trick{
		Leader: hearts.East,
		Count:  3,
		Cards: [hearts.NumPlayers]cardcore.Card{
			hearts.East:  c(rSix, sClubs),
			hearts.South: c(rSeven, sClubs),
			hearts.West:  c(rTwo, sHearts),
		},
	}
	g.Turn = hearts.North
	g.HeartsBroken = true
	// Realistic 45-card distribution (52 - 7 played, see above).
	// West holds zero clubs, consistent with the void revealed in
	// Trick 2. Each player's current hand excludes the cards they
	// already played in Tricks 1 and 2.
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rNine, sClubs),
		c(rTwo, sDiamonds), c(rThree, sDiamonds), c(rFour, sDiamonds),
		c(rThree, sHearts), c(rFour, sHearts), c(rFive, sHearts),
		c(rTwo, sSpades), c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rFive, sDiamonds), c(rSix, sDiamonds), c(rSeven, sDiamonds), c(rEight, sDiamonds),
		c(rSix, sHearts), c(rSeven, sHearts), c(rEight, sHearts), c(rNine, sHearts),
		c(rSix, sSpades), c(rSeven, sSpades), c(rEight, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rEight, sClubs), c(rTen, sClubs), c(rJack, sClubs), c(rAce, sClubs),
		c(rNine, sDiamonds), c(rTen, sDiamonds), c(rJack, sDiamonds),
		c(rTen, sHearts), c(rJack, sHearts),
		c(rNine, sSpades), c(rTen, sSpades), c(rJack, sSpades),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sClubs), c(rKing, sClubs),
		c(rQueen, sDiamonds), c(rKing, sDiamonds), c(rAce, sDiamonds),
		c(rQueen, sHearts), c(rKing, sHearts), c(rAce, sHearts),
		queenOfSpades, kingOfSpades, aceOfSpades,
	})
	return g
}

// buildPassFixture constructs a pre-trick state (TrickNum = 0, no
// TrickHistory) with the given pass direction and realistic 52-card
// hands (13 per player). West's hand contains Q♠ K♠ A♠ so that
// TestBuildConstraintsHardPassConstraint can wire PassHistory
// accordingly.
func buildPassFixture(t *testing.T, dir hearts.PassDirection) *hearts.Game {
	t.Helper()
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.PassDir = dir
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		twoOfClubs, c(rThree, sClubs), c(rFour, sClubs), c(rFive, sClubs),
		c(rSix, sClubs), c(rSeven, sClubs), c(rEight, sClubs), c(rNine, sClubs),
		c(rTen, sClubs),
		c(rTwo, sDiamonds), c(rThree, sDiamonds), c(rFour, sDiamonds), c(rFive, sDiamonds),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rSix, sDiamonds), c(rSeven, sDiamonds), c(rEight, sDiamonds), c(rNine, sDiamonds),
		c(rTen, sDiamonds), c(rJack, sDiamonds), c(rQueen, sDiamonds), c(rKing, sDiamonds),
		c(rAce, sDiamonds),
		c(rTwo, sHearts),
		queenOfSpades, kingOfSpades, aceOfSpades,
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rJack, sClubs), c(rQueen, sClubs), c(rKing, sClubs), c(rAce, sClubs),
		c(rThree, sHearts), c(rFour, sHearts), c(rFive, sHearts), c(rSix, sHearts),
		c(rSeven, sHearts), c(rEight, sHearts), c(rNine, sHearts), c(rTen, sHearts),
		c(rJack, sHearts),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rQueen, sHearts), c(rKing, sHearts), c(rAce, sHearts),
		c(rTwo, sSpades), c(rThree, sSpades), c(rFour, sSpades), c(rFive, sSpades),
		c(rSix, sSpades), c(rSeven, sSpades), c(rEight, sSpades), c(rNine, sSpades),
		c(rTen, sSpades), c(rJack, sSpades),
	})
	return g
}

// cardSlicesEqualAsSets returns true if a and b contain the same cards
// (ignoring order and assuming no duplicates within either slice).
func cardSlicesEqualAsSets(a, b []cardcore.Card) bool {
	if len(a) != len(b) {
		return false
	}
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}

// smallVoidConstraints returns a small fixture with known combinatorics:
// 6 unseen cards, capacities (2,2,2), West void in spades, East void in
// diamonds. The fixture has exactly 42 feasible deals.
func smallVoidConstraints() sampleConstraints {
	sc := sampleConstraints{
		seat:      hearts.South,
		opponents: [3]hearts.Seat{hearts.West, hearts.North, hearts.East},
		handSizes: [3]int{2, 2, 2},
		voids: [3][cardcore.NumSuits]bool{
			{sSpades: true},   // West void spades
			{},                // North no voids
			{sDiamonds: true}, // East void diamonds
		},
	}
	sc.unseenBySuit[sSpades] = []cardcore.Card{aceOfSpades}
	sc.unseenBySuit[sHearts] = []cardcore.Card{
		c(rTwo, sHearts), c(rThree, sHearts),
	}
	sc.unseenBySuit[sDiamonds] = []cardcore.Card{c(rAce, sDiamonds)}
	sc.unseenBySuit[sClubs] = []cardcore.Card{
		twoOfClubs, c(rThree, sClubs),
	}
	return sc
}
