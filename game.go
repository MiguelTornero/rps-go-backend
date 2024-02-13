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

			msg.player.move = getMove(msg.msg)

			if g.player1.move == 0 || g.player2.move == 0 {
				break
			}

			g.broadcast([]byte("message: " + getMoveString(g.player1)))
			g.broadcast([]byte("message: " + getMoveString(g.player2)))

			g.broadcast([]byte("result: " + determineWinner(g.player1, g.player2)))

			g.player1.move = 0
			g.player2.move = 0

			fmt.Println(msg.player.Name, string(msg.msg))
		case <-ctx.Done():
			g.shutdown(ctx.Err())
			return ctx.Err()
		}
	}
}

func getMove(s []byte) byte {
	if len(s) < 1 {
		return 0
	}
	if s[0] != 'r' && s[0] != 'p' && s[0] != 's' {
		return 0
	}
	return s[0]
}

func getMoveString(p *Player) string {
	if p.move == 'r' {
		return p.Name + " played rock"
	}

	if p.move == 'p' {
		return p.Name + " played paper"
	}

	if p.move == 's' {
		return p.Name + " played scissors"
	}

	return p.Name + " played an invalid move"
}

func mapMoveToInt(move byte) int {
	if move == 'r' {
		return 0
	}

	if move == 'p' {
		return 1
	}

	if move == 's' {
		return 2
	}

	return -1
}

func determineWinner(player1 *Player, player2 *Player) string {
	/*
		  r p s
		r 0 2 1
		p 1 0 2
		s 2 1 0
	*/
	winMatrix := []int{0, 2, 1, 1, 0, 2, 2, 1, 0}

	player1Move := mapMoveToInt(player1.move)
	player2Move := mapMoveToInt(player2.move)

	if player1Move < 0 || player2Move < 0 {
		return "invalid input"
	}

	winner := winMatrix[player1Move*3+player2Move]

	if winner == 0 {
		return "tie"
	}

	if winner == 1 {
		return player1.Name + " wins"
	}

	if winner == 2 {
		return player2.Name + " wins"
	}

	return "invalid input"
}
