package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/potpot1029/chirpy/internal/auth"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		brokenDB   bool
		wantStatus int
	}{
		{
			name:       "correct password",
			body:       map[string]string{"email": "user@example.com", "password": "password"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "incorrect password",
			body:       map[string]string{"email": "user@example.com", "password": "wrong password"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown email",
			body:       map[string]string{"email": "nobody@example.com", "password": "password"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid JSON",
			body:       "{",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "database error",
			body:       map[string]string{"email": "user@example.com", "password": "password"},
			brokenDB:   true,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			var user User
			if tt.brokenDB {
				server, _ = newTestServer(t, newBrokenTestDB(t))
			} else {
				server, _ = newTestServer(t, newTestDB(t))
				user = createTestUser(t, server, "user@example.com", "password")
			}

			status, body := doRequest(t, server, http.MethodPost, "/api/login", "", tt.body)

			if status != tt.wantStatus {
				t.Fatalf("POST /api/login expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if status != http.StatusOK {
				return
			}
			login := decodeJSON[UserLogin](t, body)
			if login.ID != user.ID {
				t.Errorf("POST /api/login expects user ID %v, got %v", user.ID, login.ID)
			}
			if login.RefreshToken == "" {
				t.Errorf("POST /api/login expects a refresh token, got %q", login.RefreshToken)
			}
			userID, err := auth.ValidateJWT(login.Token, testJWTSecret)
			if err != nil {
				t.Errorf("POST /api/login expects a valid access token, error = %v", err)
			}
			if userID != user.ID {
				t.Errorf("POST /api/login expects access token for user %v, got %v", user.ID, userID)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))
	user := createTestUser(t, server, "user@example.com", "password")
	login := loginTestUser(t, server, "user@example.com", "password")
	revokedLogin := loginTestUser(t, server, "user@example.com", "password")
	if status, body := doRequest(t, server, http.MethodPost, "/api/revoke", bearer(revokedLogin.RefreshToken), nil); status != http.StatusNoContent {
		t.Fatalf("POST /api/revoke expects status %d, got %d: %s", http.StatusNoContent, status, body)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "valid token",
			authHeader: bearer(login.RefreshToken),
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing token",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown token",
			authHeader: bearer("unknown-refresh-token"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "access token instead of refresh token",
			authHeader: bearer(login.Token),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "revoked token",
			authHeader: bearer(revokedLogin.RefreshToken),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "no space in Authorization",
			authHeader: "garbage",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer but no token",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodPost, "/api/refresh", tt.authHeader, nil)

			if status != tt.wantStatus {
				t.Fatalf("POST /api/refresh expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if status != http.StatusOK {
				return
			}
			resp := decodeJSON[struct {
				Token string `json:"token"`
			}](t, body)
			userID, err := auth.ValidateJWT(resp.Token, testJWTSecret)
			if err != nil {
				t.Errorf("POST /api/refresh expects a valid access token, error = %v", err)
			}
			if userID != user.ID {
				t.Errorf("POST /api/refresh expects access token for user %v, got %v", user.ID, userID)
			}
		})
	}
}

func TestRevoke(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		// useLoginToken sends the refresh token from a fresh login as authHeader
		useLoginToken bool
		brokenDB      bool
		wantStatus    int
	}{
		{
			name:       "missing token",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unknown token",
			authHeader: bearer("unknown-refresh-token"),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "no space in Authorization",
			authHeader: "garbage",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer but no token",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "valid token",
			useLoginToken: true,
			wantStatus:    http.StatusNoContent,
		},
		{
			name:       "database error",
			authHeader: bearer("some-refresh-token"),
			brokenDB:   true,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			authHeader := tt.authHeader
			if tt.brokenDB {
				server, _ = newTestServer(t, newBrokenTestDB(t))
			} else {
				server, _ = newTestServer(t, newTestDB(t))
				createTestUser(t, server, "user@example.com", "password")
				if tt.useLoginToken {
					login := loginTestUser(t, server, "user@example.com", "password")
					authHeader = bearer(login.RefreshToken)
				}
			}

			status, body := doRequest(t, server, http.MethodPost, "/api/revoke", authHeader, nil)

			if status != tt.wantStatus {
				t.Fatalf("POST /api/revoke expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if status != http.StatusNoContent {
				return
			}
			// the revoked token can no longer be refreshed
			status, _ = doRequest(t, server, http.MethodPost, "/api/refresh", authHeader, nil)
			if status != http.StatusUnauthorized {
				t.Errorf("POST /api/refresh expects status %d after revoke, got %d", http.StatusUnauthorized, status)
			}
		})
	}
}
