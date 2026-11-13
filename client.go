package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	clientHelloPrefix = "@@hello "
	tableStatePrefix  = "@@state "
)

type clientHello struct {
	Username string `json:"username"`
}

type clientLineMsg struct {
	line string
	err  error
}

type blackjackClient struct {
	conn        net.Conn
	reader      *bufio.Reader
	state       GameUIState
	width       int
	input       string
	status      string
	loginPrompt bool
	helloSent   bool
	promptReady bool
}

func RunBlackjackClient(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	model := blackjackClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
		state:  GameUIState{},
	}
	_, err = tea.NewProgram(model).Run()
	return err
}

func (m blackjackClient) Init() tea.Cmd {
	return readClientLine(m.reader)
}

func (m blackjackClient) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case clientLineMsg:
		if msg.line != "" {
			if strings.HasPrefix(msg.line, tableStatePrefix) {
				var state GameUIState
				if err := json.Unmarshal([]byte(strings.TrimPrefix(msg.line, tableStatePrefix)), &state); err != nil {
					m.status = fmt.Sprintf("Invalid table update: %v", err)
					return m, tea.Quit
				}
				m.state = cloneGameUIState(state)
				if state.Status != "" {
					m.status = state.Status
				}
			} else {
				m.status = msg.line
				lowerLine := strings.ToLower(msg.line)
				if strings.Contains(lowerLine, "enter username:") {
					m.loginPrompt = true
				}
				if strings.Contains(lowerLine, "enter your bet") ||
					strings.Contains(lowerLine, "type hit or stand:") {
					m.promptReady = true
				}
			}
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("Connection closed: %v", msg.err)
			return m, nil
		}
		return m, readClientLine(m.reader)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if (!m.helloSent && m.loginPrompt) || (m.helloSent && m.promptReady) {
				m.submitInput()
			} else {
				m.status = "Wait for a server prompt."
			}
		case tea.KeyBackspace, tea.KeyDelete:
			runes := []rune(m.input)
			if len(runes) > 0 {
				m.input = string(runes[:len(runes)-1])
			}
		case tea.KeySpace:
			m.input += " "
		case tea.KeyRunes:
			m.input += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *blackjackClient) submitInput() {
	input := strings.TrimSpace(m.input)
	if input == "" {
		return
	}

	line := input
	if !m.helloSent {
		hello, err := json.Marshal(clientHello{Username: input})
		if err != nil {
			m.status = fmt.Sprintf("Encoding login: %v", err)
			return
		}
		line = clientHelloPrefix + string(hello)
		m.helloSent = true
	}

	if _, err := fmt.Fprintln(m.conn, line); err != nil {
		m.status = fmt.Sprintf("Sending input: %v", err)
		return
	}
	m.input = ""
	if m.helloSent {
		m.promptReady = false
	}
}

func (m blackjackClient) View() string {
	board := NewGameUI(m.state)
	board.width = m.width
	status := m.status
	if status == "" {
		status = "Connecting..."
	}

	content := board.View()
	if status != m.state.Status {
		content += "\n\n" + status
	}
	content += "\n\n> " + m.input + "\n\nEnter submits · Ctrl+C quits"
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("0")).
		Render(content)
}

func readClientLine(reader *bufio.Reader) tea.Cmd {
	return func() tea.Msg {
		line, err := reader.ReadString('\n')
		return clientLineMsg{
			line: strings.TrimRight(line, "\r\n"),
			err:  err,
		}
	}
}
