package main

import (
	"context"
	"log"

	"nhooyr.io/websocket"
)

type Player struct {
	Name    string
	conn    *websocket.Conn
	game    *Game
	msgChan chan []byte
}

func handleWSConnection(ctx context.Context, conn *websocket.Conn, game *Game) error {
	p := &Player{conn: conn, game: game, Name: "Player", msgChan: make(chan []byte)}

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
			log.Println(string(msg))
		}
	}
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
				p.conn.CloseNow()
				return err
			}
		case <-ctx.Done():
			err := ctx.Err()
			p.game.playerLeave <- p
			p.conn.Close(websocket.StatusNormalClosure, err.Error())
			return err
		}
	}
}
