package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// Service handles order creation and inventory updates.
type Service struct {
	db        *pgxpool.Pool
	inventory repository.InventoryRepository
	orders    repository.OrderRepository
	carts     repository.CartRepository
	log       logger.Logger
}

// NewService constructs a new Service.
func NewService(
	db *pgxpool.Pool,
	inventory repository.InventoryRepository,
	orders repository.OrderRepository,
	carts repository.CartRepository,
	log logger.Logger,
) *Service {
	return &Service{
		db:        db,
		inventory: inventory,
		orders:    orders,
		carts:     carts,
		log:       log,
	}
}

// CreateOrderRequest defines inputs for creating an order from a cart.
type CreateOrderRequest struct {
	CartID  int64
	UserID  *int64
	Currency string
}

// CreateOrderResult is returned to the caller after order creation.
type CreateOrderResult struct {
	OrderID int64 `json:"order_id"`
}

var (
	ErrInsufficientStock = errors.New("insufficient stock for one or more items")
)

// CreateFromCart creates an order from a cart and updates inventory atomically.
func (s *Service) CreateFromCart(ctx context.Context, req CreateOrderRequest) (*CreateOrderResult, error) {
	cart, err := s.carts.GetCartWithItems(ctx, req.CartID)
	if err != nil {
		return nil, fmt.Errorf("load cart: %w", err)
	}
	if len(cart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var total int64
	for _, item := range cart.Items {
		inv, err := s.inventory.GetForUpdate(ctx, tx, item.ProductID)
		if err != nil {
			return nil, err
		}
		if inv.Stock-inv.ReservedStock < item.Quantity {
			return nil, ErrInsufficientStock
		}
		inv.Stock -= item.Quantity
		inv.ReservedStock += 0
		if err := s.inventory.Update(ctx, tx, inv); err != nil {
			return nil, err
		}
		total += item.PriceCentsSnapshot * item.Quantity
	}

	now := time.Now().UTC()
	status := "pending"
	paymentStatus := "pending"
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	var userID *int64
	if req.UserID != nil {
		userID = req.UserID
	}

	orderModel := &repository.Order{
		UserID:           userID,
		Status:           status,
		TotalAmountCents: total,
		CurrencyCode:     currency,
		PaymentStatus:    paymentStatus,
		PlacedAt:         &now,
	}

	var items []repository.OrderItem
	for _, it := range cart.Items {
		itemTotal := it.PriceCentsSnapshot * it.Quantity
		items = append(items, repository.OrderItem{
			ProductID:       it.ProductID,
			Quantity:        it.Quantity,
			UnitPriceCents:  it.PriceCentsSnapshot,
			TotalPriceCents: itemTotal,
		})
	}

	if err := s.orders.Create(ctx, tx, orderModel, items); err != nil {
		return nil, err
	}

	if err := s.carts.ClearCart(ctx, cart.ID); err != nil {
		return nil, fmt.Errorf("clear cart: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit order tx: %w", err)
	}

	s.log.Info("order created", logger.WithContext(ctx, map[string]any{
		"order_id": orderModel.ID,
		"user_id":  userID,
		"total":    total,
	}))

	return &CreateOrderResult{OrderID: orderModel.ID}, nil
}

