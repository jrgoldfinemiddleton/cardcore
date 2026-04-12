// Package hearts implements the classic Hearts card game.
//
// Hearts is a trick-taking card game for four players. The goal is to
// avoid taking penalty points: each heart card is worth one point, and the
// queen of spades is worth thirteen. The game ends when any player
// reaches 100 points, and the player with the lowest score wins.
//
// # Passing
//
// At the start of each round, players pass three cards to another
// player. The direction rotates each round: left, right, across, then
// a hold round (no passing). The cycle then repeats.
//
// # Play
//
// The player holding the two of clubs leads the first trick. Players
// must follow the led suit if able. The highest card of the led suit
// wins the trick, and the winner leads the next trick.
//
// Hearts cannot be led until a heart has been played on a previous
// trick ("breaking hearts"), unless the player has nothing but hearts.
// On the first trick, penalty cards (hearts and the queen of spades)
// may not be played unless the player has no other legal option.
//
// # Shooting the Moon
//
// If a single player takes all 26 penalty points in a round, that
// player scores zero and every other player receives 26 points instead.
package hearts
