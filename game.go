package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The Run goroutine owns all table state. Other goroutines submit changes
// through requests instead of accessing the player slice directly.
type Phase int

const (
	WaitingForPlayers Phase = iota
	Betting
	PlayerTurns
	DealerTurn
	Settling
)

const roundResultPause = 3 * time.Second

type tableRequest struct {
	player *Player
	leave  bool
}

type playerInputResult struct {
	input string
	err   error
}

type Table struct {
	requests           chan tableRequest
	players            []*Player
	dealer             Hand
	deck               *Deck
	phase              Phase
	hideDealerHoleCard bool
}

func NewTable() *Table {
	return &Table{
		requests: make(chan tableRequest, 64),
		deck:     NewDeck(),
	}
}

// Join queues a player to join the table.
func (t *Table) Join(p *Player) {
	t.requests <- tableRequest{player: p}
}

// Leave queues a player to be removed from the table.
func (t *Table) Leave(p *Player) {
	t.requests <- tableRequest{player: p, leave: true}
}

// Broadcast sends a message to every player at the table.
func (t *Table) Broadcast(msg string) {
	for i := 0; i < len(t.players); {
		p := t.players[i]
		if p.UIClient {
			i++
			continue
		}
		if err := p.Send(msg); err != nil {
			t.removePlayer(p, false)
			continue
		}
		i++
	}
	t.broadcastState(msg)
}

// Run is the main game loop. Start it with `go table.Run()`.
func (t *Table) Run() {
	for {
		if len(t.players) == 0 {
			t.phase = WaitingForPlayers
			t.handleRequest(<-t.requests)
			continue
		}

		t.playRound()
		t.processPendingRequests()
	}
}

func (t *Table) handleRequest(request tableRequest) {
	if request.player == nil {
		return
	}
	if request.leave {
		t.removePlayer(request.player, true)
		return
	}

	for _, p := range t.players {
		if p == request.player {
			return
		}
	}
	t.players = append(t.players, request.player)
	t.Broadcast(fmt.Sprintf("%s joined the table.", request.player.Name))
}

func (t *Table) processPendingRequests() {
	for {
		select {
		case request := <-t.requests:
			t.handleRequest(request)
		default:
			return
		}
	}
}

func (t *Table) removePlayer(p *Player, announce bool) {
	for i, player := range t.players {
		if player != p {
			continue
		}

		t.players = append(t.players[:i], t.players[i+1:]...)
		if err := p.Close(); err != nil {
			fmt.Printf("closing connection for %s: %v\n", p.Name, err)
		}
		if announce {
			t.Broadcast(fmt.Sprintf("%s left the table.", p.Name))
		}
		return
	}
}

func (t *Table) playRound() {
	t.deck = NewDeck()
	t.hideDealerHoleCard = false
	t.dealer.Clear()
	for _, p := range t.players {
		p.Hand.Clear()
		p.Bet = 0
	}
	t.phase = Betting
	t.broadcastState("Waiting for bets")
	t.takeBets()
	if len(t.players) == 0 {
		t.broadcastState("Waiting for players")
		return
	}

	t.dealInitialCards()
	t.phase = PlayerTurns
	t.broadcastState("Player turns")
	for _, p := range append([]*Player(nil), t.players...) {
		if t.containsPlayer(p) {
			t.runPlayerTurn(p)
		}
	}

	t.phase = DealerTurn
	t.runDealerTurn()
	t.phase = Settling
	t.broadcastState("Settling bets")
	t.settle()
}

func (t *Table) takeBets() {
	for i := 0; i < len(t.players); {
		p := t.players[i]
		t.broadcastState(fmt.Sprintf("%s is placing a bet", p.Name))
		input, err := t.promptPlayer(p, fmt.Sprintf("%s, enter your bet (1-%d): ", p.Name, p.Chips))
		if err != nil {
			t.removePlayer(p, true)
			continue
		}

		bet, err := strconv.Atoi(input)
		if err != nil || bet < 1 || bet > p.Chips {
			if err := p.Send(fmt.Sprintf("Enter a whole-number bet from 1 to %d.", p.Chips)); err != nil {
				t.removePlayer(p, true)
				continue
			}
			continue
		}
		p.Bet = bet
		p.Chips -= bet
		t.broadcastState(fmt.Sprintf("%s bet %d", p.Name, bet))
		i++
	}
}

func (t *Table) dealInitialCards() {
	t.dealer.Clear()
	for _, p := range t.players {
		p.Hand.Clear()
	}

	for i := 0; i < 2; i++ {
		for _, p := range t.players {
			if p.Bet > 0 {
				p.Hand.Add(t.deck.Draw())
			}
		}
		t.dealer.Add(t.deck.Draw())
	}

	t.hideDealerHoleCard = true
	t.Broadcast(t.tableState(true))
}

func (t *Table) tableState(hideDealerCard bool) string {
	var state strings.Builder
	state.WriteString("Table:\n")
	state.WriteString("Dealer:\n")
	state.WriteString(t.dealer.String(hideDealerCard))
	for _, p := range t.players {
		if p.Bet == 0 {
			continue
		}
		fmt.Fprintf(&state, "\n%s (chips: %d):\n%s", p.Name, p.Chips, p.Hand.String(false))
	}
	return state.String()
}

