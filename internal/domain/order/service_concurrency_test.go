package order

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

var schemaOnce sync.Once

func TestCreateFromCart_OversellConcurrency(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not set (integration test)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool := newTestPool(t, ctx, dsn)
	t.Cleanup(pool.Close)

	ensureSchema(t, ctx, pool)
	resetTestData(t, ctx, pool)

	// Seed: 1 product, stock=1, two carts each want quantity=1.
	var productID int64
	if err := pool.QueryRow(ctx, `
INSERT INTO products (slug, name, description, price_cents, currency_code, is_active)
VALUES ('p-oversell', 'Oversell Test Product', 'test', 1000, 'USD', TRUE)
RETURNING id
`, nil).Scan(&productID); err != nil {
		t.Fatalf("insert product: %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO inventory (product_id, stock, reserved_stock)
VALUES ($1, 1, 0)
`, productID); err != nil {
		t.Fatalf("insert inventory: %v", err)
	}

	cartIDs := make([]int64, 0, 2)
	for i := 0; i < 2; i++ {
		var cartID int64
		if err := pool.QueryRow(ctx, `
INSERT INTO carts (status)
VALUES ('active')
RETURNING id
`).Scan(&cartID); err != nil {
			t.Fatalf("insert cart: %v", err)
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO cart_items (cart_id, product_id, quantity, price_cents_snapshot)
VALUES ($1, $2, 1, 1000)
`, cartID, productID); err != nil {
			t.Fatalf("insert cart item: %v", err)
		}
		cartIDs = append(cartIDs, cartID)
	}

	cartRepo := repository.NewCartRepository(pool)
	invRepo := repository.NewInventoryRepository(pool)
	orderRepo := repository.NewOrderRepository()
	log := logger.New()
	svc := NewService(pool, invRepo, orderRepo, cartRepo, log)

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	type result struct {
		orderID int64
		err     error
	}
	results := make(chan result, 2)

	for _, cartID := range cartIDs {
		go func(id int64) {
			defer wg.Done()
			<-start
			res, err := svc.CreateFromCart(ctx, CreateOrderRequest{CartID: id, Currency: "USD"})
			if err != nil {
				results <- result{err: err}
				return
			}
			results <- result{orderID: res.OrderID}
		}(cartID)
	}

	close(start)
	wg.Wait()
	close(results)

	var success int
	var insufficient int
	for r := range results {
		if r.err == nil {
			success++
			continue
		}
		if r.err == ErrInsufficientStock {
			insufficient++
			continue
		}
		t.Fatalf("unexpected error: %v", r.err)
	}

	if success != 1 || insufficient != 1 {
		t.Fatalf("expected 1 success + 1 insufficient, got success=%d insufficient=%d", success, insufficient)
	}

	var stock int64
	if err := pool.QueryRow(ctx, `SELECT stock FROM inventory WHERE product_id = $1`, productID).Scan(&stock); err != nil {
		t.Fatalf("query inventory: %v", err)
	}
	if stock != 0 {
		t.Fatalf("expected final stock=0, got %d", stock)
	}

	var ordersCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&ordersCount); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if ordersCount != 1 {
		t.Fatalf("expected 1 order row, got %d", ordersCount)
	}
}

func newTestPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse POSTGRES_DSN: %v", err)
	}
	// Needed for executing multi-statement migrations (BEGIN; ...; COMMIT;).
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	return pool
}

func ensureSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	schemaOnce.Do(func() {
		var exists *string
		if err := pool.QueryRow(ctx, `SELECT to_regclass('public.users')::text`).Scan(&exists); err != nil {
			t.Fatalf("check schema: %v", err)
		}
		if exists != nil && *exists != "" {
			return
		}

		migrationSQL, err := os.ReadFile("migrations/0001_init.sql")
		if err != nil {
			t.Fatalf("read migration: %v", err)
		}
		if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
			t.Fatalf("apply migration: %v", err)
		}
	})
}

func resetTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	// Keep it focused on tables used in this test.
	if _, err := pool.Exec(ctx, `
TRUNCATE TABLE
  order_items,
  orders,
  cart_items,
  carts,
  inventory,
  products
RESTART IDENTITY
CASCADE
`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

