package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestPolkaWebhook(t *testing.T) {
	server, _ := newTestServer(t, newTestDB(t))

	// each row gets its own user, so a request that is wrongly accepted
	// cannot upgrade the user that a later row checks
	event := func(name string) func(userID uuid.UUID) any {
		return func(userID uuid.UUID) any {
			return map[string]any{
				"event": name,
				"data":  map[string]string{"user_id": userID.String()},
			}
		}
	}
	unknownUserEvent := func(name string) func(userID uuid.UUID) any {
		return func(uuid.UUID) any {
			return event(name)(uuid.New())
		}
	}
	raw := func(body string) func(userID uuid.UUID) any {
		return func(uuid.UUID) any {
			return body
		}
	}

	tests := []struct {
		name        string
		authHeader  string
		body        func(userID uuid.UUID) any
		wantStatus  int
		wantRedUser bool
	}{
		{
			name:        "missing api key",
			authHeader:  "",
			body:        event("user.upgraded"),
			wantStatus:  http.StatusUnauthorized,
			wantRedUser: false,
		},
		{
			name:        "wrong api key",
			authHeader:  "ApiKey wrong-key",
			body:        event("user.upgraded"),
			wantStatus:  http.StatusUnauthorized,
			wantRedUser: false,
		},
		{
			name:        "ApiKey but no key",
			authHeader:  "ApiKey",
			body:        event("user.upgraded"),
			wantStatus:  http.StatusUnauthorized,
			wantRedUser: false,
		},
		{
			name:        "no space in Authorization",
			authHeader:  "garbage",
			body:        event("user.upgraded"),
			wantStatus:  http.StatusUnauthorized,
			wantRedUser: false,
		},
		{
			name:        "correct key with wrong auth scheme",
			authHeader:  "Bearer " + testPolkaKey,
			body:        event("user.upgraded"),
			wantStatus:  http.StatusUnauthorized,
			wantRedUser: false,
		},
		{
			name:        "invalid JSON",
			authHeader:  "ApiKey " + testPolkaKey,
			body:        raw("{"),
			wantStatus:  http.StatusBadRequest,
			wantRedUser: false,
		},
		{
			name:        "invalid user_id",
			authHeader:  "ApiKey " + testPolkaKey,
			body:        raw(`{"event": "user.upgraded", "data": {"user_id": "not-a-uuid"}}`),
			wantStatus:  http.StatusBadRequest,
			wantRedUser: false,
		},
		{
			name:        "other events are ignored",
			authHeader:  "ApiKey " + testPolkaKey,
			body:        event("user.payment_failed"),
			wantStatus:  http.StatusNoContent,
			wantRedUser: false,
		},
		{
			name:        "unknown user",
			authHeader:  "ApiKey " + testPolkaKey,
			body:        unknownUserEvent("user.upgraded"),
			wantStatus:  http.StatusNotFound,
			wantRedUser: false,
		},
		{
			name:        "user upgraded",
			authHeader:  "ApiKey " + testPolkaKey,
			body:        event("user.upgraded"),
			wantStatus:  http.StatusNoContent,
			wantRedUser: true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := fmt.Sprintf("user%d@example.com", i)
			user := createTestUser(t, server, email, "password")

			status, body := doRequest(t, server, http.MethodPost, "/api/polka/webhooks", tt.authHeader, tt.body(user.ID))

			if status != tt.wantStatus {
				t.Fatalf("POST /api/polka/webhooks expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			login := loginTestUser(t, server, email, "password")
			if login.IsChirpyRed != tt.wantRedUser {
				t.Errorf("POST /api/polka/webhooks expects is_chirpy_red %v, got %v", tt.wantRedUser, login.IsChirpyRed)
			}
		})
	}
}
