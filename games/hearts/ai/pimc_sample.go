package ai

import (
	"math/rand/v2"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// sampleConstraints captures everything a sampler needs about a position
// to draw uniformly random feasible deals: the unseen cards bucketed by
// suit, each opponent's remaining hand-size capacity, each opponent's
// per-suit voids, and any cards hard-assigned to a recipient via the
// pass constraint.
//
// A "feasible deal" assigns every unseen card to exactly one opponent
// such that every opponent ends with their target hand size and never
// receives a card in a suit they are void in. Cards hard-assigned via
// pass go directly to the recipient and are not part of the
// dynamic-programming (DP) walk that draws the rest.
//
// The struct is built once per ChoosePlay (constraints are identical
// across all N samples) and consumed read-only by the DP.
type sampleConstraints struct {
	// seat is the player whose decision is being evaluated. Excluded
	// from opponents.
	seat hearts.Seat

	// opponents lists the three non-seat seats. Order is the canonical
	// cyclic order starting from (seat+1) % NumPlayers, which makes the
	// DP's opponent indexing deterministic given seat.
	opponents [hearts.NumPlayers - 1]hearts.Seat

	// handSizes is the number of cards the DP must deal each opponent
	// from the unseen pool. Equals the opponent's true hand size minus
	// any hard-pass cards already assigned. Indexed parallel to
	// opponents (not by raw Seat).
	handSizes [hearts.NumPlayers - 1]int

	// voids[i][suit] is true when opponents[i] cannot hold any card of
	// suit. Sources: everything analyze surfaces (completed-trick
	// off-suit plays plus exhausted-suit card counting) and current-trick
	// off-suit plays, which analyze does not see.
	voids [hearts.NumPlayers - 1][cardcore.NumSuits]bool

	// unseenBySuit groups every card not visible to seat (not in seat's
	// hand, not played in any completed trick, not played in the
	// current trick, not hard-assigned via pass) by suit. The DP
	// distributes these across opponents.
	unseenBySuit [cardcore.NumSuits][]cardcore.Card

	// hardAssigned[i] is the list of cards already known to be in
	// opponents[i]'s hand from the pass constraint (cards seat passed
	// to opponents[i] that have not yet been played). The sampler
	// installs these directly into the deal alongside the DP's output.
	hardAssigned [hearts.NumPlayers - 1][]cardcore.Card
}

// stateKey packs a DP state into a uint32 for map lookup.
// Layout (low to high): cap0 [4 bits] | cap1 [4 bits] | cap2 [4 bits] | suitIdx [2 bits].
// Capacities range 0–13, fitting in 4 bits each. suitIdx ranges 0–3.
type stateKey uint32

// sampleDealDP holds the precomputed DP table mapping each reachable
// state to the number of card-level feasible deals below it. Built once
// per ChoosePlay and walked N times (once per sample) with different
// RNGs.
type sampleDealDP struct {
	// table maps each reachable state to the count of feasible
	// card-level deals in the subtree rooted at that state.
	table map[stateKey]int64

	// constraints is the position's constraints, retained for the
	// sample walk phase (which needs unseenBySuit, voids, etc.).
	constraints *sampleConstraints
}

// dpSuitOrder is the fixed suit processing order for the DP:
// Spades, Hearts, Diamonds, Clubs. Chosen arbitrarily; must be
// consistent between table build and sample walk.
var dpSuitOrder = [cardcore.NumSuits]cardcore.Suit{
	cardcore.Spades,
	cardcore.Hearts,
	cardcore.Diamonds,
	cardcore.Clubs,
}

// indexOfOpponent returns the index in c.opponents at which seat appears.
// Panics if seat is not present (programmer error: caller passed seat's
// own seat or a malformed opponents array).
func (c *sampleConstraints) indexOfOpponent(seat hearts.Seat) int {
	for i, opp := range c.opponents {
		if opp == seat {
			return i
		}
	}
	panic("ai: indexOfOpponent: seat not in opponents")
}

// build recursively fills dp.table for the state (suitIdx, cap0, cap1, cap2).
// Returns the number of feasible card-level deals in the subtree.
func (dp *sampleDealDP) build(suitIdx, cap0, cap1, cap2 int) int64 {
	key := makeStateKey(suitIdx, cap0, cap1, cap2)
	if v, ok := dp.table[key]; ok {
		return v
	}

	suit := dpSuitOrder[suitIdx]
	k := len(dp.constraints.unseenBySuit[suit])

	// Base case: last suit. Only one split can satisfy remaining
	// capacities exactly.
	if suitIdx == cardcore.NumSuits-1 {
		if cap0+cap1+cap2 != k {
			dp.table[key] = 0
			return 0
		}
		if cap0 > 0 && dp.constraints.voids[0][suit] {
			dp.table[key] = 0
			return 0
		}
		if cap1 > 0 && dp.constraints.voids[1][suit] {
			dp.table[key] = 0
			return 0
		}
		if cap2 > 0 && dp.constraints.voids[2][suit] {
			dp.table[key] = 0
			return 0
		}
		v := multinomial(k, cap0, cap1)
		dp.table[key] = v
		return v
	}

	// Recursive case: enumerate all legal splits (w0, w1, w2) where
	// w0+w1+w2 == k, each wi <= capi, and wi == 0 if opponent i is
	// void in this suit.
	var total int64
	max0 := cap0
	if dp.constraints.voids[0][suit] {
		max0 = 0
	}
	max1 := cap1
	if dp.constraints.voids[1][suit] {
		max1 = 0
	}
	max2 := cap2
	if dp.constraints.voids[2][suit] {
		max2 = 0
	}

	for w0 := 0; w0 <= max0 && w0 <= k; w0++ {
		for w1 := 0; w1 <= max1 && w0+w1 <= k; w1++ {
			w2 := k - w0 - w1
			if w2 < 0 || w2 > max2 {
				continue
			}
			multi := multinomial(k, w0, w1)
			sub := dp.build(suitIdx+1, cap0-w0, cap1-w1, cap2-w2)
			total += multi * sub
		}
	}

	dp.table[key] = total
	return total
}

// sample walks the precomputed DP table with rng to produce one
// uniformly random feasible deal. The returned sampledDeal has seat's
// real hand at deal[seat] and sampled hands at the other three
// positions. Each opponent's hand includes both DP-assigned cards and
// any hard-assigned pass cards from the constraints.
func (dp *sampleDealDP) sample(
	g *hearts.Game, seat hearts.Seat, rng *rand.Rand,
) sampledDeal {
	c := dp.constraints

	// Accumulate cards per opponent during the walk.
	var oppCards [hearts.NumPlayers - 1][]cardcore.Card

	cap0, cap1, cap2 := c.handSizes[0], c.handSizes[1], c.handSizes[2]

	for suitIdx := range cardcore.NumSuits {
		suit := dpSuitOrder[suitIdx]
		cards := c.unseenBySuit[suit]
		k := len(cards)

		// Enumerate legal splits and their weights.
		type split struct {
			w      [hearts.NumPlayers - 1]int
			weight int64
		}
		var splits []split
		var totalWeight int64

		max0 := cap0
		if c.voids[0][suit] {
			max0 = 0
		}
		max1 := cap1
		if c.voids[1][suit] {
			max1 = 0
		}
		max2 := cap2
		if c.voids[2][suit] {
			max2 = 0
		}

		for w0 := 0; w0 <= max0 && w0 <= k; w0++ {
			for w1 := 0; w1 <= max1 && w0+w1 <= k; w1++ {
				w2 := k - w0 - w1
				if w2 < 0 || w2 > max2 {
					continue
				}
				subKey := makeStateKey(
					suitIdx+1, cap0-w0, cap1-w1, cap2-w2,
				)
				var sub int64
				if suitIdx == cardcore.NumSuits-1 {
					// Last suit: the split itself is the
					// leaf; sub-weight is 1.
					sub = 1
				} else {
					sub = dp.table[subKey]
				}
				if sub == 0 {
					continue
				}
				multi := multinomial(k, w0, w1)
				w := multi * sub
				splits = append(splits, split{
					w:      [hearts.NumPlayers - 1]int{w0, w1, w2},
					weight: w,
				})
				totalWeight += w
			}
		}

		// Pick a split proportional to weight.
		r := rng.Int64N(totalWeight)
		var chosen split
		for _, s := range splits {
			r -= s.weight
			if r < 0 {
				chosen = s
				break
			}
		}

		// Shuffle the suit's cards and deal them per the chosen split.
		perm := rng.Perm(k)
		idx := 0
		for opp := range hearts.NumPlayers - 1 {
			for range chosen.w[opp] {
				oppCards[opp] = append(oppCards[opp], cards[perm[idx]])
				idx++
			}
		}

		cap0 -= chosen.w[0]
		cap1 -= chosen.w[1]
		cap2 -= chosen.w[2]
	}

	// Install hard-assigned pass cards.
	for i := range hearts.NumPlayers - 1 {
		oppCards[i] = append(oppCards[i], c.hardAssigned[i]...)
	}

	// Build the deal.
	var deal sampledDeal
	deal[seat] = *cardcore.NewHand(g.Hands[seat].Cards)
	for i, opp := range c.opponents {
		deal[opp] = *cardcore.NewHand(oppCards[i])
	}
	return deal
}

// buildConstraints derives sampleConstraints from g and seat. It honors
// every public information source that bounds where unseen cards can be:
// completed-trick voids (via analyze), current-trick voids and played
// cards, and the hard pass constraint from g.PassHistory[seat].
//
// Hard pass-card assignments are pre-applied: cards seat passed that
// have not yet been played are removed from the unseen pool, added to
// the recipient's hardAssigned slice, and subtracted from the
// recipient's handSize capacity. The DP only sees the truly-residual
// unseen cards.
//
// Caller contract: g.Phase must be PhasePlay (so PassHistory is
// populated for non-PassHold rounds and the trick state is meaningful).
// Violation triggers a panic from downstream uses; this function does
// not validate phase itself.
func buildConstraints(g *hearts.Game, seat hearts.Seat) sampleConstraints {
	a := analyze(g, seat)

	c := sampleConstraints{seat: seat}
	for i := range c.opponents {
		c.opponents[i] = (seat + hearts.Seat(i) + 1) % hearts.NumPlayers
	}

	for i, opp := range c.opponents {
		c.handSizes[i] = len(g.Hands[opp].Cards)
		c.voids[i] = a.voids[opp]
	}

	// Add current-trick voids: any opponent who already played a card
	// in the in-progress trick that did not match the led suit has
	// publicly revealed a void in the led suit. analyze() only scans
	// completed tricks, so we add these here.
	if g.Trick.Count > 0 {
		ledSuit := g.Trick.LedSuit()
		for i := range g.Trick.Count {
			player := (g.Trick.Leader + hearts.Seat(i)) % hearts.NumPlayers
			if player == seat {
				continue
			}
			card := g.Trick.Cards[player]
			if card.Suit != ledSuit {
				oppIdx := c.indexOfOpponent(player)
				c.voids[oppIdx][ledSuit] = true
			}
		}
	}

	played := playedCardSet(g, &a)

	// Compute the unseen pool: every card except seat's hand and every
	// played card. Bucket by suit.
	for _, suit := range cardcore.AllSuits() {
		for _, rank := range cardcore.AllRanks() {
			card := cardcore.Card{Rank: rank, Suit: suit}
			if played[suit][rank] {
				continue
			}
			if g.Hands[seat].Contains(card) {
				continue
			}
			c.unseenBySuit[suit] = append(c.unseenBySuit[suit], card)
		}
	}

	// Apply hard pass constraint: every card seat passed that has not
	// been played is still in the recipient's hand. Move it from the
	// unseen pool into hardAssigned[recipient] and decrement the
	// recipient's DP capacity.
	if g.PassDir != hearts.PassHold {
		recipient := passTarget(seat, g.PassDir)
		recipientIdx := c.indexOfOpponent(recipient)
		for _, card := range g.PassHistory[seat] {
			if played[card.Suit][card.Rank] {
				continue
			}
			c.hardAssigned[recipientIdx] = append(c.hardAssigned[recipientIdx], card)
			c.handSizes[recipientIdx]--
			c.unseenBySuit[card.Suit] = removeCard(c.unseenBySuit[card.Suit], card)
		}
	}

	return c
}

// makeStateKey constructs a stateKey from suit index and per-opponent
// remaining capacities.
func makeStateKey(suitIdx int, cap0, cap1, cap2 int) stateKey {
	return stateKey(cap0 | cap1<<4 | cap2<<8 | suitIdx<<12)
}

// multinomial returns k! / (a! * b! * (k-a-b)!) for a+b <= k.
// Precondition: a, b >= 0 and a+b <= k. No overflow check (Hearts
// inputs are small enough for int64).
func multinomial(k, a, b int) int64 {
	// C(k,a) * C(k-a,b) = k!/(a!(k-a)!) * (k-a)!/(b!(k-a-b)!);
	// the (k-a)! cancels, leaving k! / (a! * b! * (k-a-b)!).
	return binomial(k, a) * binomial(k-a, b)
}

// binomial returns C(n, k) for 0 <= k <= n <= 13.
func binomial(n, k int) int64 {
	if k > n-k {
		k = n - k
	}
	var result int64 = 1
	for i := range k {
		result = result * int64(n-i) / int64(i+1)
	}
	return result
}

// newSampleDealDP builds the DP table from the given constraints. The
// table enumerates every reachable (suitIdx, cap0, cap1, cap2) state
// and stores the total number of card-level feasible deals in the
// subtree. The build is a memoized recursion over suits in dpSuitOrder.
func newSampleDealDP(c *sampleConstraints) *sampleDealDP {
	dp := &sampleDealDP{
		table:       make(map[stateKey]int64),
		constraints: c,
	}
	dp.build(0, c.handSizes[0], c.handSizes[1], c.handSizes[2])
	return dp
}

// playedCardSet returns the union of cards played in completed tricks
// (already tracked by analyze.played) and cards played so far in the
// in-progress trick. Indexed [suit][rank].
func playedCardSet(g *hearts.Game, a *analysis) [cardcore.NumSuits][cardcore.NumRanks]bool {
	played := a.played
	for i := range g.Trick.Count {
		player := (g.Trick.Leader + hearts.Seat(i)) % hearts.NumPlayers
		card := g.Trick.Cards[player]
		played[card.Suit][card.Rank] = true
	}
	return played
}

// removeCard returns cards with the first occurrence of target removed.
// The returned slice may share backing storage with the input. Panics
// if target is not present (programmer error: hard-assigned pass card
// was not in the unseen pool, meaning constraints are inconsistent).
func removeCard(cards []cardcore.Card, target cardcore.Card) []cardcore.Card {
	for i, c := range cards {
		if c == target {
			return append(cards[:i], cards[i+1:]...)
		}
	}
	panic("ai: removeCard: target not in slice")
}
