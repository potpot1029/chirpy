package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCreateUser(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	createTestUser(t, server, "taken@example.com", "password")

	tests := []struct {
		name            string
		body            any
		wantStatus      int
		wantEmail       string
		wantIsChirpyRed bool
	}{
		{
			name:            "new user",
			body:            map[string]string{"email": "user@example.com", "password": "password"},
			wantStatus:      http.StatusCreated,
			wantEmail:       "user@example.com",
			wantIsChirpyRed: false,
		},
		{
			name:            "duplicate email",
			body:            map[string]string{"email": "taken@example.com", "password": "another password"},
			wantStatus:      http.StatusConflict,
			wantEmail:       "",
			wantIsChirpyRed: false,
		},
		{
			name:            "invalid JSON",
			body:            "{not json",
			wantStatus:      http.StatusBadRequest,
			wantEmail:       "",
			wantIsChirpyRed: false,
		},
		{
			name:            "empty body",
			body:            nil,
			wantStatus:      http.StatusBadRequest,
			wantEmail:       "",
			wantIsChirpyRed: false,
		},
		{
			name:            "wrong field types",
			body:            `{"email": 1, "password": true}`,
			wantStatus:      http.StatusBadRequest,
			wantEmail:       "",
			wantIsChirpyRed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodPost, "/api/users", "", tt.body)

			if status != tt.wantStatus {
				t.Fatalf("POST /api/users expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if status != http.StatusCreated {
				return
			}
			user := decodeJSON[User](t, body)
			if user.ID == uuid.Nil {
				t.Errorf("POST /api/users expects a user ID, got %v", user.ID)
			}
			if user.Email != tt.wantEmail {
				t.Errorf("POST /api/users expects email %q, got %q", tt.wantEmail, user.Email)
			}
			if user.IsChirpyRed != tt.wantIsChirpyRed {
				t.Errorf("POST /api/users expects is_chirpy_red %v, got %v", tt.wantIsChirpyRed, user.IsChirpyRed)
			}
			if strings.Contains(string(body), "password") {
				t.Errorf("POST /api/users expects no password in the response, got %s", body)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	user := createTestUser(t, server, "user@example.com", "password")
	login := loginTestUser(t, server, "user@example.com", "password")
	createTestUser(t, server, "taken@example.com", "password")

	newCredentials := map[string]string{"email": "new@example.com", "password": "new password"}

	tests := []struct {
		name       string
		authHeader string
		body       any
		wantStatus int
		wantEmail  string
	}{
		{
			name:       "missing token",
			authHeader: "",
			body:       newCredentials,
			wantStatus: http.StatusUnauthorized,
			wantEmail:  "",
		},
		{
			name:       "invalid token",
			authHeader: bearer("not-a-jwt"),
			body:       newCredentials,
			wantStatus: http.StatusUnauthorized,
			wantEmail:  "",
		},
		{
			name:       "no space in Authorization",
			authHeader: "garbage",
			body:       newCredentials,
			wantStatus: http.StatusUnauthorized,
			wantEmail:  "",
		},
		{
			name:       "Bearer but no token",
			authHeader: "Bearer",
			body:       newCredentials,
			wantStatus: http.StatusUnauthorized,
			wantEmail:  "",
		},
		{
			name:       "invalid JSON",
			authHeader: bearer(login.Token),
			body:       "{",
			wantStatus: http.StatusBadRequest,
			wantEmail:  "",
		},
		{
			name:       "duplicate email",
			authHeader: bearer(login.Token),
			body:       map[string]string{"email": "taken@example.com", "password": "new password"},
			wantStatus: http.StatusConflict,
			wantEmail:  "",
		},
		{
			name:       "valid token",
			authHeader: bearer(login.Token),
			body:       newCredentials,
			wantStatus: http.StatusOK,
			wantEmail:  "new@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodPut, "/api/users", tt.authHeader, tt.body)

			if status != tt.wantStatus {
				t.Fatalf("PUT /api/users expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if status != http.StatusOK {
				return
			}
			updated := decodeJSON[User](t, body)
			if updated.ID != user.ID {
				t.Errorf("PUT /api/users expects user ID %v, got %v", user.ID, updated.ID)
			}
			if updated.Email != tt.wantEmail {
				t.Errorf("PUT /api/users expects email %q, got %q", tt.wantEmail, updated.Email)
			}
			// the new credentials work
			loginTestUser(t, server, newCredentials["email"], newCredentials["password"])
		})
	}
}
