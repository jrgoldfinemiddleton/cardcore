package ai

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/jrgoldfinemiddleton/cardcore"
	"github.com/jrgoldfinemiddleton/cardcore/games/hearts"
)

// TestFingerprintDeterminism verifies fingerprint produces the same
// value when called twice with identical inputs.
func TestFingerprintDeterminism(t *testing.T) {
	g := baseFingerprintGame()

	got1 := fingerprint(g, hearts.South)
	got2 := fingerprint(g, hearts.South)

	if got1 != got2 {
		t.Errorf("fingerprint not deterministic: got %d then %d", got1, got2)
	}
}

// TestFingerprintDistinguishesDecisionPoints verifies fingerprint
// produces a distinct value when any scalar in the decision tuple
// (seat, Phase, Round, TrickNum, Trick.Count) changes.
func TestFingerprintDistinguishesDecisionPoints(t *testing.T) {
	base := baseFingerprintGame()
	baseSeat := hearts.South
	baseFP := fingerprint(base, baseSeat)

	tests := []struct {
		name string
		mut  func(*hearts.Game) hearts.Seat
	}{
		{"different seat (West)", func(g *hearts.Game) hearts.Seat { return hearts.West }},
		{"different seat (North)", func(g *hearts.Game) hearts.Seat { return hearts.North }},
		{"different seat (East)", func(g *hearts.Game) hearts.Seat { return hearts.East }},
		{"different Phase (PhasePass)", func(g *hearts.Game) hearts.Seat {
			g.Phase = hearts.PhasePass
			return baseSeat
		}},
		{"different Round", func(g *hearts.Game) hearts.Seat {
			g.Round = 1
			return baseSeat
		}},
		{"different TrickNum", func(g *hearts.Game) hearts.Seat {
			g.TrickNum = 5
			return baseSeat
		}},
		{"different Trick.Count", func(g *hearts.Game) hearts.Seat {
			g.Trick.Count = 2
			return baseSeat
		}},
	}

	seen := map[uint64]string{baseFP: "base"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := baseFingerprintGame()
			seat := tt.mut(g)
			got := fingerprint(g, seat)
			if got == baseFP {
				t.Errorf("fingerprint did not change for %q: got %d (same as base)", tt.name, got)
			}
			if prev, ok := seen[got]; ok {
				t.Errorf("fingerprint collision: %q produced %d, already seen as %q",
					tt.name, got, prev)
			}
			seen[got] = tt.name
		})
	}
}

// TestFingerprintIgnoresExcludedFields verifies fingerprint is
// unchanged when fields outside the decision tuple are mutated.
func TestFingerprintIgnoresExcludedFields(t *testing.T) {
	base := baseFingerprintGame()
	seat := hearts.South
	want := fingerprint(base, seat)

	tests := []struct {
		name string
		mut  func(*hearts.Game)
	}{
		{"opponent hand", func(g *hearts.Game) {
			g.Hands[hearts.West] = nil
		}},
		{"acting seat hand", func(g *hearts.Game) {
			g.Hands[seat] = nil
		}},
		{"Scores", func(g *hearts.Game) {
			g.Scores[0] = 42
		}},
		{"RoundPts", func(g *hearts.Game) {
			g.RoundPts[1] = 13
		}},
		{"TrickHistory", func(g *hearts.Game) {
			g.TrickHistory = []hearts.Trick{{Leader: hearts.South, Count: hearts.NumPlayers}}
		}},
		{"HeartsBroken", func(g *hearts.Game) {
			g.HeartsBroken = true
		}},
		{"PassDir", func(g *hearts.Game) {
			g.PassDir = hearts.PassRight
		}},
		{"Turn", func(g *hearts.Game) {
			g.Turn = hearts.East
		}},
		{"PassHistory", func(g *hearts.Game) {
			g.PassHistory[seat] = [hearts.PassCount]cardcore.Card{
				twoOfClubs, c(rThree, sClubs), c(rFour, sClubs),
			}
		}},
		{"Trick.Leader", func(g *hearts.Game) {
			g.Trick.Leader = hearts.West
		}},
		{"Trick.Cards", func(g *hearts.Game) {
			g.Trick.Cards[hearts.South] = aceOfSpades
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := baseFingerprintGame()
			tt.mut(g)
			got := fingerprint(g, seat)
			if got != want {
				t.Errorf("fingerprint changed when mutating excluded field %q: got %d, want %d",
					tt.name, got, want)
			}
		})
	}
}

