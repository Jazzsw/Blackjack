package main

import (
	"strconv"
	"math/rand"
	"fmt"
)

type Suit int

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

// Rank: 1 = Ace, 2-10 = number cards, 11 = J, 12 = Q, 13 = K
type Card struct {
	Rank int
	Suit Suit
}

func (c Card) String() string {
	var rankStr string = ""
	var symbol string = ""

	switch(c.Rank){
	case(1):
		rankStr = "A"
	case(11):
		rankStr = "J"
	case(12):
		rankStr = "Q"
	case(13):
		rankStr = "K"
	default:
		rankStr = strconv.Itoa((c.Rank))
	}
	
	switch(c.Suit){
	case(Spades):
		symbol = "♠"
	case(Diamonds):
		symbol = "♦"
	case(Hearts):
		symbol = "♥"
	case(Clubs):
		symbol = "♣"
	}

	return rankStr + symbol
}

func getRankStr(rank int) string {
	switch rank {
	case 1:
		return "A"
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	default:
		return fmt.Sprintf("%d", rank)
	}
}

func getSuitStr(suit Suit) string {
	switch suit {
	case Spades:
		return "♠"
	case Diamonds:
		return "♦"
	case Hearts:
		return "♥"
	case Clubs:
		return "♣"
	default:
		return "?"
	}
}


// Value returns the blackjack value of a single card.
// Ace = 11 (Hand.Value handles downgrading aces to 1 when needed).
func (c Card) Value() int {
	if(c.Rank == 1){
		return 11
	}else if(c.Rank > 10){
		return 10
	}else{
		return c.Rank
	}
}

type Deck struct {
	cards []Card
}

// NewDeck builds a fresh, shuffled 52-card deck.
func NewDeck() *Deck {

	new := Deck{}

	for s := Spades; s <= Clubs; s++ {
		for r := 1; r <= 13; r++ {
			new.cards = append(new.cards, Card{Rank: r, Suit: s})
		}
	}
	new.Shuffle()

	return &new
}

// Shuffle randomizes the order of the remaining cards.
func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

// Draw removes and returns the top card.
func (d *Deck) Draw() Card {
    if len(d.cards) == 0 {
        *d = *NewDeck()
    }

    card := d.cards[0]
    d.cards = d.cards[1:]
    return card
}
