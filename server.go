package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RPSServer struct {
	router *chi.Mux
}

func NewRPSServer() *RPSServer {
	router := chi.NewRouter()

	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		JSONResponse(w, true)
	})

	return &RPSServer{router: router}
}
