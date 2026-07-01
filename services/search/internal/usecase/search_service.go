package usecase

import (
	"fmt"

	"github.com/marketplace/search/internal/domain"
)

type SearchService struct {
	index domain.VectorIndex
}

func NewSearchService(idx domain.VectorIndex) *SearchService {
	return &SearchService{index: idx}
}

func (s *SearchService) Search(vector []float32, k int) ([]domain.Neighbor, error) {
	if k <= 0 {
		k = 10
	}
	if s.index.Len() == 0 {
		return nil, fmt.Errorf("index is empty, run indexing first")
	}
	return s.index.Search(vector, k)
}

func (s *SearchService) Stats() domain.IndexStats {
	return domain.IndexStats{
		Size: s.index.Len(),
	}
}
