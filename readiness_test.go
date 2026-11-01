package main

import (
	"net/http"
	"testing"
)

func TestReadiness(t *testing.T) {
	server, _ := newTestServer(t, nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "server is ready",
			path:       "/api/healthz",
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, server, http.MethodGet, tt.path, "", nil)

			if status != tt.wantStatus {
				t.Errorf("GET %s expects status %d, got %d", tt.path, tt.wantStatus, status)
			}
			if string(body) != tt.wantBody {
				t.Errorf("GET %s expects body %q, got %q", tt.path, tt.wantBody, body)
			}
		})
	}
}
