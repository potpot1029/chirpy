package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestReplaceProfaneWords(t *testing.T) {
	tests := []struct {
		name  string
		chirp string
		want  string
	}{
		{
			name:  "no profane words",
			chirp: "hello world",
			want:  "hello world",
		},
		{
			name:  "all profane words",
			chirp: "kerfuffle sharbert fornax",
			want:  "**** **** ****",
		},
		{
			name:  "case insensitive",
			chirp: "KerFuffle FORNAX",
			want:  "**** ****",
		},
		{
			name:  "punctuation is not stripped",
			chirp: "sharbert! kerfuffle",
			want:  "sharbert! ****",
		},
		{
			name:  "empty chirp",
			chirp: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceProfaneWords(tt.chirp)

			if got != tt.want {
				t.Errorf("replaceProfaneWords(%q) expects %q, got %q", tt.chirp, tt.want, got)
			}
		})
	}
}

func TestCreateChirp(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	user := createTestUser(t, server, "user@example.com", "password")
	login := loginTestUser(t, server, "user@example.com", "password")
	nonUUIDSubjectToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		Subject:   "not-a-uuid",
	}).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("error making JWT: %v", err)
	}

	chirp := func(body string) map[string]string {
		return map[string]string{"body": body}
	}

	tests := []struct {
		name       string
		authHeader string
		body       any
		wantStatus int
		wantBody   string
	}{
		{
			name:       "valid chirp",
			authHeader: bearer(login.Token),
			body:       chirp("hello world"),
			wantStatus: http.StatusCreated,
			wantBody:   "hello world",
		},
		{
			name:       "profane words are replaced",
			authHeader: bearer(login.Token),
			body:       chirp("what a Kerfuffle and sharbert"),
			wantStatus: http.StatusCreated,
			wantBody:   "what a **** and ****",
		},
		{
			name:       "140 characters is allowed",
			authHeader: bearer(login.Token),
			body:       chirp(strings.Repeat("a", 140)),
			wantStatus: http.StatusCreated,
			wantBody:   strings.Repeat("a", 140),
		},
		{
			name:       "140 multi-byte characters is allowed",
			authHeader: bearer(login.Token),
			body:       chirp(strings.Repeat("😀", 140)),
			wantStatus: http.StatusCreated,
			wantBody:   strings.Repeat("😀", 140),
		},
		{
			name:       "141 characters is too long",
			authHeader: bearer(login.Token),
			body:       chirp(strings.Repeat("a", 141)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "",
		},
		{
			name:       "missing token",
			authHeader: "",
			body:       chirp("hello world"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "invalid token",
			authHeader: bearer("not-a-jwt"),
			body:       chirp("hello world"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "Bearer but no token",
			authHeader: "Bearer",
			body:       chirp("hello world"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "no space in Authorization",
			authHeader: "garbage",
			body:       chirp("hello world"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "wrong auth scheme",
			authHeader: "ApiKey " + login.Token,
			body:       chirp("hello world"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "token subject is not a uuid",
			authHeader: bearer(nonUUIDSubjectToken),
			body:       chirp("hello world"),
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "invalid JSON",
			authHeader: bearer(login.Token),
			body:       "{",
			wantStatus: http.StatusBadRequest,
			wantBody:   "",
		},
		{
			name:       "wrong body type",
			authHeader: bearer(login.Token),
			body:       `{"body": 42}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodPost, "/api/chirps", tt.authHeader, tt.body)

			if status != tt.wantStatus {
				t.Fatalf("POST /api/chirps expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if status != http.StatusCreated {
				return
			}
			created := decodeJSON[Chirp](t, body)
			if created.Body != tt.wantBody {
				t.Errorf("POST /api/chirps expects body %q, got %q", tt.wantBody, created.Body)
			}
			if created.UserID != user.ID {
				t.Errorf("POST /api/chirps expects user ID %v, got %v", user.ID, created.UserID)
			}
		})
	}
}

func TestGetChirps(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	alice := createTestUser(t, server, "alice@example.com", "password")
	aliceLogin := loginTestUser(t, server, "alice@example.com", "password")
	createTestUser(t, server, "bob@example.com", "password")
	bobLogin := loginTestUser(t, server, "bob@example.com", "password")

	first := createTestChirp(t, server, aliceLogin.Token, "first")
	second := createTestChirp(t, server, bobLogin.Token, "second")
	third := createTestChirp(t, server, aliceLogin.Token, "third")

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantIDs    []uuid.UUID
	}{
		{
			name:       "all chirps, oldest first",
			path:       "/api/chirps",
			wantStatus: http.StatusOK,
			wantIDs:    []uuid.UUID{first.ID, second.ID, third.ID},
		},
		{
			name:       "explicit ascending order",
			path:       "/api/chirps?sort=asc",
			wantStatus: http.StatusOK,
			wantIDs:    []uuid.UUID{first.ID, second.ID, third.ID},
		},
		{
			name:       "descending order",
			path:       "/api/chirps?sort=desc",
			wantStatus: http.StatusOK,
			wantIDs:    []uuid.UUID{third.ID, second.ID, first.ID},
		},
		{
			name:       "filter by author",
			path:       "/api/chirps?author_id=" + alice.ID.String(),
			wantStatus: http.StatusOK,
			wantIDs:    []uuid.UUID{first.ID, third.ID},
		},
		{
			name:       "filter by author, descending",
			path:       "/api/chirps?author_id=" + alice.ID.String() + "&sort=desc",
			wantStatus: http.StatusOK,
			wantIDs:    []uuid.UUID{third.ID, first.ID},
		},
		{
			name:       "author with no chirps",
			path:       "/api/chirps?author_id=" + uuid.NewString(),
			wantStatus: http.StatusOK,
			wantIDs:    []uuid.UUID{},
		},
		{
			name:       "invalid author_id",
			path:       "/api/chirps?author_id=not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantIDs:    []uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodGet, tt.path, "", nil)

			if status != tt.wantStatus {
				t.Fatalf("GET %s expects status %d, got %d: %s", tt.path, tt.wantStatus, status, body)
			}
			if status != http.StatusOK {
				return
			}
			chirps := decodeJSON[[]Chirp](t, body)
			gotIDs := make([]uuid.UUID, len(chirps))
			for i, chirp := range chirps {
				gotIDs[i] = chirp.ID
			}
			if !equalIDs(gotIDs, tt.wantIDs) {
				t.Errorf("GET %s expects chirps %v, got %v", tt.path, tt.wantIDs, gotIDs)
			}
		})
	}
}

func equalIDs(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestGetChirp(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	createTestUser(t, server, "user@example.com", "password")
	login := loginTestUser(t, server, "user@example.com", "password")
	chirp := createTestChirp(t, server, login.Token, "hello world")

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantChirp  Chirp
	}{
		{
			name:       "existing chirp",
			path:       "/api/chirps/" + chirp.ID.String(),
			wantStatus: http.StatusOK,
			wantChirp:  chirp,
		},
		{
			name:       "unknown chirp",
			path:       "/api/chirps/" + uuid.NewString(),
			wantStatus: http.StatusNotFound,
			wantChirp:  Chirp{},
		},
		{
			name:       "invalid id",
			path:       "/api/chirps/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantChirp:  Chirp{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodGet, tt.path, "", nil)

			if status != tt.wantStatus {
				t.Fatalf("GET %s expects status %d, got %d: %s", tt.path, tt.wantStatus, status, body)
			}
			if status != http.StatusOK {
				return
			}
			if got := decodeJSON[Chirp](t, body); got != tt.wantChirp {
				t.Errorf("GET %s expects %+v, got %+v", tt.path, tt.wantChirp, got)
			}
		})
	}
}

func TestDeleteChirp(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	createTestUser(t, server, "alice@example.com", "password")
	aliceLogin := loginTestUser(t, server, "alice@example.com", "password")
	createTestUser(t, server, "bob@example.com", "password")
	bobLogin := loginTestUser(t, server, "bob@example.com", "password")
	chirp := createTestChirp(t, server, aliceLogin.Token, "hello world")
	chirpPath := "/api/chirps/" + chirp.ID.String()

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "unknown chirp",
			path:       "/api/chirps/" + uuid.NewString(),
			authHeader: bearer(aliceLogin.Token),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing token",
			path:       chirpPath,
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			path:       chirpPath,
			authHeader: bearer("not-a-jwt"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "no space in Authorization",
			path:       chirpPath,
			authHeader: "garbage",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid id",
			path:       "/api/chirps/not-a-uuid",
			authHeader: bearer(aliceLogin.Token),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "someone else's chirp",
			path:       chirpPath,
			authHeader: bearer(bobLogin.Token),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "own chirp",
			path:       chirpPath,
			authHeader: bearer(aliceLogin.Token),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "already deleted",
			path:       chirpPath,
			authHeader: bearer(aliceLogin.Token),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodDelete, tt.path, tt.authHeader, nil)

			if status != tt.wantStatus {
				t.Errorf("DELETE %s expects status %d, got %d: %s", tt.path, tt.wantStatus, status, body)
			}
		})
	}
}
