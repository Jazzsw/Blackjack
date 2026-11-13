package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type GameUIPlayer struct {
	Name  string
	Hand  Hand
	Chips int
	Bet   int
}

type GameUIState struct {
	Dealer             Hand
	Players            []GameUIPlayer
	HideDealerHoleCard bool
	Status             string
}

type GameUI struct {
	state GameUIState
	width int
}

func NewGameUI(state GameUIState) GameUI {
	return GameUI{state: cloneGameUIState(state)}
}

func (m GameUI) View() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	if width < 40 {
		width = 40
	}

	innerWidth := width - 6
	lines := []string{
		centerUI("BLACKJACK", innerWidth),
		"",
		centerUI("DEALER", innerWidth),
	}
	lines = append(lines, centerLines(m.state.Dealer.String(m.state.HideDealerHoleCard), innerWidth)...)
	lines = append(lines, "", centerUI(strings.Repeat("─", innerWidth-4), innerWidth), "")

	for _, row := range playerSeatRows(m.state.Players, innerWidth) {
		for _, line := range strings.Split(row, "\n") {
			lines = append(lines, centerUI(line, innerWidth))
		}
	}
	if len(m.state.Players) == 0 {
		lines = append(lines, centerUI("Waiting for players", innerWidth))
	}
	if m.state.Status != "" {
		status := lipgloss.NewStyle().Width(innerWidth).Render(m.state.Status)
		lines = append(lines, "")
		lines = append(lines, centerLines(status, innerWidth)...)
	}

	felt := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("15")).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("0")).
		Padding(1, 2).
		Width(innerWidth).
		Render(strings.Join(lines, "\n"))
	return felt
}

func cloneGameUIState(state GameUIState) GameUIState {
	state.Dealer.Cards = append([]Card(nil), state.Dealer.Cards...)
	players := make([]GameUIPlayer, len(state.Players))
	copy(players, state.Players)
	for i := range players {
		players[i].Hand.Cards = append([]Card(nil), state.Players[i].Hand.Cards...)
	}
	state.Players = players
	return state
}

func playerSeatRows(players []GameUIPlayer, width int) []string {
	var rows []string
	var seats []string
	rowWidth := 0

	for _, player := range players {
		seat := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("15")).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("0")).
			Padding(0, 1).
			Render(fmt.Sprintf("%s\nchips: %d  ·  bet: %d\n%s",
				player.Name, player.Chips, player.Bet, player.Hand.String(false)))
		seatWidth := lipgloss.Width(seat)
		gap := 4
		if len(seats) > 0 && rowWidth+gap+seatWidth > width {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, seats...))
			seats = nil
			rowWidth = 0
		}
		if len(seats) > 0 {
			rowWidth += gap
			seat = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("0")).
				PaddingLeft(gap).
				Render(seat)
		}
		seats = append(seats, seat)
		rowWidth += seatWidth
	}
	if len(seats) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, seats...))
	}
	return rows
}

func centerLines(value string, width int) []string {
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = centerUI(lines[i], width)
	}
	return lines
}

func centerUI(value string, width int) string {
	padding := width - lipgloss.Width(value)
	if padding <= 0 {
		return value
	}
	left := padding / 2
	return strings.Repeat(" ", left) + value + strings.Repeat(" ", padding-left)
}