// TestDeriveRNGPurity verifies deriveRNG returns RNGs that produce
// identical sequences when called with identical inputs.
func TestDeriveRNGPurity(t *testing.T) {
	slot := [2]uint64{1, 2}
	fp := uint64(42)
	indices := []uint64{7, 11}

	r1 := deriveRNG(slot, fp, indices...)
	r2 := deriveRNG(slot, fp, indices...)

	const samples = 100
	for i := range samples {
		got := r1.Uint64()
		want := r2.Uint64()
		if got != want {
			t.Fatalf("deriveRNG not pure at sample %d: got %d, want %d", i, got, want)
		}
	}
}

// TestDeriveRNGDifferentInputsDiffer verifies deriveRNG produces a
// different first value when any of slot, fp, or indices changes.
func TestDeriveRNGDifferentInputsDiffer(t *testing.T) {
	baseSlot := [2]uint64{1, 2}
	baseFP := uint64(42)
	baseIndices := []uint64{7, 11}
	baseFirst := deriveRNG(baseSlot, baseFP, baseIndices...).Uint64()

	tests := []struct {
		name    string
		slot    [2]uint64
		fp      uint64
		indices []uint64
	}{
		{"different slot[0]", [2]uint64{99, 2}, baseFP, baseIndices},
		{"different slot[1]", [2]uint64{1, 99}, baseFP, baseIndices},
		{"different fp", baseSlot, 99, baseIndices},
		{"different indices values", baseSlot, baseFP, []uint64{99, 11}},
		{"different indices order", baseSlot, baseFP, []uint64{11, 7}},
		{"different indices length", baseSlot, baseFP, []uint64{7, 11, 0}},
		{"empty indices", baseSlot, baseFP, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveRNG(tt.slot, tt.fp, tt.indices...).Uint64()
			if got == baseFirst {
				t.Errorf("deriveRNG: same first value for differing inputs: got %d, want != %d",
					got, baseFirst)
			}
		})
	}
}

// TestNewPIMCPanics verifies NewPIMC panics on each invalid argument
// (nil rng, samples <= 0, nil factory, workers <= 0) and accepts the
// happy path.
func TestNewPIMCPanics(t *testing.T) {
	validRNG := rand.New(rand.NewPCG(1, 2))
	validFactory := func(r *rand.Rand) hearts.Player { return NewRandom(r) }

	tests := []struct {
		name    string
		rng     *rand.Rand
		samples int
		factory func(*rand.Rand) hearts.Player
		workers int
		wantPan bool
	}{
		{"nil rng", nil, 10, validFactory, 1, true},
		{"zero samples", validRNG, 0, validFactory, 1, true},
		{"negative samples", validRNG, -1, validFactory, 1, true},
		{"nil factory", validRNG, 10, nil, 1, true},
		{"zero workers", validRNG, 10, validFactory, 0, true},
		{"negative workers", validRNG, 10, validFactory, -1, true},
		{"happy path", validRNG, 10, validFactory, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantPan && r == nil {
					t.Errorf("NewPIMC did not panic; want panic")
				}
				if !tt.wantPan && r != nil {
					t.Errorf("NewPIMC panicked unexpectedly: %v", r)
				}
			}()
			_ = NewPIMC(tt.rng, tt.samples, tt.factory, tt.workers)
		})
	}
}

// TestNewPIMCCapturesDistinctPassSeeds verifies differently-seeded
// parent RNGs produce distinct captured seed slots, and identically-
// seeded parents produce identical slots.
func TestNewPIMCCapturesDistinctPassSeeds(t *testing.T) {
	factory := func(r *rand.Rand) hearts.Player { return NewRandom(r) }

	p1 := NewPIMC(rand.New(rand.NewPCG(1, 2)), 10, factory, 1)
	p2 := NewPIMC(rand.New(rand.NewPCG(3, 4)), 10, factory, 1)

	if p1.passSeed == p2.passSeed {
		t.Errorf("PIMCs with different parent seeds captured identical passSeed: %v", p1.passSeed)
	}
	if p1.sampleSeed == p2.sampleSeed {
		t.Errorf("PIMCs with different parent seeds captured identical sampleSeed: %v",
			p1.sampleSeed)
	}
	if p1.tiebreakSeed == p2.tiebreakSeed {
		t.Errorf("PIMCs with different parent seeds captured identical tiebreakSeed: %v",
			p1.tiebreakSeed)
	}

	pSame1 := NewPIMC(rand.New(rand.NewPCG(1, 2)), 10, factory, 1)
	pSame2 := NewPIMC(rand.New(rand.NewPCG(1, 2)), 10, factory, 1)
	if pSame1.passSeed != pSame2.passSeed {
		t.Errorf("identically-seeded PIMCs disagree on passSeed: %v vs %v",
			pSame1.passSeed, pSame2.passSeed)
	}
}

