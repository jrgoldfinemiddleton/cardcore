// Package ai provides computer-controlled players for Hearts.
//
// Three implementations are provided:
//
//   - [Random] picks a uniformly random legal move.
//   - [Heuristic] applies hand-crafted Hearts strategy.
//   - [PIMC] chooses plays via Perfect Information Monte Carlo
//     sampling: it samples constraint-satisfying deals of unseen
//     cards, simulates rollouts for each candidate move, and
//     selects the candidate with the lowest total leaf score.
//
// All satisfy the [hearts.Player] interface and use only the Go
// standard library. Each accepts an explicit *rand.Rand at
// construction so that play is reproducible from a seed.
package ai
