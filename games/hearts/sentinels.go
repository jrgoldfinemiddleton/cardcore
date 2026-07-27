package hearts

import "errors"

// ErrWrongPhase is returned when an operation is attempted in a game
// phase that does not allow it.
var ErrWrongPhase = errors.New("hearts: wrong phase")

// ErrOutOfTurn is returned when a player attempts to act when it is not
// their turn.
var ErrOutOfTurn = errors.New("hearts: out of turn")

// ErrIllegalMove is returned when a player attempts a play that violates
// the rules of Hearts.
var ErrIllegalMove = errors.New("hearts: illegal move")
