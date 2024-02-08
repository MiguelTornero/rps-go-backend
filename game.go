package main

import (
	"context"
	"errors"
	"fmt"
	"log"
)

type Game struct {
	Code        string
	players     map[*Player]bool
	player1     *Player
	player2     *Player
	server      *RPSServer
	playerJoin  chan *Player
	playerLeave chan *Player
	full        bool
	playerMsg   chan PlayerMsg
}

func NewGame(server *RPSServer) *Game {
	return &Game{server: server, Code: "", players: make(map[*Player]bool), playerJoin: make(chan *Player), playerLeave: make(chan *Player), playerMsg: make(chan PlayerMsg)}
}

func (g *Game) broadcast(msg []byte) {
	if g.player1 != nil {
		g.player1.msgChan <- msg
	}
	if g.player2 != nil {
		g.player2.msgChan <- msg
	}
}

func (g *Game) shutdown(err error) {
	log.Println("shutting down game", g.Code)
	g.server.removeGame <- g
	if g.player1 != nil {
		g.player1.shutdownChan <- err
	}
	if g.player2 != nil {
		g.player2.shutdownChan <- err
	}
}

func (g *Game) run(ctx context.Context) error {
	for {
		select {
		case p := <-g.playerJoin:
			if g.player1 == nil {
				p.Name = "Player 1"
				p.number = 1
				g.player1 = p
			} else if g.player2 == nil {
				p.Name = "Player 2"
				p.number = 2
				g.player2 = p
				g.full = true
			} else {
				log.Println("player tried to join full game")
				break
			}
			log.Println("player", p.Name, "connected to game", g.Code)
			g.broadcast([]byte(p.Name + " joined"))
		case p := <-g.playerLeave:
			err := errors.New(p.Name + " disconnected")
			g.shutdown(err)
			return err
		case msg := <-g.playerMsg:
			if !g.full {
				break
			}
			fmt.Println(msg.player.Name, string(msg.msg))
		case <-ctx.Done():
			g.shutdown(ctx.Err())
			return ctx.Err()
		}
	}
}
