package usecase

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/marketplace/search/internal/domain"
)

const embeddingBatchSize = 64

type IndexService struct {
	index     domain.VectorIndex
	catalog   domain.CatalogClient
	embedding domain.EmbeddingClient
	indexing  atomic.Bool
}

func NewIndexService(idx domain.VectorIndex, cat domain.CatalogClient, emb domain.EmbeddingClient) *IndexService {
	return &IndexService{
		index:     idx,
		catalog:   cat,
		embedding: emb,
	}
}

func (s *IndexService) IsIndexing() bool {
	return s.indexing.Load()
}

func (s *IndexService) BuildIndex(ctx context.Context) error {
	if !s.indexing.CompareAndSwap(false, true) {
		return fmt.Errorf("indexing already in progress")
	}
	defer s.indexing.Store(false)

	products, err := s.catalog.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("fetch catalog: %w", err)
	}
	if len(products) == 0 {
		return fmt.Errorf("catalog is empty")
	}

	log.Printf("Building index for %d products...", len(products))

	if err := s.index.Reset(); err != nil {
		return fmt.Errorf("reset index: %w", err)
	}

	if err := s.index.Reserve(len(products)); err != nil {
		return fmt.Errorf("reserve index capacity: %w", err)
	}

	texts := make([]string, len(products))
	for i, p := range products {
		texts[i] = p.Name + " " + p.Category + " " + p.Description
	}

	for i := 0; i < len(texts); i += embeddingBatchSize {
		end := i + embeddingBatchSize
		if end > len(texts) {
			end = len(texts)
		}

		vectors, err := s.embedding.Embed(ctx, texts[i:end])
		if err != nil {
			return fmt.Errorf("embed batch %d-%d: %w", i, end, err)
		}

		for j, vec := range vectors {
			productID := uint64(products[i+j].ID)
			if err := s.index.Add(productID, vec); err != nil {
				return fmt.Errorf("add product %d to index: %w", productID, err)
			}
		}

		log.Printf("Indexed %d/%d products", end, len(texts))
	}

	log.Printf("Index built: %d vectors", s.index.Len())
	return nil
}
