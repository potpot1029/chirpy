package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/potpot1029/http-server-go/internal/auth"
	"github.com/potpot1029/http-server-go/internal/database"
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
		respondWithError(w, http.StatusInternalServerError, "error decoding request body", err)
		return
	}

	// authorization
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

	// chirps checking
	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
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
	// optional paramaters
	authorID := req.URL.Query().Get("author_id")
	sortOrder := req.URL.Query().Get("sort")

	var dbChirps []database.Chirp
	var err error

	if authorID != "" {
		dbChirps, err = cfg.dbQueries.GetChirpsByUserID(req.Context(), uuid.MustParse(authorID))
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

	chirpID := uuid.MustParse(req.PathValue("chirpID"))

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
	// get chirp
	chirpID := uuid.MustParse(req.PathValue("chirpID"))

	chirp, err := cfg.dbQueries.GetChirp(req.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "error getting chirp from database", err)
		return
	}

	// get user info
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

	if userID != chirp.UserID { // unauthorized
		respondWithError(w, http.StatusForbidden, "you cannot delete a chirp that is not yours", nil)
		return
	}

	// delete chirp
	err = cfg.dbQueries.DeleteChirp(req.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error deleting chirp in database", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
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
