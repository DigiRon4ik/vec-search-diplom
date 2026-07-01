package domain

import "context"

type Product struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Lang        string `json:"lang"`
}

type Neighbor struct {
	ID       int64   `json:"id"`
	Distance float32 `json:"distance"`
	Score    float32 `json:"score"`
}

type SearchRequest struct {
	Vector []float32 `json:"vector"`
	K      int       `json:"k"`
}

type SearchResponse struct {
	Results []Neighbor `json:"results"`
}

type IndexStats struct {
	Size       int  `json:"size"`
	IsIndexing bool `json:"is_indexing"`
}

type VectorIndex interface {
	Add(key uint64, vector []float32) error
	Search(vector []float32, k int) ([]Neighbor, error)
	Len() int
	Reset() error
	Reserve(capacity int) error
}

type CatalogClient interface {
	GetAll(ctx context.Context) ([]Product, error)
}

type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
