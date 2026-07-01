package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/marketplace/catalog/internal/domain"
)

// ProductService implements business logic for product operations.
type ProductService struct {
	repo domain.ProductRepository
}

// NewProductService creates a new ProductService with the given repository.
func NewProductService(repo domain.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

// ListProducts returns all products from the repository.
func (s *ProductService) ListProducts(ctx context.Context) ([]domain.Product, error) {
	return s.repo.FindAll(ctx)
}

// GetProduct returns a single product by its ID.
func (s *ProductService) GetProduct(ctx context.Context, id int64) (*domain.Product, error) {
	return s.repo.FindByID(ctx, id)
}

// CreateProduct persists a new product and returns it with the assigned ID.
func (s *ProductService) CreateProduct(ctx context.Context, p *domain.Product) error {
	return s.repo.Create(ctx, p)
}

// GetProductsByIDs returns products matching the given IDs.
func (s *ProductService) GetProductsByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	return s.repo.FindByIDs(ctx, ids)
}

// ListFiltered returns paginated products with optional category filter.
func (s *ProductService) ListFiltered(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int64, error) {
	return s.repo.FindFiltered(ctx, f)
}

// ListCategories returns all distinct categories.
func (s *ProductService) ListCategories(ctx context.Context) ([]string, error) {
	return s.repo.FindCategories(ctx)
}

// SeedFromFile reads a JSON array of products from filePath and inserts them
// into the database. Seeding is skipped if the products table is not empty.
func (s *ProductService) SeedFromFile(ctx context.Context, filePath string) error {
	count, err := s.repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("seed: count products: %w", err)
	}
	if count > 0 {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("seed: read file %s: %w", filePath, err)
	}

	var products []domain.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return fmt.Errorf("seed: unmarshal json: %w", err)
	}

	if len(products) == 0 {
		return nil
	}

	if err := s.repo.CreateMany(ctx, products); err != nil {
		return fmt.Errorf("seed: create many: %w", err)
	}

	return nil
}
