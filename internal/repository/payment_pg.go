package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Payment represents a row in payments.
type Payment struct {
	ID                int64
	OrderID           int64
	Provider          string
	ProviderPaymentID *string
	AmountCents       int64
	CurrencyCode      string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// PaymentRepository provides persistence for payments and payment_logs.
type PaymentRepository interface {
	UpsertForOrder(ctx context.Context, tx pgx.Tx, p *Payment) (*Payment, error)
	InsertLogIfNew(ctx context.Context, tx pgx.Tx, paymentID int64, eventType, providerEventID string, rawPayload json.RawMessage) (bool, error)
}

type pgPaymentRepository struct{}

// NewPaymentRepository creates a PaymentRepository backed by Postgres.
func NewPaymentRepository(_ *pgxpool.Pool) PaymentRepository {
	// We keep this stateless and accept tx in methods.
	return &pgPaymentRepository{}
}

func (r *pgPaymentRepository) UpsertForOrder(ctx context.Context, tx pgx.Tx, p *Payment) (*Payment, error) {
	if p == nil {
		return nil, fmt.Errorf("payment is required")
	}
	if p.OrderID <= 0 {
		return nil, fmt.Errorf("order_id is required")
	}
	if strings.TrimSpace(p.Provider) == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(p.CurrencyCode) == "" {
		p.CurrencyCode = "USD"
	}
	if strings.TrimSpace(p.Status) == "" {
		return nil, fmt.Errorf("status is required")
	}
	if p.AmountCents < 0 {
		return nil, fmt.Errorf("amount_cents must be >= 0")
	}

	const q = `
INSERT INTO payments (order_id, provider, provider_payment_id, amount_cents, currency_code, status)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (order_id)
DO UPDATE SET
  provider = EXCLUDED.provider,
  provider_payment_id = EXCLUDED.provider_payment_id,
  amount_cents = EXCLUDED.amount_cents,
  currency_code = EXCLUDED.currency_code,
  status = EXCLUDED.status,
  updated_at = NOW()
RETURNING id, created_at, updated_at
`
	var providerPaymentID any
	if p.ProviderPaymentID != nil {
		providerPaymentID = *p.ProviderPaymentID
	}

	if err := tx.QueryRow(ctx, q, p.OrderID, p.Provider, providerPaymentID, p.AmountCents, p.CurrencyCode, p.Status).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("upsert payment: %w", err)
	}
	return p, nil
}

func (r *pgPaymentRepository) InsertLogIfNew(ctx context.Context, tx pgx.Tx, paymentID int64, eventType, providerEventID string, rawPayload json.RawMessage) (bool, error) {
	if paymentID <= 0 {
		return false, fmt.Errorf("payment_id is required")
	}
	if strings.TrimSpace(eventType) == "" {
		return false, fmt.Errorf("event_type is required")
	}
	if strings.TrimSpace(providerEventID) == "" {
		return false, fmt.Errorf("provider_event_id is required")
	}
	if len(rawPayload) == 0 {
		rawPayload = json.RawMessage(`{}`)
	}

	const q = `
INSERT INTO payment_logs (payment_id, event_type, raw_payload, provider_event_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (provider_event_id) DO NOTHING
RETURNING id
`
	var id int64
	err := tx.QueryRow(ctx, q, paymentID, eventType, rawPayload, providerEventID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("insert payment log: %w", err)
	}
	return true, nil
}

