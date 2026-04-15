package hearts

import "github.com/jrgoldfinemiddleton/cardcore"

// Player is the interface that any Hearts player — human or AI — must
// satisfy. Methods receive a copy of the game state and may mutate it
// freely (e.g., for simulation). The seat parameter identifies which
// player is acting, allowing a single Player instance to play multiple
// seats.
type Player interface {
	// ChoosePass selects three cards to pass from the hand at seat.
	ChoosePass(g *Game, seat Seat) [PassCount]cardcore.Card

	// ChoosePlay selects a card to play from the hand at seat.
	// The player should call g.LegalMoves(seat) to determine which
	// cards are legal.
	ChoosePlay(g *Game, seat Seat) cardcore.Card
}
