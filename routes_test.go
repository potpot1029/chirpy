package main

import (
	"net/http"
	"testing"
)

func TestFileServer(t *testing.T) {
	server, _ := newTestServer(t, nil)

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "index page",
			path:       "/app/",
			wantStatus: http.StatusOK,
		},
		{
			name:       "logo",
			path:       "/app/assets/logo.png",
			wantStatus: http.StatusOK,
		},
		{
			name:       "source files are not served",
			path:       "/app/main.go",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := doRequest(t, server, http.MethodGet, tt.path, "", nil)

			if status != tt.wantStatus {
				t.Errorf("GET %s expects status %d, got %d", tt.path, tt.wantStatus, status)
			}
		})
	}
}
