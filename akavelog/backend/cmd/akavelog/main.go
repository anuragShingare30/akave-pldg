package main

import (
	"context"
	"log"
	"os"

	"github.com/akave-ai/akavelog/internal/config"
	"github.com/akave-ai/akavelog/internal/database"
	"github.com/akave-ai/akavelog/internal/server"
)

func main() {
	cfg := config.LoadApp()

	// Run from backend directory so migrations path internal/database/migrations resolves.
	baseDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}
	if err := database.RunMigrations(cfg.DatabaseURL, baseDir); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database pool: %v", err)
	}
	defer pool.Close()

	srv := server.New(cfg, pool)
	if err := srv.Start(ctx); err != nil {
		log.Printf("server exited: %v", err)
		os.Exit(1)
	}
}
