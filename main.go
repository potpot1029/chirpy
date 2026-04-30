package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	log.Printf("starting http server at port %s...\n", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[error] http server ListenAndServe: %v", err)
	}
}
