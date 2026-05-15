package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func handlerValidate(w http.ResponseWriter, req *http.Request) {
	type parameter struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error decoding response body", err)
		return
	}

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "chirp is too long", nil)
		return
	}

	type returnJSON struct {
		Valid bool `json:"valid"`
	}
	respondWithJSON(w, http.StatusOK, returnJSON{
		Valid: true,
	})
}

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
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling response body: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(data)
}
