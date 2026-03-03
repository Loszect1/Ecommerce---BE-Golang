package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderPaymentRepository updates order payment fields.
type OrderPaymentRepository interface {
	UpdatePayment(ctx context.Context, tx pgx.Tx, orderID int64, orderStatus, paymentStatus, provider, reference string) error
}

type pgOrderPaymentRepository struct{}

// NewOrderPaymentRepository creates an OrderPaymentRepository backed by Postgres.
func NewOrderPaymentRepository(_ *pgxpool.Pool) OrderPaymentRepository {
	// Stateless; uses tx passed in.
	return &pgOrderPaymentRepository{}
}

func (r *pgOrderPaymentRepository) UpdatePayment(ctx context.Context, tx pgx.Tx, orderID int64, orderStatus, paymentStatus, provider, reference string) error {
	if orderID <= 0 {
		return fmt.Errorf("order_id is required")
	}
	if strings.TrimSpace(paymentStatus) == "" {
		return fmt.Errorf("payment_status is required")
	}
	if strings.TrimSpace(provider) == "" {
		return fmt.Errorf("provider is required")
	}

	var orderStatusAny any
	if strings.TrimSpace(orderStatus) != "" {
		orderStatusAny = orderStatus
	}
	var referenceAny any
	if strings.TrimSpace(reference) != "" {
		referenceAny = reference
	}

	const q = `
UPDATE orders
SET payment_status = $1,
    payment_provider = $2,
    payment_reference = $3,
    status = COALESCE($4, status),
    updated_at = NOW()
WHERE id = $5
`
	tag, err := tx.Exec(ctx, q, paymentStatus, provider, referenceAny, orderStatusAny, orderID)
	if err != nil {
		return fmt.Errorf("update order payment fields: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("order not found")
	}
	return nil
}

