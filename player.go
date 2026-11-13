package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	StartingChips = 100
	readTimeout   = 30 * time.Second
)

type Player struct {
	Name     string
	Conn     net.Conn
	Reader   *bufio.Reader
	Hand     Hand
	Chips    int
	Bet      int
	UIClient bool
}

func NewPlayer(conn net.Conn) *Player {
	return &Player{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
		Chips:  StartingChips,
	}
}

// Send writes a line of text to the player's connection.
func (p *Player) Send(msg string) error {
	_, err := fmt.Fprintln(p.Conn, msg)
	return err
}

func (p *Player) SendState(state []byte) error {
	_, err := fmt.Fprintf(p.Conn, "%s%s\n", tableStatePrefix, state)
	return err
}

// ReadLine reads one trimmed, lowercased line from the player.
func (p *Player) ReadLine() (string, error) {
	if err := p.Conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return "", err
	}

	line, err := p.Reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.TrimSpace(line)), nil
}

// Prompt sends a message and waits for one line from the player.
func (p *Player) Prompt(msg string) (string, error) {
	if err := p.Send(msg); err != nil {
		return "", err
	}
	return p.ReadLine()
}

func (p *Player) Close() error {
	return p.Conn.Close()
}
