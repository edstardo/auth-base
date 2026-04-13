package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/edstardo/auth-base/pkg/auth"
	"github.com/edstardo/auth-base/pkg/config"
	"github.com/edstardo/auth-base/pkg/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	database, err := db.NewPostgresConnection(cfg.DB)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/signup", auth.SignupHandler(database))

	addr := ":" + cfg.Service.Port
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
