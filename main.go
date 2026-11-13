package main

import "log"

// Entry point. connect using `nc localhost 4000`.
func main() {
	srv := NewServer(":4000")
	log.Println("blackjack server listening on :4000")
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
