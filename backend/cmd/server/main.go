package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/opencontext/backend/internal/api"
	"github.com/opencontext/backend/internal/config"
	"github.com/opencontext/backend/internal/graphiti"
	"github.com/opencontext/backend/internal/store"
)

func main() {
	cfg := config.Load()
	db, err := store.Connect(cfg.PostgresDSN, cfg.ProjectUUID)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	g := graphiti.New(cfg.GraphitiURL)
	a := &api.API{Cfg: cfg, DB: db, G: g}
	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: a.Handler(),
	}
	log.Printf("open-context backend listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
		os.Exit(1)
	}
}
