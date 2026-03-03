package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Category represents a category row.
type Category struct {
	ID        int64
	Name      string
	Slug      string
	ParentID  *int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CategoryRepository provides access to categories.
type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
}

type pgCategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository creates a Postgres-backed CategoryRepository.
func NewCategoryRepository(pool *pgxpool.Pool) CategoryRepository {
	return &pgCategoryRepository{pool: pool}
}

func (r *pgCategoryRepository) List(ctx context.Context) ([]Category, error) {
	const q = `
SELECT id, name, slug, parent_id, created_at, updated_at
FROM categories
ORDER BY name ASC
`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.ParentID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return out, nil
}

