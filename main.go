package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8080"
	const filepathRoot = "."

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(filepathRoot)))

	server := http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}

	log.Printf("serving file from %s for http server at port %s...\n", filepathRoot, port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[error] http server ListenAndServe: %v", err)
	}
}
