package main

import "context"

type Game struct {
	Code        string
	players     map[*Player]bool
	server      *RPSServer
	playerJoin  chan *Player
	playerLeave chan *Player
}

func NewGame(server *RPSServer) *Game {
	return &Game{server: server, Code: "", players: make(map[*Player]bool), playerJoin: make(chan *Player), playerLeave: make(chan *Player)}
}

func (g *Game) run(ctx context.Context) error {
	for {
		select {
		case p := <-g.playerJoin:
			g.players[p] = true
		case p := <-g.playerLeave:
			delete(g.players, p)
		case <-ctx.Done():
			g.server.removeGame <- g
			return ctx.Err()
		}
	}
}
