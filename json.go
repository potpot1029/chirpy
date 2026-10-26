package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}

	type returnError struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, returnError{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	if payload == nil {
		w.WriteHeader(code)
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling response body: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}
