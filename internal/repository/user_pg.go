package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User represents a row in the users table.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	FullName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var ErrUserNotFound = errors.New("user not found")

// UserRepository defines persistence operations for users.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, user *User) error
}

type pgUserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a Postgres-backed UserRepository.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &pgUserRepository{pool: pool}
}

func (r *pgUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	const q = `
SELECT id, email, password_hash, full_name, is_active, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1)
`
	row := r.pool.QueryRow(ctx, q, email)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user by email: %w", err)
	}

	return &u, nil
}

func (r *pgUserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	const q = `
SELECT id, email, password_hash, full_name, is_active, created_at, updated_at
FROM users
WHERE id = $1
`
	row := r.pool.QueryRow(ctx, q, id)

	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user by id: %w", err)
	}

	return &u, nil
}

func (r *pgUserRepository) Create(ctx context.Context, user *User) error {
	const q = `
INSERT INTO users (email, password_hash, full_name, is_active)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, updated_at
`
	row := r.pool.QueryRow(ctx, q, user.Email, user.PasswordHash, user.FullName, user.IsActive)
	if err := row.Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

