// Package ai provides computer-controlled players for Hearts.
//
// Two implementations are currently provided:
//
//   - [Random] picks a uniformly random legal move. It serves as a
//     baseline opponent and as a control in statistical tests.
//   - [Heuristic] applies hand-crafted Hearts strategy: shedding high
//     cards on the pass, ducking tricks when safe, dumping the Queen
//     of Spades on opponents, and attempting to shoot the moon when
//     the hand supports it.
//
// Both types satisfy the [hearts.Player] interface and use only the
// Go standard library. Each accepts an explicit *rand.Rand at
// construction so that play is deterministic and reproducible from a
// seed.
//
// Players receive the live *[hearts.Game] in ChoosePass and ChoosePlay.
// They must treat it as read-only; mutating game state from a Player
// method is a bug.
package ai
