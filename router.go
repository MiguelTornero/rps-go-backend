package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"nhooyr.io/websocket"
)

const (
	TURN_TIME_MIN = 1
	TURN_TIME_MAX = 10

	REQUIRED_WINS_MAX = 1
	REQUIRED_WINS_MIN = 10

	GAME_TIME_LIMIT_SECONDS = 3600
	GAME_LIMIT              = 100
)

func Clamp(n int, min int, max int) int {
	if n < min {
		return min
	}

	if n > max {
		return max
	}

	return n
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func NewErrorResponse(w http.ResponseWriter, status int, message string) {
	output, err := json.Marshal(&ErrorResponse{Status: status, Message: message})
	if err != nil {
		panic(err)
	}
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func NewJSONResponse(w http.ResponseWriter, data any) {
	resp, err := json.Marshal(&data)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

type GameInfo struct {
	WSUrl        string `json:"websocket_url"`
	Code         string `json:"code"`
	RequiredWins int    `json:"required_wins"`
	TimeLimit    int    `json:"time_limit"`
	StartTime    int    `json:"start_time"`
}

type GamesMap struct {
	m sync.Map
}

func (gm *GamesMap) Load(code string) (*GameInfo, bool) {
	v, ok := gm.m.Load(code)
	if ok {
		return v.(*GameInfo), true
	}
	return nil, false
}

func (gm *GamesMap) Store(code string, info *GameInfo) {
	gm.m.Store(code, info)
}

func (gm *GamesMap) Delete(code string) bool {
	_, ok := gm.m.LoadAndDelete(code)

	return ok
}

type GlobalContext struct {
	Games       GamesMap
	GameCounter atomic.Int64
}

func (ctx *GlobalContext) GetCode(counter int64) string {
	now := int(time.Now().UnixMicro() % 100)

	return string([]byte{'A' + byte(counter/26), 'A' + byte(counter%26)}) + strconv.Itoa(now/10) + strconv.Itoa(now%10) + string([]byte{'A' + byte(rand.Int()%26), 'A' + byte(rand.Int()%26)})
}

type CreateGameInput struct {
	RequiredWins int `json:"required_wins"`
	TimeLimit    int `json:"time_limit"`
}

func (ctx *GlobalContext) CreateGame(w http.ResponseWriter, r *http.Request) {
	counter := ctx.GameCounter.Load()
	if counter >= GAME_LIMIT {
		NewErrorResponse(w, http.StatusBadRequest, "game limit reached")
		return
	}

	decoder := json.NewDecoder(r.Body)

	var input CreateGameInput

	err := decoder.Decode(&input)
	if err != nil {
		NewErrorResponse(w, http.StatusBadRequest, "invalid input")
		return
	}

	code := ctx.GetCode(counter)

	game_info := &GameInfo{
		WSUrl:        "TODO",
		Code:         code,
		RequiredWins: Clamp(input.RequiredWins, REQUIRED_WINS_MIN, REQUIRED_WINS_MAX),
		TimeLimit:    Clamp(input.TimeLimit, TURN_TIME_MIN, TURN_TIME_MAX),
		StartTime:    int(time.Now().Unix()),
	}

	ctx.Games.Store(code, game_info)
	ctx.GameCounter.Add(1)

	go func() {
		time.Sleep(time.Second * GAME_TIME_LIMIT_SECONDS)
		if ctx.Games.Delete(code) {
			ctx.GameCounter.Add(-1)
			log.Println("stopping game:", code)
		}
	}()

	NewJSONResponse(w, game_info)
}

func (gctx *GlobalContext) ConnectGame(w http.ResponseWriter, r *http.Request) {
	gameId := r.URL.Query().Get("gameId")
	log.Println("connecting to game:", gameId)

	_, ok := gctx.Games.Load(gameId)

	if !ok {
		NewErrorResponse(w, http.StatusBadRequest, "invalid game id")
		return
	}

	websocketConnection, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Println(err)
		return
	}
	defer websocketConnection.CloseNow()

	ctx, cancel := context.WithTimeout(r.Context(), time.Second*GAME_TIME_LIMIT_SECONDS)
	defer cancel()

	go func() {
		for {
			_, msg, err := websocketConnection.Read(ctx)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println(string(msg))
		}
	}()

	go func() {
		for i := 1; i <= 1000; i++ {
			err := websocketConnection.Write(r.Context(), websocket.MessageText, []byte("Hello. "+strconv.Itoa(i)))
			if err != nil {
				fmt.Println(err)
				return
			}
			time.Sleep(2 * time.Second)
		}
	}()

	<-ctx.Done()
}

func NewRouter() *chi.Mux {
	router := chi.NewRouter()
	var ctx GlobalContext

	router.Use(middleware.Recoverer)

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		NewJSONResponse(w, true)
	})

	router.Post("/create_game", ctx.CreateGame)
	router.Get("/connect", ctx.ConnectGame)

	return router
}
