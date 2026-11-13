# Blackjack
A simple multiplayer terminal blackjack game, designed in Go.

Run `go run .` to start the server and its Bubble Tea client together. Running it again automatically joins the existing local server as another client. Use `go run . -server-only` to start just the server, or `go run . -client -addr localhost:4000` to connect to an existing server. Plain-text clients can connect with `nc localhost 4000`.