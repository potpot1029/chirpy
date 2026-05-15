package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8080"
	const filepathRoot = "."

	apiCfg := apiConfig{}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot)))))

	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidate)

	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetric)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	server := http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("serving file from %s for http server at port %s...\n", filepathRoot, port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[error] http server ListenAndServe: %v", err)
	}
}
