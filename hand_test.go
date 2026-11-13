package main

import (
	"strings"
	"testing"
)

func TestHandStringHidesDealerScoreWithHoleCard(t *testing.T) {
	hand := Hand{Cards: []Card{
		{Rank: 10, Suit: Spades},
		{Rank: 1, Suit: Hearts},
	}}

	got := hand.String(true)
	if !strings.Contains(got, "Score: ?") {
		t.Fatalf("hidden hand should have an unknown score, got %q", got)
	}
	if strings.Contains(got, "Score: 21") {
		t.Fatalf("hidden hand should not reveal its total, got %q", got)
	}
}

func TestHandStringShowsScoreWhenHoleCardIsRevealed(t *testing.T) {
	hand := Hand{Cards: []Card{
		{Rank: 10, Suit: Spades},
		{Rank: 1, Suit: Hearts},
	}}

	got := hand.String(false)
	if !strings.Contains(got, "Score: 21") {
		t.Fatalf("visible hand should show its total, got %q", got)
	}
}
