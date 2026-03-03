package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserProvider represents a row in user_providers.
type UserProvider struct {
	ID             int64
	UserID         int64
	Provider       string
	ProviderUserID string
}

var ErrUserProviderNotFound = errors.New("user provider not found")

// UserProviderRepository persists OAuth/social provider links.
type UserProviderRepository interface {
	GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*UserProvider, error)
	Create(ctx context.Context, link *UserProvider) error
}

type pgUserProviderRepository struct {
	pool *pgxpool.Pool
}

// NewUserProviderRepository creates a Postgres-backed UserProviderRepository.
func NewUserProviderRepository(pool *pgxpool.Pool) UserProviderRepository {
	return &pgUserProviderRepository{pool: pool}
}

func (r *pgUserProviderRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*UserProvider, error) {
	const q = `
SELECT id, user_id, provider, provider_user_id
FROM user_providers
WHERE provider = $1 AND provider_user_id = $2
`
	row := r.pool.QueryRow(ctx, q, provider, providerUserID)
	var up UserProvider
	if err := row.Scan(&up.ID, &up.UserID, &up.Provider, &up.ProviderUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserProviderNotFound
		}
		return nil, fmt.Errorf("scan user provider: %w", err)
	}
	return &up, nil
}

func (r *pgUserProviderRepository) Create(ctx context.Context, link *UserProvider) error {
	const q = `
INSERT INTO user_providers (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING id
`
	if err := r.pool.QueryRow(ctx, q, link.UserID, link.Provider, link.ProviderUserID).Scan(&link.ID); err != nil {
		return fmt.Errorf("insert user provider: %w", err)
	}
	return nil
}

