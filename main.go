package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"syscall"
)

// Entry point for the server and Bubble Tea client.
func main() {
	client := flag.Bool("client", false, "run the Bubble Tea client")
	serverOnly := flag.Bool("server-only", false, "run the server without launching a client")
	addr := flag.String("addr", "localhost:4000", "server address for the client")
	listen := flag.String("listen", ":4000", "address for the server to listen on")
	flag.Parse()

	if *client {
		if err := RunBlackjackClient(*addr); err != nil {
			log.Fatal(err)
		}
		return
	}

	srv := NewServer(*listen)
	if *serverOnly {
		log.Printf("blackjack server listening on %s", *listen)
		if err := srv.Start(); err != nil {
			log.Fatal(err)
		}
		return
	}

	listener, err := srv.listen()
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			clientAddr, addrErr := clientAddressForListen(*listen)
			if addrErr != nil {
				log.Fatal(addrErr)
			}
			log.Printf("server already listening on %s; joining it as a client", *listen)
			if err := RunBlackjackClient(clientAddr); err != nil {
				log.Fatal(err)
			}
			return
		}
		log.Fatal(err)
	}
	clientAddr, err := localClientAddress(listener.Addr())
	if err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			log.Printf("closing blackjack listener: %v", closeErr)
		}
		log.Fatal(err)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.serve(listener)
	}()
	log.Printf("blackjack server listening on %s", listener.Addr())
	if err := RunBlackjackClient(clientAddr); err != nil {
		log.Fatal(err)
	}

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
	default:
	}
}

func localClientAddress(addr net.Addr) (string, error) {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return "", fmt.Errorf("expected a TCP listener address, got %T", addr)
	}

	host := tcpAddr.IP.String()
	if tcpAddr.IP == nil || tcpAddr.IP.IsUnspecified() {
		if tcpAddr.IP != nil && tcpAddr.IP.To4() != nil {
			host = "127.0.0.1"
		} else {
			host = "localhost"
		}
	}
	return net.JoinHostPort(host, strconv.Itoa(tcpAddr.Port)), nil
}

func clientAddressForListen(address string) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return "", err
	}
	return localClientAddress(tcpAddr)
}
