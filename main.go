package main

import (
	"database/sql"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/potpot1029/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	jwtSecret      string
	polkaKey       string
}

func main() {
	godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	apiCfg := apiConfig{
		dbQueries: database.New(db),
		platform:  cfg.platform,
		jwtSecret: cfg.jwtSecret,
		polkaKey:  cfg.polkaKey,
	}

	server := http.Server{
		Handler: apiCfg.routes(),
		Addr:    ":" + cfg.port,
	}
	log.Printf("serving files from %s at /app/ on port %s", filepathRoot, cfg.port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[error] http server ListenAndServe: %v", err)
	}
}