// TestPIMCChoosePassDelegatesToHeuristic verifies the captured
// passSeed flows through fingerprint and deriveRNG into the
// Heuristic's tiebreak shuffle, by constructing a hand with a
// deliberate passScore tie and observing that different parent seeds
// produce different pass selections.
func TestPIMCChoosePassDelegatesToHeuristic(t *testing.T) {
	// South's hand is constructed so that A♣ and A♦ tie at the top of
	// passScore: both score rank=12 + 0 (not a spade honor) + 0 (not a
	// heart) + 0 (suitLen=4, no short-suit bonus). The third pass slot
	// is J♠ at score 9. Different RNG-driven shuffles in Heuristic
	// produce [A♣, A♦, J♠] vs [A♦, A♣, J♠] — different arrays,
	// proving the captured passSeed flows through fingerprint and
	// deriveRNG into the Heuristic's tiebreak shuffle.
	factory := func(r *rand.Rand) hearts.Player { return NewRandom(r) }

	makeGame := func() *hearts.Game {
		g := hearts.New()
		g.Phase = hearts.PhasePass
		g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
			twoOfClubs,
			c(rThree, sClubs),
			c(rFive, sClubs),
			c(rAce, sClubs),
			c(rThree, sDiamonds),
			c(rFour, sDiamonds),
			c(rSix, sDiamonds),
			c(rAce, sDiamonds),
			c(rSeven, sSpades),
			c(rEight, sSpades),
			c(rNine, sSpades),
			c(rTen, sSpades),
			c(rJack, sSpades),
		})
		g.Hands[hearts.West] = cardcore.NewHand(nil)
		g.Hands[hearts.North] = cardcore.NewHand(nil)
		g.Hands[hearts.East] = cardcore.NewHand(nil)
		return g
	}

	p1 := NewPIMC(rand.New(rand.NewPCG(1, 2)), 10, factory, 1)
	p2 := NewPIMC(rand.New(rand.NewPCG(1, 2)), 10, factory, 1)

	got1 := p1.ChoosePass(makeGame(), hearts.South)
	got2 := p2.ChoosePass(makeGame(), hearts.South)
	if got1 != got2 {
		t.Errorf("identically-seeded PIMCs disagree on ChoosePass: got %v and %v", got1, got2)
	}

	// A single seed pair can collide on the same pass by luck even when plumbing works;
	// requiring all N to match is what would indicate the passSeed isn't flowing through.
	const tries = 8
	allMatch := true
	for i := uint64(100); i < 100+tries; i++ {
		other := NewPIMC(rand.New(rand.NewPCG(i, i+1)), 10, factory, 1)
		if other.ChoosePass(makeGame(), hearts.South) != got1 {
			allMatch = false
			break
		}
	}
	if allMatch {
		t.Errorf("ChoosePass returned identical result for %d distinct seeds; passSeed broken",
			tries)
	}
}

