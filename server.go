package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type Server struct {
	addr  string
	table *Table
}

func NewServer(addr string) *Server {
	server := Server{}

	server.addr = addr
	server.table = NewTable()

	return &server
}

// Start listens for connections and blocks forever.
func (s *Server) Start() error {
	ln, err := s.listen()
	if err != nil {
		return err
	}
	return s.serve(ln)
}

func (s *Server) listen() (net.Listener, error) {
	return net.Listen("tcp", s.addr)
}

func (s *Server) serve(ln net.Listener) error {
	defer ln.Close()

	go s.table.Run()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// handleConn runs in its own goroutine per connection.
func (s *Server) handleConn(conn net.Conn) {
	p := NewPlayer(conn)
	username, err := p.Prompt("Enter Username: ")
	if err != nil {
		if closeErr := p.Close(); closeErr != nil {
			fmt.Printf("closing connection: %v\n", closeErr)
		}
		return
	}

	if strings.HasPrefix(username, clientHelloPrefix) {
		var hello clientHello
		if err := json.Unmarshal([]byte(strings.TrimPrefix(username, clientHelloPrefix)), &hello); err != nil {
			if sendErr := p.Send("Invalid UI client handshake."); sendErr != nil {
				fmt.Printf("sending handshake error: %v\n", sendErr)
			}
			if closeErr := p.Close(); closeErr != nil {
				fmt.Printf("closing connection: %v\n", closeErr)
			}
			return
		}
		username = hello.Username
		p.UIClient = true
	}
	if username == "" {
		if closeErr := p.Close(); closeErr != nil {
			fmt.Printf("closing connection: %v\n", closeErr)
		}
		return
	}

	p.Name = username
	if err := p.Send(fmt.Sprintf("Welcome %s, Good Luck!", username)); err != nil {
		if closeErr := p.Close(); closeErr != nil {
			fmt.Printf("closing connection for %s: %v\n", username, closeErr)
		}
		return
	}
	s.table.Join(p)
}