func (t *Table) runPlayerTurn(p *Player) {
	if p.Hand.IsBlackjack() {
		t.Broadcast(fmt.Sprintf("%s has blackjack.", p.Name))
		return
	}

	for !p.Hand.IsBust() {
		t.broadcastState(fmt.Sprintf("%s's turn", p.Name))
		input, err := t.promptPlayer(p, fmt.Sprintf(
			"%s, your hand is:\n%s\nType hit or stand: ",
			p.Name, p.Hand.String(false),
		))
		if err != nil {
			t.removePlayer(p, true)
			return
		}

		switch input {
		case "hit":
			p.Hand.Add(t.deck.Draw())
			if p.Hand.IsBust() {
				t.Broadcast(fmt.Sprintf("%s busts with %d.", p.Name, p.Hand.Value()))
			} else {
				t.Broadcast(fmt.Sprintf("%s hits.\n%s", p.Name, p.Hand.String(false)))
			}
		case "stand":
			t.Broadcast(fmt.Sprintf("%s stands on %d.", p.Name, p.Hand.Value()))
			return
		default:
			if err := p.Send("Please type hit or stand."); err != nil {
				t.removePlayer(p, true)
				return
			}
		}
	}
}

func (t *Table) runDealerTurn() {
	t.hideDealerHoleCard = false
	t.Broadcast("Dealer's hand:\n" + t.dealer.String(false))
	for t.dealer.Value() < 17 {
		t.dealer.Add(t.deck.Draw())
		t.Broadcast(fmt.Sprintf("Dealer hits.\n%s", t.dealer.String(false)))
	}
	if t.dealer.IsBust() {
		t.Broadcast(fmt.Sprintf("Dealer busts with %d.", t.dealer.Value()))
	} else {
		t.Broadcast(fmt.Sprintf("Dealer stands on %d.", t.dealer.Value()))
	}
}

func (t *Table) settle() {
	dealerBlackjack := t.dealer.IsBlackjack()
	dealerBust := t.dealer.IsBust()
	dealerValue := t.dealer.Value()
	results := make([]string, 0, len(t.players))

	for _, p := range append([]*Player(nil), t.players...) {
		if !t.containsPlayer(p) || p.Bet == 0 {
			continue
		}

		playerValue := p.Hand.Value()
		var result string
		switch {
		case p.Hand.IsBust():
			result = fmt.Sprintf("%s loses %d chips.", p.Name, p.Bet)
		case p.Hand.IsBlackjack() && dealerBlackjack:
			p.Chips += p.Bet
			result = fmt.Sprintf("%s pushes with blackjack.", p.Name)
		case p.Hand.IsBlackjack():
			p.Chips += p.Bet + (p.Bet*3)/2
			result = fmt.Sprintf("%s wins with blackjack! Chips: %d.", p.Name, p.Chips)
		case dealerBust || playerValue > dealerValue:
			p.Chips += p.Bet * 2
			result = fmt.Sprintf("%s wins! Chips: %d.", p.Name, p.Chips)
		case playerValue == dealerValue:
			p.Chips += p.Bet
			result = fmt.Sprintf("%s pushes. Chips: %d.", p.Name, p.Chips)
		default:
			result = fmt.Sprintf("%s loses %d chips.", p.Name, p.Bet)
		}

		t.Broadcast(result)
		results = append(results, result)
		if p.Chips == 0 {
			outOfChips := fmt.Sprintf("%s is out of chips.", p.Name)
			t.Broadcast(outOfChips)
			results = append(results, outOfChips)
			t.removePlayer(p, false)
		}
	}
	if len(results) == 0 {
		results = append(results, "Round complete")
	}
	t.broadcastRoundResults(results)
	time.Sleep(roundResultPause)
}

func (t *Table) containsPlayer(target *Player) bool {
	for _, p := range t.players {
		if p == target {
			return true
		}
	}
	return false
}

func (t *Table) promptPlayer(p *Player, prompt string) (string, error) {
	if err := p.Send(prompt); err != nil {
		return "", err
	}

	result := make(chan playerInputResult, 1)
	go func() {
		input, err := p.ReadLine()
		result <- playerInputResult{input: input, err: err}
	}()

	for {
		select {
		case input := <-result:
			return input.input, input.err
		case request := <-t.requests:
			t.handleRequest(request)
		}
	}
}

func (t *Table) broadcastState(status string) {
	if newline := strings.IndexByte(status, '\n'); newline >= 0 {
		status = status[:newline]
	}
	if status == "Table:" {
		status = "Cards dealt"
	}
	t.sendTableState(status)
}

func (t *Table) broadcastRoundResults(results []string) {
	t.sendTableState(strings.Join(results, "\n"))
}

func (t *Table) sendTableState(status string) {
	state := GameUIState{
		Dealer:             t.dealer,
		HideDealerHoleCard: t.hideDealerHoleCard,
		Status:             status,
		Players:            make([]GameUIPlayer, 0, len(t.players)),
	}
	for _, player := range t.players {
		state.Players = append(state.Players, GameUIPlayer{
			Name:  player.Name,
			Hand:  player.Hand,
			Chips: player.Chips,
			Bet:   player.Bet,
		})
	}
	payload, err := json.Marshal(cloneGameUIState(state))
	if err != nil {
		fmt.Printf("encoding table state: %v\n", err)
		return
	}

	for i := 0; i < len(t.players); {
		player := t.players[i]
		if !player.UIClient {
			i++
			continue
		}
		if err := player.SendState(payload); err != nil {
			t.removePlayer(player, false)
			continue
		}
		i++
	}
}