// TestPIMCChoosePlaySingleLegalMove verifies ChoosePlay returns the
// only legal card without invoking the rollout factory.
func TestPIMCChoosePlaySingleLegalMove(t *testing.T) {
	// Build a position at trick 12 (TrickNum=12, 0-indexed) where South
	// has exactly one card left: 7♠. The rollout factory panics if
	// called, proving the short-circuit fires.
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.Turn = hearts.South
	g.TrickNum = 12
	g.Trick = hearts.Trick{Leader: hearts.South, Count: 0}
	g.Hands[hearts.South] = cardcore.NewHand([]cardcore.Card{
		c(rSeven, sSpades),
	})
	g.Hands[hearts.West] = cardcore.NewHand([]cardcore.Card{
		c(rEight, sSpades),
	})
	g.Hands[hearts.North] = cardcore.NewHand([]cardcore.Card{
		c(rNine, sSpades),
	})
	g.Hands[hearts.East] = cardcore.NewHand([]cardcore.Card{
		c(rTen, sSpades),
	})
	g.HeartsBroken = true

	panicFactory := func(_ *rand.Rand) hearts.Player {
		panic("factory must not be called for single legal move")
	}
	p := NewPIMC(rand.New(rand.NewPCG(1, 2)), 100, panicFactory, 1)

	got := p.ChoosePlay(g, hearts.South)
	want := c(rSeven, sSpades)
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestPIMCChoosePlayDeterminism verifies the same (seed, game, seat)
// produces the same card across multiple calls.
func TestPIMCChoosePlayDeterminism(t *testing.T) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }

	g := freshPlayGame(t)

	p1 := NewPIMC(rand.New(rand.NewPCG(42, 43)), 10, factory, 1)
	p2 := NewPIMC(rand.New(rand.NewPCG(42, 43)), 10, factory, 1)

	got1 := p1.ChoosePlay(g.Clone(), g.Turn)
	got2 := p2.ChoosePlay(g.Clone(), g.Turn)
	if got1 != got2 {
		t.Errorf("identically-seeded PIMCs disagree: got %v and %v", got1, got2)
	}
}

// TestPIMCChoosePlayDeterminismAcrossWorkers verifies the determinism
// contract holds across varying worker counts: same (seed, g, seat)
// produces the same card for W in {1, 2, 4, 8, 16}.
func TestPIMCChoosePlayDeterminismAcrossWorkers(t *testing.T) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }

	g := freshPlayGame(t)
	seat := g.Turn

	workerCounts := []int{1, 2, 4, 8, 16}
	seed := [2]uint64{42, 43}

	baseline := NewPIMC(rand.New(rand.NewPCG(seed[0], seed[1])), 20, factory, workerCounts[0])
	want := baseline.ChoosePlay(g.Clone(), seat)

	for _, w := range workerCounts[1:] {
		p := NewPIMC(rand.New(rand.NewPCG(seed[0], seed[1])), 20, factory, w)
		got := p.ChoosePlay(g.Clone(), seat)
		if got != want {
			t.Errorf("workers=%d: got %v, want %v (from workers=%d)", w, got, want, workerCounts[0])
		}
	}
}

// TestPIMCChoosePlayDifferentSeeds verifies the RNG seed actually
// flows through by checking that at least one of several differently-
// seeded PIMCs picks a different card.
func TestPIMCChoosePlayDifferentSeeds(t *testing.T) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }

	// Advance past trick 0 (forced 2♣ lead) so the leader has a real
	// choice and RNG variation can surface.
	g := freshPlayGame(t)
	policy := firstLegalPolicy{}
	for g.TrickNum == 0 {
		seat := g.Turn
		card := policy.ChoosePlay(g.Clone(), seat)
		if err := g.PlayCard(seat, card); err != nil {
			t.Fatalf("PlayCard(%d, %v): %v", seat, card, err)
		}
	}

	baseline := NewPIMC(rand.New(rand.NewPCG(1, 2)), 10, factory, 1)
	baseCard := baseline.ChoosePlay(g.Clone(), g.Turn)

	const tries = 20
	for i := uint64(100); i < 100+tries; i++ {
		p := NewPIMC(rand.New(rand.NewPCG(i, i+1)), 10, factory, 1)
		if p.ChoosePlay(g.Clone(), g.Turn) != baseCard {
			return // success: at least one seed diverged
		}
	}
	t.Errorf("ChoosePlay returned %v for all %d distinct seeds; RNG not flowing through",
		baseCard, tries)
}

// CHECKME: new information-leakage fuzz test — mutates forbidden
// fields, asserts ChoosePlay unchanged

