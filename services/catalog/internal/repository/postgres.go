package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/marketplace/catalog/internal/domain"
)

// PostgresRepository implements domain.ProductRepository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Migrate creates the products table if it does not exist.
func (r *PostgresRepository) Migrate(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS products (
			id          BIGSERIAL PRIMARY KEY,
			name        TEXT NOT NULL,
			category    TEXT NOT NULL,
			description TEXT NOT NULL,
			lang        TEXT NOT NULL DEFAULT 'ru'
		);
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// FindFiltered returns products with optional category filter, pagination, and deterministic pseudo-random order.
func (r *PostgresRepository) FindFiltered(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int64, error) {
	countQuery := "SELECT COUNT(*) FROM products"
	dataQuery := "SELECT id, name, category, description, lang FROM products"
	var args []any
	argIdx := 1

	if f.Category != "" {
		where := fmt.Sprintf(" WHERE category = $%d", argIdx)
		countQuery += where
		dataQuery += where
		args = append(args, f.Category)
		argIdx++
	}

	var total int64
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("find filtered count: %w", err)
	}

	dataQuery += " ORDER BY md5(id::text)"

	if f.Limit > 0 {
		dataQuery += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, f.Limit)
		argIdx++
	}
	if f.Offset > 0 {
		dataQuery += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, f.Offset)
	}

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("find filtered: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Description, &p.Lang); err != nil {
			return nil, 0, fmt.Errorf("find filtered scan: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

// FindCategories returns distinct category names.
func (r *PostgresRepository) FindCategories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT DISTINCT category FROM products ORDER BY category")
	if err != nil {
		return nil, fmt.Errorf("find categories: %w", err)
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("find categories scan: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// FindAll returns all products.
func (r *PostgresRepository) FindAll(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, name, category, description, lang FROM products ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("find all: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Description, &p.Lang); err != nil {
			return nil, fmt.Errorf("find all scan: %w", err)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// FindByID returns a single product by ID.
func (r *PostgresRepository) FindByID(ctx context.Context, id int64) (*domain.Product, error) {
	var p domain.Product
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, category, description, lang FROM products WHERE id = $1", id,
	).Scan(&p.ID, &p.Name, &p.Category, &p.Description, &p.Lang)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}
	return &p, nil
}

// FindByIDs returns products matching any of the given IDs.
func (r *PostgresRepository) FindByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		"SELECT id, name, category, description, lang FROM products WHERE id IN (%s) ORDER BY id",
		strings.Join(placeholders, ", "),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find by ids: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Description, &p.Lang); err != nil {
			return nil, fmt.Errorf("find by ids scan: %w", err)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

// Create inserts a single product and populates its ID.
func (r *PostgresRepository) Create(ctx context.Context, p *domain.Product) error {
	return r.db.QueryRowContext(ctx,
		"INSERT INTO products (name, category, description, lang) VALUES ($1, $2, $3, $4) RETURNING id",
		p.Name, p.Category, p.Description, p.Lang,
	).Scan(&p.ID)
}

// Count returns the total number of products.
func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM products").Scan(&count)
	return count, err
}

// CreateMany inserts multiple products in a single transaction using batched inserts.
func (r *PostgresRepository) CreateMany(ctx context.Context, products []domain.Product) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create many begin tx: %w", err)
	}
	defer tx.Rollback()

	const batchSize = 500
	for i := 0; i < len(products); i += batchSize {
		end := i + batchSize
		if end > len(products) {
			end = len(products)
		}
		batch := products[i:end]

		placeholders := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*4)
		for j, p := range batch {
			base := j * 4
			placeholders = append(placeholders, fmt.Sprintf(
				"($%d, $%d, $%d, $%d)",
				base+1, base+2, base+3, base+4,
			))
			args = append(args, p.Name, p.Category, p.Description, p.Lang)
		}

		query := fmt.Sprintf(
			"INSERT INTO products (name, category, description, lang) VALUES %s",
			strings.Join(placeholders, ", "),
		)

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("create many batch insert: %w", err)
		}
	}

	return tx.Commit()
}
