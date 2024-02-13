package main

import (
	"context"
	"log"

	"nhooyr.io/websocket"
)

type Player struct {
	Name         string
	conn         *websocket.Conn
	game         *Game
	msgChan      chan []byte
	shutdownChan chan error
	number       int
	move         byte
}

type PlayerMsg struct {
	player *Player
	msg    []byte
}

func handleWSConnection(ctx context.Context, conn *websocket.Conn, game *Game) error {
	p := &Player{conn: conn, game: game, Name: "Player", msgChan: make(chan []byte), shutdownChan: make(chan error), number: -1}

	return p.run(ctx)
}

func (p *Player) readPump(ctx context.Context) {
	for {
		msgType, msg, err := p.conn.Read(ctx)

		if err != nil {
			log.Println(err)
			p.game.playerLeave <- p
			return
		}

		if msgType == websocket.MessageText {
			p.game.playerMsg <- PlayerMsg{player: p, msg: msg}
		}
	}
}

func (p *Player) shutdown(err error, notifyGame bool) {
	log.Println("shutting down player", p.Name)
	if notifyGame {
		p.game.playerLeave <- p
	}
	p.conn.Close(websocket.StatusNormalClosure, err.Error())
}

func (p *Player) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer close(p.msgChan)

	go p.readPump(ctx)

	go func(p *Player) { p.game.playerJoin <- p }(p)

	for {
		select {
		case msg := <-p.msgChan:
			err := p.conn.Write(ctx, websocket.MessageText, msg)
			if err != nil {
				log.Println(err)
				p.game.playerLeave <- p
				if websocket.CloseStatus(err) != -1 {
					p.conn.Close(websocket.StatusNormalClosure, err.Error())
				}
				return err
			}
		case err := <-p.shutdownChan:
			p.shutdown(err, false)
			return err
		case <-ctx.Done():
			p.shutdown(ctx.Err(), true)
			return ctx.Err()
		}
	}
}
