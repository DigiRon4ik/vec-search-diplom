package domain

type Product struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Lang        string `json:"lang"`
}

type SearchResult struct {
	Product  Product `json:"product"`
	Distance float32 `json:"distance"`
	Score    float32 `json:"score"`
}

type SearchRequest struct {
	Query string `json:"query"`
	K     int    `json:"k"`
}

type EmbedResponse struct {
	Vectors [][]float32 `json:"vectors"`
}

type VectorSearchRequest struct {
	Vector []float32 `json:"vector"`
	K      int       `json:"k"`
}

type Neighbor struct {
	ID       int64   `json:"id"`
	Distance float32 `json:"distance"`
	Score    float32 `json:"score"`
}

type VectorSearchResponse struct {
	Results []Neighbor `json:"results"`
}

type CatalogResponse struct {
	Products []Product `json:"products"`
}
