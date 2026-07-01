package usecase

import (
	"context"
	"fmt"

	"github.com/marketplace/gateway/internal/domain"
)

type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type SearchClient interface {
	Search(ctx context.Context, vector []float32, k int) ([]domain.Neighbor, error)
	Reindex(ctx context.Context) error
}

type CatalogClient interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.Product, error)
}

type SearchService struct {
	embedding EmbeddingClient
	search    SearchClient
	catalog   CatalogClient
}

func NewSearchService(e EmbeddingClient, s SearchClient, c CatalogClient) *SearchService {
	return &SearchService{embedding: e, search: s, catalog: c}
}

func (s *SearchService) Search(ctx context.Context, query string, k int) ([]domain.SearchResult, error) {
	if k <= 0 {
		k = 10
	}

	vectors, err := s.embedding.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("embedding returned empty result")
	}

	neighbors, err := s.search.Search(ctx, vectors[0], k)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	if len(neighbors) == 0 {
		return nil, nil
	}

	ids := make([]int64, len(neighbors))
	for i, n := range neighbors {
		ids[i] = n.ID
	}

	products, err := s.catalog.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch products: %w", err)
	}

	results := make([]domain.SearchResult, 0, len(neighbors))
	for _, n := range neighbors {
		if p, ok := products[n.ID]; ok {
			results = append(results, domain.SearchResult{
				Product:  p,
				Distance: n.Distance,
				Score:    n.Score,
			})
		}
	}
	return results, nil
}

func (s *SearchService) Reindex(ctx context.Context) error {
	return s.search.Reindex(ctx)
}
