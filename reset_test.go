package main

import (
	"net/http"
	"testing"
)

func TestReset(t *testing.T) {
	tests := []struct {
		name           string
		platform       string
		wantStatus     int
		wantHits       int32
		wantUserExists bool
	}{
		{
			name:           "dev resets hits and users",
			platform:       "dev",
			wantStatus:     http.StatusOK,
			wantHits:       0,
			wantUserExists: false,
		},
		{
			name:           "other platforms are forbidden",
			platform:       "production",
			wantStatus:     http.StatusForbidden,
			wantHits:       1,
			wantUserExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, apiCfg := newTestServer(t, newTestDB(t))
			apiCfg.platform = tt.platform
			createTestUser(t, server, "user@example.com", "password")
			doRequest(t, server, http.MethodGet, "/app/", "", nil)

			status, body := doRequest(t, server, http.MethodPost, "/admin/reset", "", nil)

			if status != tt.wantStatus {
				t.Errorf("POST /admin/reset expects status %d, got %d: %s", tt.wantStatus, status, body)
			}
			if hits := apiCfg.fileserverHits.Load(); hits != tt.wantHits {
				t.Errorf("POST /admin/reset expects hit counter %d, got %d", tt.wantHits, hits)
			}
			_, err := apiCfg.dbQueries.GetUserByEmail(t.Context(), "user@example.com")
			if userExists := err == nil; userExists != tt.wantUserExists {
				t.Errorf("POST /admin/reset expects user exists %v, got %v", tt.wantUserExists, userExists)
			}
		})
	}
}
