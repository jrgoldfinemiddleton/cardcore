package tienlen

import "github.com/jrgoldfinemiddleton/cardcore"

// Player is the interface that any Tiến Lên player — human or AI — must
// satisfy. Methods receive the live game state and must not mutate it.
// An implementation that needs to simulate or look ahead must work on
// its own copy (see Game.Clone) rather than the value passed in.
// The seat parameter identifies which player is acting, allowing a
// single Player instance to play multiple seats.
//
// The caller is responsible for invoking methods at the correct time:
// ChooseDeclareAutoWin at most once per eligible seat during the
// declaration window (after Deal, before StartPlay), and ChoosePlay only
// on the seat's turn. Implementations may panic if called in an invalid
// state — precondition violations are programming errors, not
// recoverable conditions.
type Player interface {
	// ChoosePlay selects the cards to play as one combination, or
	// returns nil or an empty slice to pass. The player should call
	// g.LegalMoves(seat) and g.CanPass(seat) to determine which actions
	// are legal.
	ChoosePlay(g *Game, seat Seat) []cardcore.Card

	// ChooseDeclareAutoWin reports whether seat declares its automatic
	// win. Returning false is always legal: declaration is optional
	// under Killer rules.
	ChooseDeclareAutoWin(g *Game, seat Seat) bool
}
