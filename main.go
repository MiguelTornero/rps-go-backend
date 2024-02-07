package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

const (
	DEFAULT_PORT = 5000
)

func main() {
	r := NewRPSServer()

	port := os.Getenv("RPS_APP_PORT")
	portNum, err := strconv.Atoi(port)

	if err != nil {
		portNum = DEFAULT_PORT
	}

	port = strconv.Itoa(portNum)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.run(ctx)

	fmt.Println("Listening on port:", port)
	http.ListenAndServe(":"+port, r)
}
