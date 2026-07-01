package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketplace/search/config"
	catalogAdapter "github.com/marketplace/search/internal/adapters/catalog"
	embeddingAdapter "github.com/marketplace/search/internal/adapters/embedding"
	usearchAdapter "github.com/marketplace/search/internal/adapters/usearch"
	transportHTTP "github.com/marketplace/search/internal/transport/http"
	"github.com/marketplace/search/internal/usecase"
)

func main() {
	cfg := config.Load()

	vectorIndex, err := usearchAdapter.NewIndex()
	if err != nil {
		log.Fatalf("Failed to create vector index: %v", err)
	}
	defer vectorIndex.Close()

	catalogClient := catalogAdapter.NewClient(cfg.CatalogURL)
	embeddingClient := embeddingAdapter.NewClient(cfg.EmbeddingURL)

	indexSvc := usecase.NewIndexService(vectorIndex, catalogClient, embeddingClient)
	searchSvc := usecase.NewSearchService(vectorIndex)
	handler := transportHTTP.NewHandler(indexSvc, searchSvc)
	router := transportHTTP.NewRouter(handler)

	go func() {
		log.Println("Starting auto-indexing...")
		for attempt := 1; attempt <= 30; attempt++ {
			err := indexSvc.BuildIndex(context.Background())
			if err == nil {
				log.Println("Auto-indexing completed successfully")
				return
			}
			log.Printf("Auto-index attempt %d failed: %v", attempt, err)
			time.Sleep(5 * time.Second)
		}
		log.Println("WARNING: Auto-indexing failed after 30 attempts. Use POST /index to retry.")
	}()

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Search service listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down search service...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
