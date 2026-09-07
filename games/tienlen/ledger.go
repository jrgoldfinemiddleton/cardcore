package tienlen

import (
	"fmt"
	"slices"
)

// TransferReason identifies why value moved between two seats.
type TransferReason uint8

// Transfer reasons. Killer settles place payments and chop chains only;
// the Standard variant adds rotten, blank, and instant-win reasons.
const (
	// TransferPlace settles the last-place payment: the last-place seat
	// pays the hand's winner one stake.
	TransferPlace TransferReason = iota
	// TransferChop settles one payable chop chain: the last chopped seat
	// pays the last chopper the chain's escalated amount. The settling
	// chop is the chain's last chop of another seat's play — a self-chop
	// neither settles nor cancels — so From and To always differ.
	TransferChop
)

// Transfer records one movement of value between two seats, in the
// caller's smallest integral units (see Config.Stake). Amount is always
// positive; direction is carried entirely by From and To, and the two
// never coincide: a transfer always moves value between two different
// seats. Transfers are recorded in settlement order — a hand's chop
// chains in chain order, then its place transfer — and are never netted
// or merged: each transfer settles in full, independently of the others.
type Transfer struct {
	// Hand is the 0-based hand index the transfer settled in.
	Hand int
	// From is the paying seat.
	From Seat
	// To is the receiving seat.
	To Seat
	// Amount is the transferred value; always positive.
	Amount int
	// Reason is why the transfer occurred.
	Reason TransferReason
}

// chainKey identifies one payable chop chain within a hand: chains are
// numbered per pile.
type chainKey struct {
	// pile is the 1-based pile number within the hand.
	pile int
	// chain is the 1-based chain number within the pile.
	chain int
}

// Balances returns each seat's cumulative match balance in the caller's
// smallest integral units (see Config.Stake): positive is ahead and
// negative is behind. Balances are all zero until the first hand settles
// and always sum to zero. The result is a copy; mutating it does not
// affect the game.
func (g *Game) Balances() []int {
	return slices.Clone(g.balances)
}

// Transfers returns the match's settlement record: every transfer of
// every settled hand, in settlement order. The log grows only when a
// hand completes. The result is a copy; mutating it does not affect the
// game.
func (g *Game) Transfers() []Transfer {
	return slices.Clone(g.transfers)
}

// Winner reports the seat with the highest cumulative balance at match
// end. The boolean reports whether the win is outright: it is false when
// the highest balance is tied (a draw), in which case the seat is
// meaningless. It returns an error wrapping ErrWrongPhase when the match
// is not over.
func (g *Game) Winner() (Seat, bool, error) {
	if g.Phase != PhaseEnd {
		return 0, false, fmt.Errorf(
			"cannot determine a winner in phase %d: %w", g.Phase, ErrWrongPhase,
		)
	}
	winner := Seat(0)
	tied := false
	for seat := Seat(1); seat < Seat(len(g.balances)); seat++ {
		switch {
		case g.balances[seat] > g.balances[winner]:
			winner = seat
			tied = false
		case g.balances[seat] == g.balances[winner]:
			tied = true
		}
	}
	return winner, !tied, nil
}

// settleHand folds the current hand's events into transfers and applies
// them to the match balances. It runs exactly once per hand, at the
// transition into PhaseScore (see completeHand); the hand-ended event
// must already be in the log, because the fold reads the finishing order
// from it.
func (g *Game) settleHand() {
	for _, t := range foldHand(g.events, g.Config, g.Hand) {
		g.balances[t.From] -= t.Amount
		g.balances[t.To] += t.Amount
		g.transfers = append(g.transfers, t)
	}
}

// foldHand folds one hand's events into that hand's transfers. It
// returns nil when the log holds no hand-ended event for the hand: an
// incomplete hand never settles.
func foldHand(events []Event, cfg Config, hand int) []Transfer {
	switch cfg.Variant {
	case Killer:
		return foldKillerHand(events, cfg.Stake, hand)
	}
	panic(fmt.Sprintf("tienlen: unknown variant %d", cfg.Variant))
}

// foldKillerHand folds one Killer hand's events into transfers: one
// transfer per payable chop chain, settled at the chain's last
// inter-player chop — a self-chop neither settles nor cancels; it stops
// the escalation, freezing the chain at the last chop of another
// player's play — in chain order, then the last-place payment. The log
// must be engine-shaped: a chop never opens a pile (Chain > 0 implies
// Play > 0), and an ended hand's finishing order is complete.
func foldKillerHand(events []Event, stake, hand int) []Transfer {
	// piles[pile-1][play-1] is the seat that played the play'th card in
	// the pile'th pile. The last play in a pile is always the last
	// chopper in that pile's chain.
	var piles [][]Seat
	var order []chainKey
	// candidates[key] is the last inter-player chop in the chain
	// identified by key. It is the only play in the chain that pays.
	candidates := make(map[chainKey]PlayEvent)
	var places []Seat
	ended := false
	for _, e := range events {
		switch ev := e.(type) {
		case PlayEvent:
			if ev.Hand != hand {
				continue
			}
			// Ensure the piles slice is long enough to hold the pile number.
			for len(piles) < ev.Pile {
				piles = append(piles, nil)
			}
			piles[ev.Pile-1] = append(piles[ev.Pile-1], ev.Seat)
			// Skip non-chop plays: they neither start nor continue a chain.
			if ev.Chain == 0 {
				continue
			}
			key := chainKey{pile: ev.Pile, chain: ev.Chain}
			if piles[ev.Pile-1][ev.Play-1] == ev.Seat {
				// A self-chop stops the escalation without cancelling
				// the chain: the previous candidate stands.
				continue
			}
			if _, ok := candidates[key]; !ok {
				order = append(order, key)
			}
			// Update the candidate for the chain; the last inter-player chop in
			// the chain is the only one that pays.
			candidates[key] = ev
		case HandEndedEvent:
			if ev.Hand != hand {
				continue
			}
			places = ev.Places
			ended = true
		}
	}
	if !ended {
		return nil
	}
	transfers := make([]Transfer, 0, len(order)+1)
	for _, key := range order {
		ev := candidates[key]
		transfers = append(transfers, Transfer{
			Hand: hand,
			// The last inter-player chop in the chain pays the last
			// chopper the chain's escalated amount. The last chopper is
			// the player who played the last card in the pile, which is
			// always the player at Play-1 (0-based) in the pile.
			From:   piles[key.pile-1][ev.Play-1],
			To:     ev.Seat,
			Amount: ev.Depth * stake,
			Reason: TransferChop,
		})
	}
	return append(transfers, Transfer{
		Hand:   hand,
		From:   places[len(places)-1],
		To:     places[0],
		Amount: stake,
		Reason: TransferPlace,
	})
}
