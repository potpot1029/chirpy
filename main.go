package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8080"
	const filepathRoot = "."

	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot))))
	mux.HandleFunc("/healthz", handlerReadiness)

	server := http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("serving file from %s for http server at port %s...\n", filepathRoot, port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[error] http server ListenAndServe: %v", err)
	}
}

func handlerReadiness(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
