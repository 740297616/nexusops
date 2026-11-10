package main

import (
	"log"
	"net/http"
	"time"

	"github.com/740297616/nexusops/server/internal/api"
	"github.com/740297616/nexusops/server/internal/config"
)

func main() {
	cfg := config.Load()
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           api.NewRouter(cfg.Version),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("starting server on %s", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
