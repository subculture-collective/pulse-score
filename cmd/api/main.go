package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/onnwee/pulse-score/internal/config"
)

func main() {
	cfg := config.Load()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting PulseScore API on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Printf("Server error: %v", err)
		os.Exit(1)
	}
}
