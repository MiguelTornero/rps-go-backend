package main

import (
	"context"
	"log"
	"strconv"
)

type Game struct {
	Code           string
	players        map[*Player]bool
	server         *RPSServer
	playerJoin     chan *Player
	playerLeave    chan *Player
	playerNum      int // used for names
	playerQuantity int
}

func NewGame(server *RPSServer) *Game {
	return &Game{server: server, Code: "", players: make(map[*Player]bool), playerJoin: make(chan *Player), playerLeave: make(chan *Player)}
}

func (g *Game) run(ctx context.Context) error {
	for {
		select {
		case p := <-g.playerJoin:
			if _, ok := g.players[p]; ok {
				break
			}
			g.players[p] = true
			g.playerNum++
			g.playerQuantity++
			p.Name = "Player " + strconv.Itoa(g.playerNum)
			log.Println("player", p.Name, "connected to game", g.Code)
		case p := <-g.playerLeave:
			if _, ok := g.players[p]; !ok {
				break
			}

			g.playerQuantity--
			delete(g.players, p)
			log.Println("player", p.Name, "left game", g.Code)
		case <-ctx.Done():
			g.server.removeGame <- g
			return ctx.Err()
		}
	}
}
