package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Product represents a product row.
type Product struct {
	ID           int64
	Slug         string
	Name         string
	Description  string
	PriceCents   int64
	CurrencyCode string
	MainImageURL *string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ProductRepository provides access to products.
type ProductRepository interface {
	List(ctx context.Context, limit, offset int) ([]Product, error)
	ListAll(ctx context.Context, limit, offset int) ([]Product, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	Create(ctx context.Context, p *Product, initialStock int64) error
	Update(ctx context.Context, p *Product) error
	Deactivate(ctx context.Context, id int64) error
}

type pgProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository creates a Postgres-backed ProductRepository.
func NewProductRepository(pool *pgxpool.Pool) ProductRepository {
	return &pgProductRepository{pool: pool}
}

func (r *pgProductRepository) List(ctx context.Context, limit, offset int) ([]Product, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
SELECT id, slug, name, description, price_cents, currency_code, main_image_url, is_active, created_at, updated_at
FROM products
WHERE is_active = TRUE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2
`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.PriceCents, &p.CurrencyCode, &p.MainImageURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, nil
}

func (r *pgProductRepository) ListAll(ctx context.Context, limit, offset int) ([]Product, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
SELECT id, slug, name, description, price_cents, currency_code, main_image_url, is_active, created_at, updated_at
FROM products
ORDER BY created_at DESC
LIMIT $1 OFFSET $2
`
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.PriceCents, &p.CurrencyCode, &p.MainImageURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, nil
}

func (r *pgProductRepository) GetByID(ctx context.Context, id int64) (*Product, error) {
	const q = `
SELECT id, slug, name, description, price_cents, currency_code, main_image_url, is_active, created_at, updated_at
FROM products
WHERE id = $1
`
	row := r.pool.QueryRow(ctx, q, id)

	var p Product
	if err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.PriceCents, &p.CurrencyCode, &p.MainImageURL, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("scan product by id: %w", err)
	}

	return &p, nil
}

func (r *pgProductRepository) Create(ctx context.Context, p *Product, initialStock int64) error {
	if p == nil {
		return fmt.Errorf("product is required")
	}
	if p.Slug == "" || p.Name == "" {
		return fmt.Errorf("slug and name are required")
	}
	if p.PriceCents < 0 {
		return fmt.Errorf("price_cents must be >= 0")
	}
	if p.CurrencyCode == "" {
		p.CurrencyCode = "USD"
	}
	if initialStock < 0 {
		return fmt.Errorf("initial_stock must be >= 0")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const q = `
INSERT INTO products (slug, name, description, price_cents, currency_code, main_image_url, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at
`
	if err := tx.QueryRow(ctx, q, p.Slug, p.Name, p.Description, p.PriceCents, p.CurrencyCode, p.MainImageURL, p.IsActive).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return fmt.Errorf("insert product: %w", err)
	}

	const invQ = `
INSERT INTO inventory (product_id, stock, reserved_stock, version)
VALUES ($1, $2, 0, 1)
`
	if _, err := tx.Exec(ctx, invQ, p.ID, initialStock); err != nil {
		return fmt.Errorf("insert inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *pgProductRepository) Update(ctx context.Context, p *Product) error {
	if p == nil {
		return fmt.Errorf("product is required")
	}
	if p.ID <= 0 {
		return fmt.Errorf("id is required")
	}
	if p.Slug == "" || p.Name == "" {
		return fmt.Errorf("slug and name are required")
	}
	if p.PriceCents < 0 {
		return fmt.Errorf("price_cents must be >= 0")
	}
	if p.CurrencyCode == "" {
		p.CurrencyCode = "USD"
	}

	const q = `
UPDATE products
SET slug = $1,
    name = $2,
    description = $3,
    price_cents = $4,
    currency_code = $5,
    main_image_url = $6,
    is_active = $7,
    updated_at = NOW()
WHERE id = $8
RETURNING updated_at
`
	if err := r.pool.QueryRow(ctx, q, p.Slug, p.Name, p.Description, p.PriceCents, p.CurrencyCode, p.MainImageURL, p.IsActive, p.ID).Scan(&p.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("update product: %w", err)
	}
	return nil
}

func (r *pgProductRepository) Deactivate(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id is required")
	}
	const q = `
UPDATE products
SET is_active = FALSE,
    updated_at = NOW()
WHERE id = $1
`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deactivate product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("product not found")
	}
	return nil
}

