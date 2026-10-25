package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/potpot1029/chirpy/internal/database"
)

const (
	testJWTSecret = "test-jwt-secret"
	testPolkaKey  = "test-polka-key"
)

// newTestDB connects to TEST_DB_URL and empties it before and after the test.
// Tests that need a database are skipped when TEST_DB_URL is not set.
// Never point TEST_DB_URL at a database whose data you want to keep.
func newTestDB(t *testing.T) *database.Queries {
	t.Helper()

	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		t.Skip("TEST_DB_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("error opening test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("error connecting to test database: %v", err)
	}

	dbQueries := database.New(db)
	if err := dbQueries.ResetUser(t.Context()); err != nil {
		t.Fatalf("error resetting test database: %v", err)
	}
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanup runs
		if err := dbQueries.ResetUser(context.Background()); err != nil {
			t.Errorf("error resetting test database: %v", err)
		}
		db.Close()
	})

	return dbQueries
}

func newTestServer(t *testing.T, dbQueries *database.Queries) (*httptest.Server, *apiConfig) {
	t.Helper()

	apiCfg := &apiConfig{
		dbQueries: dbQueries,
		platform:  "dev",
		jwtSecret: testJWTSecret,
		polkaKey:  testPolkaKey,
	}
	server := httptest.NewServer(apiCfg.routes())
	t.Cleanup(server.Close)

	return server, apiCfg
}

// doRequest sends body as JSON (or as-is when it is a string) and returns the
// status code and response body.
func doRequest(t *testing.T, server *httptest.Server, method, path, authHeader string, body any) (int, []byte) {
	t.Helper()

	var reqBody io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reqBody = strings.NewReader(b)
	default:
		data, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("error marshaling request body: %v", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, server.URL+path, reqBody)
	if err != nil {
		t.Fatalf("error creating request: %v", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		// net/http recovers a handler panic by closing the connection, so the client sees EOF
		t.Fatalf("%s %s failed (did the handler panic?): %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body: %v", err)
	}

	return resp.StatusCode, data
}

func decodeJSON[T any](t *testing.T, data []byte) T {
	t.Helper()

	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("error decoding response body %q: %v", data, err)
	}
	return v
}

func bearer(token string) string {
	return "Bearer " + token
}

func createTestUser(t *testing.T, server *httptest.Server, email, password string) User {
	t.Helper()

	status, body := doRequest(t, server, http.MethodPost, "/api/users", "", map[string]string{
		"email":    email,
		"password": password,
	})
	if status != http.StatusCreated {
		t.Fatalf("create user expects status %d, got %d: %s", http.StatusCreated, status, body)
	}
	return decodeJSON[User](t, body)
}

func loginTestUser(t *testing.T, server *httptest.Server, email, password string) UserLogin {
	t.Helper()

	status, body := doRequest(t, server, http.MethodPost, "/api/login", "", map[string]string{
		"email":    email,
		"password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login expects status %d, got %d: %s", http.StatusOK, status, body)
	}
	return decodeJSON[UserLogin](t, body)
}

func createTestChirp(t *testing.T, server *httptest.Server, token, chirpBody string) Chirp {
	t.Helper()

	status, body := doRequest(t, server, http.MethodPost, "/api/chirps", bearer(token), map[string]string{
		"body": chirpBody,
	})
	if status != http.StatusCreated {
		t.Fatalf("create chirp expects status %d, got %d: %s", http.StatusCreated, status, body)
	}
	return decodeJSON[Chirp](t, body)
}
