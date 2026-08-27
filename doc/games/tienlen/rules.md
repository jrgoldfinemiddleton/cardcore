# Tiến Lên: Game Rules

Also known as: Thirteen, Tien Len, Tiến Lên Miền Nam ("Southern
Tiến Lên"; miền Nam means "Southern region"), Killer.

This document defines the rules of Tiến Lên as implemented by the
`tienlen` package. The body describes the canonical Southern game
(miền nam); the [Variants](#variants) section describes the Killer
variant as deltas from the baseline.

Primary references:
- [vi.wikipedia.org: Bài Tiến lên](https://vi.wikipedia.org/wiki/Bài_Tiến_lên)
- [Pagat: Thirteen](https://www.pagat.com/climbing/thirteen.html) (John McLeod)
- [en.wikipedia.org: Tiến lên](https://en.wikipedia.org/wiki/Tiến_lên)
- [Board Game Arena: Tiến Lên ruleset](https://boardgamearena.com/gamepanel?game=tienlen)

## Definitions

Unless otherwise noted, definitions describe the baseline Southern
game; terms that differ under Killer are defined in
[Variants](#variants).

- **Tiến Lên**: the game's name, "advance" or "press forward".
  **Miền Bắc** ("Northern region"): the Northern Vietnamese variant.
- **Pig** (heo): a 2. The **red pigs** (heo đỏ) are the 2♥ and 2♦;
  the **black pigs** (heo đen) are the 2♠ and 2♣.
- **Hand**: all play from one deal through settlement. A match
  consists of multiple hands.
- **Combination**: a legal set of cards played together (single, pair,
  triple, four-of-a-kind, straight, or pair sequence).
- **Bomb-class combination** (hàng, "goods"): a four-of-a-kind or a
  pair sequence.
- **Chop** (chặt): a bomb-class combination played to beat a 2, a pair
  of 2s, or another bomb-class combination. A **stacked chop**
  (chặt chồng) is a chop played on top of another chop; a
  **free chop** (chặt tự do) is one played out of turn.
- **Pass**: to decline to beat the current combination on one's turn.
- **Pile**: one contested combination plus the plays made to beat it. A
  pile ends when every other player has passed on the latest play.
- **Lead**: to play the first combination of a new pile.
- **Lockout**: the state of a player who has passed: barred from
  further play on the current pile.
- **Finisher**: a player shedding their final card or combination.
- **Go out**: to completely empty one's hand.
- **Place**: a player's finishing position in a hand (first through
  last).
- **Lead inheritance** (hưởng sái): when a finisher's final play
  stands unbeaten, the next player in turn order inherits the lead.
- **Rotten cards** (thối): scoring-relevant cards still in hand when
  the hand ends. Unplayed 2s are **rotten 2s** (thối heo); unplayed
  bomb-class combinations are **rotten bomb-class combinations**
  (thối hàng).
- **Blank** (cóng): finishing a hand having played zero cards.
- **Dragon straight** (sảnh rồng): straight that runs twelve
  consecutive cards from 3 to A.
- **Auto-win/Instant win** (tới trắng): a win declared at the deal,
  before any card is played. Also known as "winning white".
- **Stake**: the betting unit players agree on before play. All
  payments are multiples or fractions of one stake.
- Suit names: **Hearts** (Cơ), **Diamonds** (Rô), **Clubs** (Chuồn),
  **Spades** (Bích).

## Overview

Tiến Lên is a Vietnamese climbing (shedding) card game for two to four
players, with no partnerships. Each player tries to be the first to get
rid of all thirteen cards by beating the combinations played before them.
Play continues until only one player still holds cards in their hand.

## Players and Cards

- 2 to 4 players, no partnerships.
- Standard 52-card deck.
- Rank order: 2 (highest), A, K, Q, J, 10, 9, 8, 7, 6, 5, 4, 3 (lowest).
- Suit order within a rank: ♥ (highest), ♦, ♣, ♠ (lowest).
- Rank dominates suit: for example, 8♠ beats 7♥.
- 2♥ is the highest card in the deck and 3♠ the lowest.

## Deal

Thirteen cards are dealt to each player in counterclockwise order. In
two- and three-player games the remaining cards stay in the deck, out
of play.

## Instant Wins

A player whose dealt hand matches one of the following wins the hand
instantly at the deal (tới trắng), before any card is played. The win
must be declared immediately after the deal; failing to declare before
the first play forfeits it.

Canonical set (any hand of the match):

- **Dragon straight** (sảnh rồng): twelve consecutive cards, 3 to A.
- **Four 2s** (tứ quý heo).
- **Five consecutive pairs** (5 đôi thông).
- **Any six pairs** (6 đôi bất kỳ): they need not be consecutive.
- **Twelve cards of the same color** (12 lá cùng màu): all red or all
  black.

Opening-hand extras (first hand of the match only):

- **Four 3s** (tứ quý 3).
- **Three consecutive pairs including the 3♠** (3 đôi thông).

If two players hold instant-win hands in the same deal, only the
higher-ranked instant win wins; the other holder is an ordinary loser.
Priority follows the payout order: dragon straight, then twelve
same-color cards, then five consecutive pairs, then four 2s, then any
six pairs; the opening-hand extras rank below all of these, four 3s
above three consecutive pairs including the 3♠. Between two instant
wins of the same kind, compare the highest card by rank, then suit;
exact ties are impossible in a single deck.

## Play

### Direction of Play

Play proceeds counterclockwise: the next player is the one to the
current player's right.

### The First Lead

In the first hand, the holder of the 3♠ leads, and the first play
must include the 3♠: as a single, or within any legal combination
containing it (for example a pair of 3s or the straight 3♠-4-5). If no
one holds the 3♠ (possible in two- and three-player games), the holder
of the lowest card in play leads, including that card. In subsequent
hands, the winner of the previous hand leads, with a free choice of
combination.

### Combinations

- **Single**: any one card. The lowest single is the 3♠ and the
  highest is the 2♥.
- **Pair** (đôi): two cards of the same rank, such as 7♣-7♦.
- **Triple**: three cards of the same rank, such as 5♣-5♦-5♥.
- **Four-of-a-kind** (tứ quý): four cards of the same rank, such as
  9♠-9♣-9♦-9♥.
- **Straight** (sảnh): three or more cards of consecutive rank; the
  suits need not match; a 2 cannot be included. The lowest three-card
  straight is 3-4-5 and the highest is Q-K-A.
- **Pair sequence** (đôi thông): three or more pairs of consecutive
  rank, such as 3-3-4-4-5-5 or 7-7-8-8-9-9-10-10. 2s may not be
  included.

A combination can be beaten only by a higher combination of the same
type and length, and a single only by a higher single. A player
cannot, for example, beat a pair with a triple, or a four-card
straight with a five-card straight, or vice versa. To determine which
of two combinations of the same type is higher, compare the highest
card's rank first, then its suit. For example, 7♠-7♥ beats 7♣-7♦
because the 7♥ beats the 7♦. In straights, 8♠-9♠-10♦ beats
8♥-9♥-10♣ because the 10♦ beats the 10♣. The only exceptions to the
type-match requirement are the chops described below.

### Chops

The bomb-class combinations (hàng) may chop (chặt) 2s and lower
bomb-class combinations. The hierarchy, strongest first:

1. Four consecutive pairs (4 đôi thông)
2. Four-of-a-kind (tứ quý)
3. Three consecutive pairs (3 đôi thông)

- Three consecutive pairs chop a single 2.
- Four-of-a-kind chops a single 2, a pair of 2s (đôi heo), and any
  three consecutive pairs.
- Four consecutive pairs chop everything below them: any single 2, any
  pair of 2s, any four-of-a-kind, and any three consecutive pairs.
- Within the same type and length, the higher bomb-class combination
  chops the lower, comparing highest cards as usual.
- Chops are the only exception to same-type, same-length matching, and
  a player may chop even after passing on the current pile.
- Four consecutive pairs may also be played out of turn (a free chop,
  chặt tự do): it need not be the player's turn, and a locked-out
  player may free-chop. A locked-out player who free-chops returns to
  play for the rest of the pile. After a free chop, the next player in
  turn order after the chopper (to the chopper's right) responds; the
  out-of-turn privilege is for chopping only and may not be used to
  lead a new pile. A free chop may itself be free-chopped by a higher
  four consecutive pairs, out of turn; where two players attempt to
  free-chop simultaneously, the first played stands.
- Longer pair sequences (five or more consecutive pairs) appear only
  as instant-win material; they may not be played as chops. No
  combination beats triple 2s: chops target only single 2s, pairs of 2s,
  and bomb-class combinations.

### Passing, Lockout, and Pile Resolution

- On their turn, a player must either beat the current combination or
  pass.
- A player who passes is locked out of the current pile: they may not
  play on it again, except to chop (see above).
- When every other player has passed on the latest play, the pile
  ends. The player of the unbeaten combination takes the pile (the
  cards are set aside) and leads any single or combination to start
  the next pile.
- Going out and lead inheritance (hưởng sái): when a player sheds
  their last cards, their final combination remains live, and players
  still active in the pile may still beat it; a locked-out player may
  only chop it. If no one does, the next player in turn
  order (the finisher's right) inherits the lead and starts the next
  pile freely. If a finisher's final combination is chopped, no chop
  payment is made: the finisher's place is already fixed, and the chop
  earns only continuation of play. A further chop beating such a
  payment-free chop earns payment normally, as the first chop of a new
  chain.
- The hand ends when only one player still holds cards; the order in
  which players went out fixes the places (first through last).

## Scoring and Settlement

Players agree on a stake before play. Chop payments and penalties are
tallied during play and settled at the end of the hand. The place
payment, chop payments, and penalties are cumulative: each transfer
settles in full, independently of the others.

Places: the last-place player pays the winner 1 stake; other players
do not transfer.

Chop payments: a chopped black 2 is ½ stake, a chopped red 2 is 1
stake, three consecutive pairs 1.5 stakes, four-of-a-kind 2 stakes,
and four consecutive pairs 2.5 stakes — the same rates as the
corresponding rotten penalties (listed below); the chopped player pays
the chopping player. When a chop is itself beaten by a later chop (a
stacked chop, chặt chồng), the payments accumulate, and the last
chopped player pays the accumulated total; earlier chops in the chain
are not settled individually.

Penalties: when a hand ends in play, the player still holding cards
pays the winner for the following:

- **Rotten 2s** (thối heo): 1 stake per red 2 (2♥, 2♦); ½ stake
  per black 2 (2♠, 2♣).
- **Rotten bomb-class combinations** (thối hàng): three consecutive
  pairs, 1.5 stakes; four-of-a-kind, 2 stakes; four consecutive
  pairs, 2.5 stakes; five consecutive pairs, 3 stakes; six
  consecutive pairs, 3.5 stakes.
- **Blank** (cóng): a player who played zero cards all hand pays 2
  stakes, plus the value of every 2 and bomb-class combination
  remaining in their hand, counted at the rates above.

A hand that ends by instant win settles differently: each other player
pays the declarer a fixed amount for the declared hand — dragon
straight, 9 stakes; twelve cards of the same color, 5 stakes; five
consecutive pairs, 4 stakes; four 2s, 3 stakes; any six pairs, 2
stakes; an opening-hand extra (four 3s, or three consecutive pairs
including the 3♠), 2 stakes. Rotten and blank penalties are not
counted on an instant win.

All payments net to zero.

### Game End

A hand ends when one player remains with cards, or immediately upon a
declared instant win. A match consists of multiple hands with a running
tally; match length is left to the players' agreement.

## Variants

### Killer

Killer is a variant of Tiến Lên popularized by Vietnamese-Americans
in San Diego, California.

Except where this section says otherwise, players, deck, deal,
ranking, combinations, comparison, and pile resolution are as in the
baseline game.

#### Definitions

- **Killer**: four cards of the same rank (a four-of-a-kind). To "kill"
  is to play a Killer on another player's card(s).
- **Bomb**: three or more consecutive pairs (a pair sequence). To "bomb"
  is to play a Bomb on another player's card(s).
- **The three**: the 3♠.
- **Out of the blue**: when a Bomb or Killer is played to open a pile
  rather than to beat the current combination; it earns no payment.

#### Deals

Thirteen cards are dealt to each player in clockwise order.

#### Direction of Play

Clockwise: the next player is the one to the current player's left.

#### Start of Play

The player holding the lowest single card (usually the 3♠) starts, and
may play that card as a single or within any legal combination. Every
hand starts this way.

#### Lockout and Self-Beat

Unlike in Tiến Lên, a locked-out player may not chop. The non-passer
keeps self-beating — including chopping their own 2 or pair of 2s —
and when their play could not have beaten their previous one (whether
by ordinary matching or the chop exceptions), that play starts a new
pile and all other players are unlocked. Even when a player goes out,
any remaining player who was locked out of that pile may not jump back
in to the pile, which continues unless no other player opts to beat
the finishing combo.

#### Chop Exceptions

Killer defines the following exceptions to same-type
matching:

- A Killer chops any single 2 or pair of 2s (but no other single or
  pair, such as an ace or a pair of kings). A Killer also chops any
  lower Killer, any six-card Bomb, or any eight-card Bomb.
- A six-card Bomb (such as 7-7-8-8-9-9) chops any single 2 (but not
  any other single). A six-card Bomb also chops a lower six-card
  Bomb.
- An eight-card Bomb (such as 5-5-6-6-7-7-8-8) chops any single 2
  or pair of 2s (but not any other single or pair). An eight-card
  Bomb also chops any lower eight-card Bomb, any six-card Bomb, or
  any Killer.
- A ten-card Bomb chops any single 2 or pair of 2s (but not any
  other single or pair). A ten-card Bomb also chops any lower
  ten-card Bomb, any eight-card Bomb, any six-card Bomb, or any
  Killer.
- A twelve-card Bomb chops any single 2 or pair of 2s (but not any
  other single or pair). A twelve-card Bomb also chops any lower
  twelve-card Bomb, any ten-, eight-, or six-card Bomb, or any Killer.
  A twelve-card Bomb not declared as an automatic win before play may
  still be thrown as a chop during the hand. With thirteen-card hands,
  a twelve-card Bomb is the longest possible Bomb.

No combination beats triple 2s under any circumstance.

Stacked chops: the Killer and the eight-card Bomb are the only two
bomb-class combinations that can beat each other in both directions;
when both directions are legal, the most recently played chop wins.

Free chops are not allowed under any circumstance.

#### Automatic Wins

A player holding one of the following may throw their hand down face
up to claim an automatic win. The hand does not end: play continues
for the remaining places (see Game End).

- A Killer of 2s.
- Any six pairs.

If more than one player holds an automatic win, the one with the
higher top pair wins — comparing rank first, then suit; exact ties are
impossible in a single deck. A Killer of 2s overrides any other's
auto-win.

*Note: A player may opt not to use their auto win. They may wish to
instead attempt a chop to increase their winnings.*

#### Scoring and Settlement

All amounts are multiples of the agreed stake:

- The loser (last place) pays the winner 1 stake. Other players do not
  transfer.
- If a Bomb or Killer beats anything previously played (thus, not
  played "out of the blue"), the player who was bombed or killed pays
  the chopping player 1 stake.
- Within a single pile, each successive chop in a chain raises the
  payment by 1 stake and cancels every earlier chop debt in that
  chain: the first chop is 1 stake, the second 2, the third 3, and so
  on; only the last chopped player pays, settling with the last
  chopper. A player may therefore free another from debt (debts are
  forgiven transitively). Chops in different piles settle
  independently — a chop opening a new pile is worth 1 stake again,
  and earlier piles' debts stand.
- Bombing or killing a finisher's final combination earns nothing: a
  player who has gone out is protected. Such a chop earns only
  continuation of play, and a further chop beating it earns payment
  normally, as the first chop of a new chain.
- There are no penalties for any other scenarios, including cards
  left in any player's hand.

The place payment and all chop payments are cumulative: each transfer
settles in full, independently of the others.

Because later chops can erase earlier debts, chop payments are tallied
and settled at the end of the hand.

#### Game End

The first player to get rid of all thirteen cards wins the hand. Play
continues after the winner finishes so that last place (who owes the
winner) is determined, and an automatic win does not end the match
early. When an automatic win is declared, the declarer takes the next
available place and leaves the hand — first place, or second if another
player has already declared; between two holders, priority follows the
tie-break above. Their cards are set aside unbeaten — there is no final
combination to beat — and play continues among the remaining players as
if the declarer had never been dealt in: the player holding the lowest
card still in any player's hand leads to open the first pile.