// TestPIMCChoosePlayInformationLeakage verifies that ChoosePlay depends
// only on fields the seat is legitimately allowed to read. It runs
// ChoosePlay on a baseline game state, then mutates each forbidden
// field to a different legal value and asserts the chosen card is
// unchanged.
func TestPIMCChoosePlayInformationLeakage(t *testing.T) {
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }
	seed := [2]uint64{99, 100}

	// Use a PassHold round so there are no hard-pass constraints
	// that would make opponent-hand swaps invalid.
	g := freshPlayGameOnRound(t, 3)
	seat := g.Turn

	baseline := NewPIMC(rand.New(rand.NewPCG(seed[0], seed[1])), 20, factory, 2)
	want := baseline.ChoosePlay(g.Clone(), seat)

	// Build opponent seat list (all seats except the acting seat).
	var opponents [3]hearts.Seat
	idx := 0
	for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
		if s != seat {
			opponents[idx] = s
			idx++
		}
	}

	mutations := []struct {
		name string
		mut  func(*hearts.Game)
	}{
		{"redistribute opponent hands", func(g *hearts.Game) {
			// Collect all opponent cards into one pool, then
			// redistribute in rotated order: opponent 0 gets
			// opponent 1's original cards, etc.
			hands := [3][]cardcore.Card{
				g.Hands[opponents[0]].Cards,
				g.Hands[opponents[1]].Cards,
				g.Hands[opponents[2]].Cards,
			}
			g.Hands[opponents[0]] = cardcore.NewHand(hands[1])
			g.Hands[opponents[1]] = cardcore.NewHand(hands[2])
			g.Hands[opponents[2]] = cardcore.NewHand(hands[0])
		}},
		{"PassHistory opponent", func(g *hearts.Game) {
			other := nextSeat(seat)
			g.PassHistory[other] = [hearts.PassCount]cardcore.Card{
				aceOfSpades, kingOfSpades, queenOfSpades,
			}
		}},
	}

	for _, tt := range mutations {
		t.Run(tt.name, func(t *testing.T) {
			mutated := g.Clone()
			tt.mut(mutated)

			p := NewPIMC(rand.New(rand.NewPCG(seed[0], seed[1])), 20, factory, 2)
			got := p.ChoosePlay(mutated, seat)
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

// TestPIMCFullGameIntegration plays complete games with a PIMC player
// at every seat and verifies structural invariants (point conservation,
// game reaches PhaseEnd, winner is correct).
func TestPIMCFullGameIntegration(t *testing.T) {
	const (
		numGames   = 5
		maxRounds  = 20
		numSamples = 10
	)
	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }

	for game := range numGames {
		seed := uint64(game) + 1
		var players [hearts.NumPlayers]hearts.Player
		for seat := range hearts.NumPlayers {
			playerSeed := seed*uint64(hearts.NumPlayers) + uint64(seat)
			players[seat] = NewPIMC(
				rand.New(rand.NewPCG(playerSeed, playerSeed+1)),
				numSamples,
				factory,
				1,
			)
		}

		g := hearts.New()
		for range maxRounds {
			playRoundWithPlayers(t, g, players, seed)
			if g.Phase == hearts.PhaseEnd {
				break
			}
		}

		if g.Phase != hearts.PhaseEnd {
			t.Fatalf("game %d: did not end within %d rounds", game, maxRounds)
		}

		winner, err := g.Winner()
		if err != nil {
			t.Fatalf("game %d: Winner error: %v", game, err)
		}
		for i := hearts.Seat(0); i < hearts.NumPlayers; i++ {
			if g.Scores[i] < g.Scores[winner] {
				t.Errorf("game %d: seat %d score %d < winner %d score %d",
					game, i, g.Scores[i], winner, g.Scores[winner])
			}
		}
	}
}

// TestPIMCStatisticalCompetenceIntegration runs 100 complete games with
// one PIMC player against three Heuristic opponents and asserts the
// PIMC wins more than 30% of the time. The PIMC seat rotates across
// games (game % 4) so the per-seat assertion catches position-dependent
// skill that the aggregate would hide. The master seed is time-based
// and logged on every run so a failure can be reproduced by temporarily
// hardcoding the logged seed.
func TestPIMCStatisticalCompetenceIntegration(t *testing.T) {
	const (
		numGames          = 100
		maxRounds         = 20
		numSamples        = 30
		numWorkers        = 4
		minWinRate        = 0.30
		minWinRatePerSeat = 0.30
	)
	masterSeed := uint64(time.Now().UnixNano())
	t.Logf("masterSeed = %d (hardcode this to reproduce a failure)", masterSeed)

	factory := func(r *rand.Rand) hearts.Player { return NewHeuristic(r) }

	var (
		winsPerSeat  [hearts.NumPlayers]int
		gamesPerSeat [hearts.NumPlayers]int
	)

	for game := range numGames {
		pimcSeat := hearts.Seat(game % int(hearts.NumPlayers))

		var players [hearts.NumPlayers]hearts.Player
		for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
			playerSeed := masterSeed + uint64(game)*uint64(hearts.NumPlayers) + uint64(seat)
			if seat == pimcSeat {
				players[seat] = NewPIMC(
					rand.New(rand.NewPCG(playerSeed, playerSeed+1)),
					numSamples,
					factory,
					numWorkers,
				)
			} else {
				players[seat] = newSeededHeuristic(playerSeed)
			}
		}

		g := hearts.New()
		for range maxRounds {
			playRoundWithPlayers(t, g, players, uint64(game))
			if g.Phase == hearts.PhaseEnd {
				break
			}
		}
		if g.Phase != hearts.PhaseEnd {
			t.Fatalf("game %d: did not end within %d rounds", game, maxRounds)
		}

		winner, err := g.Winner()
		if err != nil {
			t.Fatalf("game %d: Winner error: %v", game, err)
		}
		gamesPerSeat[pimcSeat]++
		if winner == pimcSeat {
			winsPerSeat[pimcSeat]++
		}
	}

	totalWins := 0
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		totalWins += winsPerSeat[seat]
	}
	aggregateRate := float64(totalWins) / float64(numGames)

	t.Logf("PIMC wins: %d/%d aggregate (%.1f%%)",
		totalWins, numGames, aggregateRate*100)
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		t.Logf("  seat %d: %d/%d (%.1f%%)",
			seat, winsPerSeat[seat], gamesPerSeat[seat],
			float64(winsPerSeat[seat])*100/float64(gamesPerSeat[seat]))
	}

	if aggregateRate <= minWinRate {
		t.Errorf("aggregate: PIMC won %d/%d (%.1f%%); want > %.0f%%",
			totalWins, numGames, aggregateRate*100, minWinRate*100)
	}
	for seat := hearts.Seat(0); seat < hearts.NumPlayers; seat++ {
		seatRate := float64(winsPerSeat[seat]) / float64(gamesPerSeat[seat])
		if seatRate <= minWinRatePerSeat {
			t.Errorf("seat %d: PIMC won %d/%d (%.1f%%); want > %.0f%%",
				seat, winsPerSeat[seat], gamesPerSeat[seat],
				seatRate*100, minWinRatePerSeat*100)
		}
	}
}

