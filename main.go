package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("SERVICE_PORT")
	if port == "" {
		port = "8000"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	addr := ":" + port
	log.Printf("auth service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Auth service running",
	})
}
