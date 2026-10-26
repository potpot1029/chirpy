package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/potpot1029/chirpy/internal/auth"
	"github.com/potpot1029/chirpy/internal/database"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, req *http.Request) {
	type parameter struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "error decoding request body", err)
		return
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error hashing password", err)
		return
	}

	user, err := cfg.dbQueries.CreateUser(req.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hash,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			respondWithError(w, http.StatusConflict, "email already registered", err)
			return
		}

		respondWithError(w, http.StatusInternalServerError, "error creating user in database", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, response{
		User: User{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
	})
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, req *http.Request) {
	type parameter struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		User
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "error decoding request body", err)
		return
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error hashing password", err)
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

	user, err := cfg.dbQueries.GetUserByID(req.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error retrieving user info by ID", err)
		return
	}

	new_user, err := cfg.dbQueries.UpdateUser(req.Context(), database.UpdateUserParams{
		Email:          user.Email,
		Email_2:        params.Email,
		HashedPassword: hash,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			respondWithError(w, http.StatusConflict, "email already registered", err)
			return
		}

		respondWithError(w, http.StatusInternalServerError, "error updating user in database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:          new_user.ID,
			CreatedAt:   new_user.CreatedAt,
			UpdatedAt:   new_user.UpdatedAt,
			Email:       new_user.Email,
			IsChirpyRed: new_user.IsChirpyRed,
		},
	})
}