// baseFingerprintGame returns a minimal *hearts.Game suitable for
// fingerprint testing. Callers mutate fields to produce variants.
func baseFingerprintGame() *hearts.Game {
	g := hearts.New()
	g.Phase = hearts.PhasePlay
	g.Round = 0
	g.TrickNum = 0
	g.Trick.Count = 0
	return g
}

// freshPlayGameOnRound plays through complete rounds using
// firstLegalPolicy until the target round is reached, then returns
// the game in PhasePlay of that round.
func freshPlayGameOnRound(t *testing.T, targetRound int) *hearts.Game {
	t.Helper()
	g := hearts.New()
	policy := firstLegalPolicy{}
	for g.Round < targetRound || g.Phase != hearts.PhasePlay {
		switch g.Phase {
		case hearts.PhaseDeal:
			if err := g.Deal(); err != nil {
				t.Fatalf("round %d: Deal: %v", g.Round, err)
			}
		case hearts.PhasePass:
			for s := hearts.Seat(0); s < hearts.NumPlayers; s++ {
				cards := policy.ChoosePass(g.Clone(), s)
				if err := g.SetPass(s, cards); err != nil {
					t.Fatalf("round %d: SetPass(%d): %v", g.Round, s, err)
				}
			}
		case hearts.PhasePlay:
			seat := g.Turn
			card := policy.ChoosePlay(g.Clone(), seat)
			if err := g.PlayCard(seat, card); err != nil {
				t.Fatalf("round %d: PlayCard(%d, %v): %v",
					g.Round, seat, card, err)
			}
		case hearts.PhaseScore:
			if err := g.EndRound(); err != nil {
				t.Fatalf("round %d: EndRound: %v", g.Round, err)
			}
		case hearts.PhaseEnd:
			t.Fatalf("game ended before reaching round %d", targetRound)
		default:
			t.Fatalf("unexpected phase %d", g.Phase)
		}
	}
	return g
}
