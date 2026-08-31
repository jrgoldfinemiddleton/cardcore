// Package tienlen implements the Tiến Lên climbing card game.
//
// Tiến Lên is a Vietnamese climbing (shedding) card game for two to four
// players with no partnerships. Each player is dealt thirteen cards and
// races to shed them all by beating the combination on the pile; play
// continues until only one player still holds cards.
//
// # Combinations
//
// Legal plays are the single, pair, triple, four-of-a-kind, straight
// (three or more consecutive cards), and pair sequence (three or more
// consecutive pairs). Straights and pair sequences may not contain 2s.
// A combination is beaten only by a higher combination of the same type
// and length, comparing highest cards by rank first, then suit. Ranks
// run 3 (low) through A, then 2 (high); suits run ♠ < ♣ < ♦ < ♥, so the
// 3♠ is the lowest card in the deck and the 2♥ the highest.
//
// # Chops
//
// The bomb-class combinations — four-of-a-kind and pair sequences — are
// the only exception to same-type matching: per the variant's chop table,
// they chop 2s and other bomb-class combinations, earning payments from
// the chopped player. Betting is always on: players agree on a stake
// before play, and chop payments and penalties settle at the end of the
// hand. No combination beats triple 2s.
//
// # Variants
//
// Rulesets are selected with the Variant type. The San Diego "Killer"
// ruleset ships first; the canonical Southern (Standard) ruleset
// follows. This package currently provides combination classification,
// comparison, and the Killer chop table; the game state machine is a
// later step.
//
// See ADR-006 (doc/decisions/006-rules-driven-development.md) and
// doc/games/tienlen/rules.md.
package tienlen
