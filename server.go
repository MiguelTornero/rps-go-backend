package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"nhooyr.io/websocket"
)

const (
	CODE_CHARS                    string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	CODE_LENGTH_DEFAULT           int    = 5
	CODE_TRIES_DEFAULT            int    = 10
	GAME_LIFETIME_SECONDS_DEFAULT int    = 3600
	GAME_PLAYER_LIMIT_DEFAULT     int    = 2
)

type RPSServer struct {
	router     *chi.Mux
	games      map[string]*Game
	addGame    chan *Game
	removeGame chan *Game
	rng        *rand.Rand
}

func NewRPSServer() *RPSServer {
	server := &RPSServer{router: chi.NewRouter(), games: map[string]*Game{}, rng: rand.New(rand.NewSource(time.Now().UnixMicro())), addGame: make(chan *Game), removeGame: make(chan *Game)}

	server.router.Get("/test", func(w http.ResponseWriter, r *http.Request) { JSONResponse(w, true) })
	server.router.Get("/connect", server.HandleConnect)

	test_game := NewGame(server)
	go test_game.run(context.Background())
	test_game.Code = "TEST"
	server.games["TEST"] = test_game

	return server
}

func (rps *RPSServer) HandleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})

	if err != nil {
		log.Println(err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(GAME_LIFETIME_SECONDS_DEFAULT)*time.Second)
	defer cancel()

	gameId := r.URL.Query().Get("gameId")
	gameId = strings.TrimSpace(gameId)
	gameId = strings.ToUpper(gameId)
	log.Println("connecting to game:", gameId)

	game, ok := rps.games[gameId]
	if !ok {
		conn.Close(websocket.StatusNormalClosure, "invalid connect code")
		return
	}

	if game.full {
		conn.Close(websocket.StatusNormalClosure, "game full")
		return
	}

	handleWSConnection(ctx, conn, game)

}

func (rps *RPSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rps.router.ServeHTTP(w, r)
}

func (rps *RPSServer) run(ctx context.Context) error {
	for {
		select {
		case g := <-rps.addGame:
			code, err := rps.generateConnectCode(CODE_LENGTH_DEFAULT, 0, CODE_TRIES_DEFAULT)
			if err != nil {
				log.Println(err)
				break
			}
			g.Code = code
			rps.games[code] = g

		case g := <-rps.removeGame:
			code := g.Code
			delete(rps.games, code)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (rps *RPSServer) generateConnectCode(length int, currentTry int, maxTries int) (string, error) {
	if currentTry >= maxTries {
		return "", errors.New("couldn't generate unique code in " + strconv.Itoa(maxTries) + " tries")
	}
	output := []byte{}
	for i := 0; i <= length; i++ {
		n := rps.rng.Intn(len(CODE_CHARS))
		letter := CODE_CHARS[n]
		output = append(output, letter)
	}
	str_output := string(output)
	if _, ok := rps.games[str_output]; ok {
		return rps.generateConnectCode(length, currentTry+1, maxTries)
	}
	return str_output, nil
}
