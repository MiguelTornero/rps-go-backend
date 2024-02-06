package main

import (
	"encoding/json"
	"net/http"
)

func JSONResponse(w http.ResponseWriter, data any) {
	resp, err := json.Marshal(&data)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}
