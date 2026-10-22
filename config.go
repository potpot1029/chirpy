package main

import (
	"fmt"
	"os"
	"strings"
)

const defaultPort = "8080"

type config struct {
	dbURL     string
	jwtSecret string
	polkaKey  string
	platform  string
	port      string
}

func loadConfig() (config, error) {
	cfg := config{
		dbURL:     os.Getenv("DB_URL"),
		jwtSecret: os.Getenv("JWT_SECRET"),
		polkaKey:  os.Getenv("POLKA_KEY"),
		platform:  os.Getenv("PLATFORM"),
		port:      os.Getenv("PORT"),
	}

	var missing []string
	if cfg.dbURL == "" {
		missing = append(missing, "DB_URL")
	}
	if cfg.jwtSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.polkaKey == "" {
		missing = append(missing, "POLKA_KEY")
	}
	if len(missing) > 0 {
		return config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if cfg.port == "" {
		cfg.port = defaultPort
	}

	return cfg, nil
}
