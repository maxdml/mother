package main

import (
	"log"
	"net/http"

	"mother/control-plane/api"
)

const (
	listenAddr     = ":8080"
	coderBinaryPath = "/Users/mother/mother/coder/coder"
)

func main() {
	store := NewJobStore()
	service := NewCoderService(coderBinaryPath)
	handler := &Handler{Store: store, Service: service}

	mux := api.Handler(handler)

	log.Printf("control-plane listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
