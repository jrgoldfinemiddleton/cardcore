// Package ai provides computer-controlled players for Hearts.
//
// Each difficulty level is a separate type satisfying the hearts.Player
// interface. All implementations use only the Go standard library.
//
// Players receive a copy of the game state and may mutate it freely
// for analysis. Randomized players accept an explicit *rand.Rand for
// deterministic, reproducible play.
package ai
