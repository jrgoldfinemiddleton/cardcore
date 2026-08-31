package tienlen

import "errors"

// ErrWrongPhase is returned when an operation is attempted in a game
// phase that does not allow it.
var ErrWrongPhase = errors.New("tienlen: wrong phase")

// ErrOutOfTurn is returned when a player attempts to act when it is not
// their turn.
var ErrOutOfTurn = errors.New("tienlen: out of turn")

// ErrIllegalMove is returned when a player attempts a play that violates
// the rules of Tiến Lên.
var ErrIllegalMove = errors.New("tienlen: illegal move")
