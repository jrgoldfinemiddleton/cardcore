# Hearts — Game Rules

Also known as: Black Lady, Chase the Lady, Rickety Kate, Black Queen,
Black Widow.

This document defines the rules of Hearts as implemented by the
`hearts` package. It serves as the specification that the implementation
is built against. The [Variants](#variants) section describes alternative
rules that may be supported in the future.

Primary references:
- [Pagat — Hearts](https://www.pagat.com/reverse/hearts.html) (John McLeod)
- [Bicycle Cards — Hearts](https://bicyclecards.com/how-to-play/hearts)

## Definitions

Unless otherwise noted, definitions describe the standard four-player
game. Variants may alter specific terms; see [Variants](#variants) for
details.

- **Trick**: A single turn of play in which each player contributes one
  card from their hand. A full trick contains four cards.
- **Round**: A complete sequence of the deal, the pass, play of all
  tricks until hands are exhausted, and scoring. A game consists of
  multiple rounds.
- **Penalty card**: A card that carries penalty points — any heart (1
  point each) or the Q♠ (Queen of Spades, 13 points).
- **Lead**: To play the first card of a trick. The player who leads
  chooses the suit for that trick.
- **Led suit**: The suit of the first card played in a trick.
- **Follow suit**: To play a card of the same suit that was led.
- **Void**: Having no cards of a particular suit in hand.
- **Slough**: To discard a card when unable to follow suit. Also
  spelled "sluff."
- **Painting**: Discarding a penalty card on a trick, forcing the trick
  winner to take the points.
- **Breaking hearts**: Playing a heart to a trick for the first time in
  a round. Once hearts are broken, hearts may be led.
- **Shooting the moon**: Taking all penalty cards in a single round.
- **Hold hand**: The round in which no cards are passed. The fourth
  round in the pass cycle.
- **Safe trick**: Informal term for the first trick, which normally
  contains no penalty points due to the first trick restriction.

## Overview

Hearts is a trick-taking card game for four players. The goal is to
avoid taking penalty points. The game ends when any player reaches or
exceeds 100 points, and the player with the lowest score wins.

## Players and Cards

- 4 players, no partnerships.
- Standard 52-card deck.
- Suits: ♣ Clubs, ♦ Diamonds, ♥ Hearts, ♠ Spades.
- Cards rank Ace (high) down to Two (low) within each suit.
- There is no trump suit.

## Deal

All 52 cards are dealt out one at a time, so each player receives 13
cards.

## Passing

At the start of each round, each player selects three cards from their
hand and passes them face-down to another player. The pass direction
rotates each round in a four-round cycle:

1. Left
2. Right
3. Across
4. Hold (no passing)

A player must select their cards to pass before looking at the cards
they receive.

## Play

The player holding the 2♣ leads it to the first trick. Since there are
no restrictions on which cards may be passed, who holds the 2♣ can only
be determined after passing is complete.

Play proceeds clockwise. Each player must follow the led suit if able.
If a player has no card of the led suit, they may play any card, subject
to the first trick restriction below. The highest card of the led suit
wins the trick, and the winner leads the next trick.

### First Trick Restriction

On the first trick, a player who cannot follow suit may not play a
penalty card unless their hand contains nothing but penalty cards.

### Breaking Hearts

Hearts may not be led until a heart has been played on a previous trick.

*Exception:* A player whose hand contains nothing but hearts may lead a
heart even if hearts have not been broken.

The Q♠ does not break hearts.

## Scoring

At the end of each round, after all tricks have been played, each
player scores penalty points for the cards in the tricks they won.

### Penalty Points

Each heart card taken in tricks is worth 1 penalty point. The Q♠ is
worth 13 penalty points. All other cards carry no penalty. The total
penalty points in each round is always 26.

### Shooting the Moon

If a single player takes all 26 penalty points in a round (all thirteen
hearts and the Q♠), that player receives zero penalty points and every
other player receives 26 penalty points.

### Game End

A round is always played to completion, even if a player's cumulative
score would reach or exceed 100 points before the end of the round.
After scoring, if any player's
cumulative score has reached or exceeded 100 points, the game ends. The
player with the lowest score wins.

## Variants

The following are well-known rule variants that differ from the standard
rules above. They are documented here for future reference; the
`hearts` package does not yet support them.

### Shooter's Choice

When shooting the moon, the shooter may choose either to subtract 26
from their own score or to add 26 to every other player's score. In
both cases the shooter receives zero penalty points for the round. (The
standard rules always add 26 to opponents with no choice.)

Reference: Pagat (treats this as the default rule).

### Queen of Spades Breaks Hearts

Playing the Q♠ counts as breaking hearts, allowing hearts to be led on
subsequent tricks.

Reference: Bicycle Cards, Pagat (variant).

### No First Trick Restriction

Hearts and the Q♠ may be played on the first trick without restriction.
This was the original rule in early versions of Hearts (late 19th
century).

Reference: Pagat (original/variant).

### Omnibus Hearts (Jack of Diamonds)

The J♦ is a bonus card worth −10 points for the player who takes it. To
shoot the moon, a player must still take all thirteen hearts and the
Q♠; the J♦ scoring is applied separately.

Reference: Pagat (variant).

### Spot Hearts

Hearts carry their pip value as penalty points (Two = 2, Three = 3, ...
Ten = 10, Jack = 11, Queen = 12, King = 13, Ace = 14). The Q♠ is worth
25. The game is played to 500 points.

Reference: Pagat (variant).

### Alternative Player Counts

#### 3 Players

Remove the 2♣ from the deck. Deal 17 cards to each player. The player
holding the 3♣ leads it to the first trick. Tricks contain 3 cards.

Pass cycle options:
1. Left, right, hold (three-round cycle)
2. Left, right (two-round cycle, no hold)

#### 5 Players

Remove the 2♣ and 2♦ from the deck. Deal 10 cards to each player. The
player holding the 3♣ leads it to the first trick. Tricks contain 5
cards.

Pass cycle options:
1. Left, right, hold (three-round cycle)
2. Left, right, second left, second right, hold (five-round cycle)

Reference: Pagat, Bicycle Cards.
