package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Inventory represents stock information for a product.
type Inventory struct {
	ProductID     int64
	Stock         int64
	ReservedStock int64
	Version       int64
	UpdatedAt     time.Time
}

// InventoryRepository provides operations for inventory with pessimistic locking.
type InventoryRepository interface {
	GetForUpdate(ctx context.Context, tx pgx.Tx, productID int64) (*Inventory, error)
	Update(ctx context.Context, tx pgx.Tx, inv *Inventory) error
}

type pgInventoryRepository struct {
	pool *pgxpool.Pool
}

// NewInventoryRepository creates a Postgres-backed InventoryRepository.
func NewInventoryRepository(pool *pgxpool.Pool) InventoryRepository {
	return &pgInventoryRepository{pool: pool}
}

func (r *pgInventoryRepository) GetForUpdate(ctx context.Context, tx pgx.Tx, productID int64) (*Inventory, error) {
	const q = `
SELECT product_id, stock, reserved_stock, version, updated_at
FROM inventory
WHERE product_id = $1
FOR UPDATE
`
	row := tx.QueryRow(ctx, q, productID)
	var inv Inventory
	if err := row.Scan(&inv.ProductID, &inv.Stock, &inv.ReservedStock, &inv.Version, &inv.UpdatedAt); err != nil {
		return nil, fmt.Errorf("get inventory for update: %w", err)
	}
	return &inv, nil
}

func (r *pgInventoryRepository) Update(ctx context.Context, tx pgx.Tx, inv *Inventory) error {
	const q = `
UPDATE inventory
SET stock = $1,
    reserved_stock = $2,
    version = version + 1,
    updated_at = NOW()
WHERE product_id = $3
`
	_, err := tx.Exec(ctx, q, inv.Stock, inv.ReservedStock, inv.ProductID)
	if err != nil {
		return fmt.Errorf("update inventory: %w", err)
	}
	return nil
}

