package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/potpot1029/http-server-go/internal/auth"
)

func (cfg *apiConfig) handlerPolkaWebhook(w http.ResponseWriter, req *http.Request) {
	type parameter struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	// check api key
	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil || apiKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "invalid polka api key", nil)
		return
	}

	// decode request
	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error decoding request body", err)
		return
	}

	if params.Event != "user.upgraded" {
		respondWithJSON(w, http.StatusNoContent, nil)
		return
	}

	err = cfg.dbQueries.UpgradeChirpyRedByID(req.Context(), params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "error upgrading to chirpy red by user id", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}
