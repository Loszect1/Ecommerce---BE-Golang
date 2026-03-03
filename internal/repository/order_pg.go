package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Order represents an order header.
type Order struct {
	ID                 int64
	UserID             *int64
	Status             string
	TotalAmountCents   int64
	CurrencyCode       string
	PaymentStatus      string
	PaymentProvider    *string
	PaymentReference   *string
	PlacedAt           *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// OrderItem represents a single order item.
type OrderItem struct {
	ID               int64
	OrderID          int64
	ProductID        int64
	Quantity         int64
	UnitPriceCents   int64
	TotalPriceCents  int64
}

// OrderRepository persists orders and order_items inside a transaction.
type OrderRepository interface {
	Create(ctx context.Context, tx pgx.Tx, order *Order, items []OrderItem) error
}

type pgOrderRepository struct{}

// NewOrderRepository creates an OrderRepository backed by Postgres.
func NewOrderRepository() OrderRepository {
	return &pgOrderRepository{}
}

func (r *pgOrderRepository) Create(ctx context.Context, tx pgx.Tx, order *Order, items []OrderItem) error {
	const insertOrder = `
INSERT INTO orders (user_id, status, total_amount_cents, currency_code, payment_status, payment_provider, payment_reference, placed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at
`

	var userID any
	if order.UserID != nil {
		userID = *order.UserID
	}

	if err := tx.QueryRow(
		ctx,
		insertOrder,
		userID,
		order.Status,
		order.TotalAmountCents,
		order.CurrencyCode,
		order.PaymentStatus,
		order.PaymentProvider,
		order.PaymentReference,
		order.PlacedAt,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	const insertItem = `
INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents, total_price_cents)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`

	for i := range items {
		item := &items[i]
		item.OrderID = order.ID
		if err := tx.QueryRow(ctx, insertItem, item.OrderID, item.ProductID, item.Quantity, item.UnitPriceCents, item.TotalPriceCents).Scan(&item.ID); err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	return nil
}

