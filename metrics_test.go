package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestMetrics(t *testing.T) {
	tests := []struct {
		name       string
		visits     []string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "no visits",
			visits:     []string{},
			wantStatus: http.StatusOK,
			wantBody:   "visited 0 times",
		},
		{
			name:       "every file server request is counted",
			visits:     []string{"/app/", "/app/assets/logo.png", "/app/main.go"},
			wantStatus: http.StatusOK,
			wantBody:   "visited 3 times",
		},
		{
			name:       "api requests are not counted",
			visits:     []string{"/app/", "/api/healthz"},
			wantStatus: http.StatusOK,
			wantBody:   "visited 1 times",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := newTestServer(t, nil)
			for _, path := range tt.visits {
				doRequest(t, server, http.MethodGet, path, "", nil)
			}

			status, body := doRequest(t, server, http.MethodGet, "/admin/metrics", "", nil)

			if status != tt.wantStatus {
				t.Errorf("GET /admin/metrics expects status %d, got %d", tt.wantStatus, status)
			}
			if !strings.Contains(string(body), tt.wantBody) {
				t.Errorf("GET /admin/metrics expects body to contain %q, got %q", tt.wantBody, body)
			}
		})
	}
}
