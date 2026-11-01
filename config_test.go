package main

import (
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		wantErr      string
		wantPort     string
		wantPlatform string
	}{
		{
			name: "all variables set",
			env: map[string]string{
				"DB_URL":     "postgres://localhost/chirpy",
				"JWT_SECRET": "secret",
				"POLKA_KEY":  "key",
				"PLATFORM":   "dev",
				"PORT":       "9000",
			},
			wantErr:      "",
			wantPort:     "9000",
			wantPlatform: "dev",
		},
		{
			name: "port defaults to 8080",
			env: map[string]string{
				"DB_URL":     "postgres://localhost/chirpy",
				"JWT_SECRET": "secret",
				"POLKA_KEY":  "key",
			},
			wantErr:      "",
			wantPort:     defaultPort,
			wantPlatform: "",
		},
		{
			name: "one required variable missing",
			env: map[string]string{
				"DB_URL":    "postgres://localhost/chirpy",
				"POLKA_KEY": "key",
			},
			wantErr:      "missing required environment variables: JWT_SECRET",
			wantPort:     "",
			wantPlatform: "",
		},
		{
			name:         "all required variables missing",
			env:          map[string]string{},
			wantErr:      "missing required environment variables: DB_URL, JWT_SECRET, POLKA_KEY",
			wantPort:     "",
			wantPlatform: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{"DB_URL", "JWT_SECRET", "POLKA_KEY", "PLATFORM", "PORT"} {
				t.Setenv(key, tt.env[key])
			}

			cfg, err := loadConfig()

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadConfig() expects error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadConfig() wantErr = %q, error = %v", tt.wantErr, err)
			}
			if cfg.dbURL != tt.env["DB_URL"] || cfg.jwtSecret != tt.env["JWT_SECRET"] || cfg.polkaKey != tt.env["POLKA_KEY"] {
				t.Errorf("loadConfig() expects values from the environment, got %+v", cfg)
			}
			if cfg.port != tt.wantPort {
				t.Errorf("loadConfig() expects port %q, got %q", tt.wantPort, cfg.port)
			}
			if cfg.platform != tt.wantPlatform {
				t.Errorf("loadConfig() expects platform %q, got %q", tt.wantPlatform, cfg.platform)
			}
		})
	}
}
