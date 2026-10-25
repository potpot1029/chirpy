package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/potpot1029/chirpy/internal/auth"
	"github.com/potpot1029/chirpy/internal/database"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, req *http.Request) {
	type parameter struct {
		Body string `json:"body"`
	}
	type response struct {
		Chirp
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "error decoding request body", err)
		return
	}

	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "cannot retrieve bearer token", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid bearer token", err)
		return
	}

	const maxChirpLength = 140
	if utf8.RuneCountInString(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "chirp is too long", nil)
		return
	}
	params.Body = replaceProfaneWords(params.Body)

	chirp, err := cfg.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   params.Body,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating chirp in database", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		Chirp: Chirp(chirp),
	})
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, req *http.Request) {
	authorIDStr := req.URL.Query().Get("author_id")
	sortOrder := req.URL.Query().Get("sort")

	var dbChirps []database.Chirp
	var err error

	if authorIDStr != "" {
		authorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid author ID", err)
			return
		}

		dbChirps, err = cfg.dbQueries.GetChirpsByUserID(req.Context(), authorID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error getting chirps from database", err)
			return
		}
	} else {
		dbChirps, err = cfg.dbQueries.GetChirps(req.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error getting chirps from database", err)
			return
		}
	}

	if sortOrder == "desc" {
		sort.Slice(dbChirps, func(i, j int) bool { return dbChirps[i].CreatedAt.After(dbChirps[j].CreatedAt) })
	}

	chirps := make([]Chirp, len(dbChirps))
	for i, chirp := range dbChirps {
		chirps[i] = Chirp(chirp)
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Chirp
	}

	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid chirp ID", err)
		return
	}

	chirp, err := cfg.dbQueries.GetChirp(req.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "error getting chirp from database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Chirp: Chirp(chirp),
	})
}

func replaceProfaneWords(chirp string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}

	words := strings.Split(chirp, " ")

	for i := 0; i < len(words); i++ {
		word := words[i]
		wordLowered := strings.ToLower(word)

		for _, profaneWord := range profaneWords {
			if profaneWord == wordLowered {
				words[i] = "****"
				break
			}
		}
	}

	return strings.Join(words, " ")
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, req *http.Request) {
	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid chirp ID", err)
		return
	}

	chirp, err := cfg.dbQueries.GetChirp(req.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "error getting chirp from database", err)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "cannot retrieve access token from header", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid access token", err)
		return
	}

	if userID != chirp.UserID {
		respondWithError(w, http.StatusForbidden, "you cannot delete a chirp that is not yours", nil)
		return
	}

	err = cfg.dbQueries.DeleteChirp(req.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error deleting chirp in database", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}
