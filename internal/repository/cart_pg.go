package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Cart represents a shopping cart.
type Cart struct {
	ID        int64
	UserID    *int64
	SessionID *string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Items     []CartItem
}

// CartItem represents an item inside a cart.
type CartItem struct {
	ID                 int64
	CartID             int64
	ProductID          int64
	Quantity           int64
	PriceCentsSnapshot int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CartRepository provides cart operations.
type CartRepository interface {
	GetOrCreateActiveCart(ctx context.Context, userID *int64, sessionID *string) (*Cart, error)
	UpsertItem(ctx context.Context, cartID, productID int64, quantity int64, priceSnapshot int64) error
	RemoveItem(ctx context.Context, cartID, productID int64) error
	GetCartWithItems(ctx context.Context, cartID int64) (*Cart, error)
	ClearCart(ctx context.Context, cartID int64) error
}

type pgCartRepository struct {
	pool *pgxpool.Pool
}

// NewCartRepository creates a CartRepository backed by Postgres.
func NewCartRepository(pool *pgxpool.Pool) CartRepository {
	return &pgCartRepository{pool: pool}
}

func (r *pgCartRepository) GetOrCreateActiveCart(ctx context.Context, userID *int64, sessionID *string) (*Cart, error) {
	var cart *Cart
	if userID != nil {
		c, err := r.getActiveByUser(ctx, *userID)
		if err == nil {
			cart = c
		}
	}
	if cart == nil && sessionID != nil {
		c, err := r.getActiveBySession(ctx, *sessionID)
		if err == nil {
			cart = c
		}
	}

	if cart != nil {
		return cart, nil
	}

	const q = `
INSERT INTO carts (user_id, session_id, status)
VALUES ($1, $2, 'active')
RETURNING id, user_id, session_id, status, created_at, updated_at
`
	var uID any
	if userID != nil {
		uID = *userID
	}
	var sID any
	if sessionID != nil {
		sID = *sessionID
	}

	row := r.pool.QueryRow(ctx, q, uID, sID)
	var c Cart
	if err := row.Scan(&c.ID, &c.UserID, &c.SessionID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert cart: %w", err)
	}
	return &c, nil
}

func (r *pgCartRepository) getActiveByUser(ctx context.Context, userID int64) (*Cart, error) {
	const q = `
SELECT id, user_id, session_id, status, created_at, updated_at
FROM carts
WHERE user_id = $1 AND status = 'active'
`
	row := r.pool.QueryRow(ctx, q, userID)
	var c Cart
	if err := row.Scan(&c.ID, &c.UserID, &c.SessionID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *pgCartRepository) getActiveBySession(ctx context.Context, sessionID string) (*Cart, error) {
	const q = `
SELECT id, user_id, session_id, status, created_at, updated_at
FROM carts
WHERE session_id = $1 AND status = 'active'
`
	row := r.pool.QueryRow(ctx, q, sessionID)
	var c Cart
	if err := row.Scan(&c.ID, &c.UserID, &c.SessionID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *pgCartRepository) UpsertItem(ctx context.Context, cartID, productID int64, quantity int64, priceSnapshot int64) error {
	const q = `
INSERT INTO cart_items (cart_id, product_id, quantity, price_cents_snapshot)
VALUES ($1, $2, $3, $4)
ON CONFLICT (cart_id, product_id)
DO UPDATE SET quantity = EXCLUDED.quantity,
              price_cents_snapshot = EXCLUDED.price_cents_snapshot,
              updated_at = NOW()
`
	_, err := r.pool.Exec(ctx, q, cartID, productID, quantity, priceSnapshot)
	if err != nil {
		return fmt.Errorf("upsert cart item: %w", err)
	}
	return nil
}

func (r *pgCartRepository) RemoveItem(ctx context.Context, cartID, productID int64) error {
	const q = `
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2
`
	_, err := r.pool.Exec(ctx, q, cartID, productID)
	if err != nil {
		return fmt.Errorf("remove cart item: %w", err)
	}
	return nil
}

func (r *pgCartRepository) GetCartWithItems(ctx context.Context, cartID int64) (*Cart, error) {
	const cartQuery = `
SELECT id, user_id, session_id, status, created_at, updated_at
FROM carts
WHERE id = $1
`
	cartRow := r.pool.QueryRow(ctx, cartQuery, cartID)
	var c Cart
	if err := cartRow.Scan(&c.ID, &c.UserID, &c.SessionID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	const itemsQuery = `
SELECT id, cart_id, product_id, quantity, price_cents_snapshot, created_at, updated_at
FROM cart_items
WHERE cart_id = $1
`
	rows, err := r.pool.Query(ctx, itemsQuery, cartID)
	if err != nil {
		return nil, fmt.Errorf("query cart items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it CartItem
		if err := rows.Scan(&it.ID, &it.CartID, &it.ProductID, &it.Quantity, &it.PriceCentsSnapshot, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		c.Items = append(c.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cart items: %w", err)
	}

	return &c, nil
}

func (r *pgCartRepository) ClearCart(ctx context.Context, cartID int64) error {
	const q = `
DELETE FROM cart_items
WHERE cart_id = $1
`
	_, err := r.pool.Exec(ctx, q, cartID)
	if err != nil {
		return fmt.Errorf("clear cart items: %w", err)
	}
	return nil
}

