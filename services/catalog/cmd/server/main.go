package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/marketplace/catalog/config"
	"github.com/marketplace/catalog/internal/repository"
	transporthttp "github.com/marketplace/catalog/internal/transport/http"
	"github.com/marketplace/catalog/internal/usecase"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	repo := repository.NewPostgresRepository(db)

	if err := repo.Migrate(ctx); err != nil {
		log.Fatalf("failed to run migration: %v", err)
	}

	svc := usecase.NewProductService(repo)

	if cfg.SeedFile != "" {
		if err := svc.SeedFromFile(ctx, cfg.SeedFile); err != nil {
			log.Fatalf("failed to seed data: %v", err)
		}
		log.Printf("seed completed from %s", cfg.SeedFile)
	}

	handler := transporthttp.NewHandler(svc)
	router := transporthttp.NewRouter(handler)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("catalog service listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server stopped")
}
