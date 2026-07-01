package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketplace/gateway/config"
	catalogAdapter "github.com/marketplace/gateway/internal/adapters/catalog"
	embeddingAdapter "github.com/marketplace/gateway/internal/adapters/embedding"
	searchAdapter "github.com/marketplace/gateway/internal/adapters/search"
	transportHTTP "github.com/marketplace/gateway/internal/transport/http"
	"github.com/marketplace/gateway/internal/usecase"
)

func main() {
	cfg := config.Load()

	embeddingClient := embeddingAdapter.NewClient(cfg.EmbeddingURL)
	searchClient := searchAdapter.NewClient(cfg.SearchURL)
	catalogClient := catalogAdapter.NewClient(cfg.CatalogURL)

	svc := usecase.NewSearchService(embeddingClient, searchClient, catalogClient)
	handler := transportHTTP.NewHandler(svc, cfg.CatalogURL)
	router := transportHTTP.NewRouter(handler, cfg.FrontendDir)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Gateway listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
