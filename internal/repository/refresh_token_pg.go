package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshToken represents a row in refresh_tokens.
type RefreshToken struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// RefreshTokenStore persists refresh tokens.
type RefreshTokenStore interface {
	Insert(ctx context.Context, t *RefreshToken) error
	Revoke(ctx context.Context, token string) error
	GetValid(ctx context.Context, token string) (*RefreshToken, error)
}

type pgRefreshTokenStore struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenStore creates a Postgres-backed RefreshTokenStore.
func NewRefreshTokenStore(pool *pgxpool.Pool) RefreshTokenStore {
	return &pgRefreshTokenStore{pool: pool}
}

func (s *pgRefreshTokenStore) Insert(ctx context.Context, t *RefreshToken) error {
	const q = `
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING id, created_at
`
	row := s.pool.QueryRow(ctx, q, t.UserID, t.Token, t.ExpiresAt)
	if err := row.Scan(&t.ID, &t.CreatedAt); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (s *pgRefreshTokenStore) Revoke(ctx context.Context, token string) error {
	const q = `
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token = $1 AND revoked_at IS NULL
`
	_, err := s.pool.Exec(ctx, q, token)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (s *pgRefreshTokenStore) GetValid(ctx context.Context, token string) (*RefreshToken, error) {
	const q = `
SELECT id, user_id, token, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token = $1
  AND revoked_at IS NULL
  AND expires_at > NOW()
`
	row := s.pool.QueryRow(ctx, q, token)

	var t RefreshToken
	if err := row.Scan(&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt); err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &t, nil
}

