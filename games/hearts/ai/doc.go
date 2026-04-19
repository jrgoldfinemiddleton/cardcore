// Package ai provides computer-controlled players for Hearts.
//
// Two implementations are provided:
//
//   - [Random] picks a uniformly random legal move.
//   - [Heuristic] applies hand-crafted Hearts strategy.
//
// Both satisfy the [hearts.Player] interface and use only the Go
// standard library. Each accepts an explicit *rand.Rand at
// construction so that play is reproducible from a seed.
package ai
