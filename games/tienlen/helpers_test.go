package tienlen

import "github.com/jrgoldfinemiddleton/cardcore"

// Rank aliases for test readability.
const (
	rAce   = cardcore.Ace
	rTwo   = cardcore.Two
	rThree = cardcore.Three
	rFour  = cardcore.Four
	rFive  = cardcore.Five
	rSix   = cardcore.Six
	rSeven = cardcore.Seven
	rEight = cardcore.Eight
	rNine  = cardcore.Nine
	rTen   = cardcore.Ten
	rJack  = cardcore.Jack
	rQueen = cardcore.Queen
	rKing  = cardcore.King
)

// Suit aliases for test readability.
const (
	sClubs    = cardcore.Clubs
	sDiamonds = cardcore.Diamonds
	sHearts   = cardcore.Hearts
	sSpades   = cardcore.Spades
)

// c is a shorthand constructor for [cardcore.Card].
func c(r cardcore.Rank, s cardcore.Suit) cardcore.Card {
	return cardcore.Card{Rank: r, Suit: s}
}
