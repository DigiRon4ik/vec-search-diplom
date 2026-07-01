package domain

import "context"

// Product represents a catalog product entity.
type Product struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Lang        string `json:"lang"`
}

type ProductFilter struct {
	Category string
	Limit    int
	Offset   int
}

// ProductRepository defines the interface for product persistence operations.
type ProductRepository interface {
	FindAll(ctx context.Context) ([]Product, error)
	FindFiltered(ctx context.Context, f ProductFilter) ([]Product, int64, error)
	FindByID(ctx context.Context, id int64) (*Product, error)
	FindByIDs(ctx context.Context, ids []int64) ([]Product, error)
	FindCategories(ctx context.Context) ([]string, error)
	Create(ctx context.Context, p *Product) error
	Count(ctx context.Context) (int64, error)
	CreateMany(ctx context.Context, products []Product) error
}
