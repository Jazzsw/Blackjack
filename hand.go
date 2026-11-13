package main

import ("fmt"
		"strings")

type Hand struct {
	Cards []Card
}

func (h *Hand) Add(c Card) {
	h.Cards = append(h.Cards, c)
}

func (h *Hand) Clear() {
	h.Cards = h.Cards[:0]
}

// Value returns the best total for the hand.
// Count aces as 11, then drop each one to 1 while the total is over 21.
func (h *Hand) Value() int {

    total := 0
    aces := 0

    for _, c := range h.Cards {
        total += c.Value()
        if c.Rank == 1 {
            aces++
        }
    }

    for total > 21 && aces > 0 {
        total -= 10
        aces--
    }

    return total
}

func (h *Hand) IsBust() bool {
	sum := h.sumCards()
	
	if(sum > 21){
		return true
	}

	return false
}

// IsBlackjack is true for exactly two cards totalling 21.
func (h *Hand) IsBlackjack() bool {
	sum := h.sumCards()
	if(sum == 21 && len(h.Cards) == 2){
		return true
	}
	return false
}

// String renders the hand for display, e.g. "A♠ K♥ (21)".
// hideSecond lets us show the dealer's hand with the hole card hidden.
func (h *Hand) String(hideSecond bool) string {
	if len(h.Cards) == 0 {
		return "(empty hand)"
	}

	cardLines := make([][]string, 5)

	for i, card := range h.Cards {
		var rankStr, suitStr string

		if hideSecond && i == 1 {
			rankStr = "?"
			suitStr = "?"
		} else {
			rankStr = getRankStr(card.Rank)
			suitStr = getSuitStr(card.Suit)
		}

		// Ensure spacing is consistent for single/double-digit ranks (like "10")
		topRank := fmt.Sprintf("%-2s", rankStr)
		botRank := fmt.Sprintf("%2s", rankStr)


		lines := []string{
			"┌───────┐",
			fmt.Sprintf("│ %s    │", topRank),
			fmt.Sprintf("│   %s   │", suitStr),
			fmt.Sprintf("│    %s │", botRank),
			"└───────┘",
		}

		for lineIdx, line := range lines {
			cardLines[lineIdx] = append(cardLines[lineIdx], line)
		}
	}

	var sb strings.Builder
	for i := 0; i < 5; i++ {
		sb.WriteString(strings.Join(cardLines[i], "  "))
		sb.WriteString("\n")
	}

	if hideSecond {
		sb.WriteString("Score: ?")
	} else {
		sb.WriteString(fmt.Sprintf("Score: %d", h.Value()))
	}

	return sb.String()
}


func (h *Hand) sumCards() int {
	sum := 0
	for _, card := range h.Cards{
		sum += card.Value()
	}
	return sum
}
