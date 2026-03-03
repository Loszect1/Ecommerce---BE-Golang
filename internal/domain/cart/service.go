package cart

import (
	"context"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// Service exposes cart operations.
type Service struct {
	carts    repository.CartRepository
	products repository.ProductRepository
}

// NewService constructs a cart Service.
func NewService(
	carts repository.CartRepository,
	products repository.ProductRepository,
) *Service {
	return &Service{
		carts:    carts,
		products: products,
	}
}

// AddItemRequest defines payload to add or update a cart item.
type AddItemRequest struct {
	UserID    *int64
	SessionID *string
	ProductID int64
	Quantity  int64
}

// CartDTO represents cart with items for responses.
type CartDTO struct {
	ID     int64             `json:"id"`
	Items  []CartItemDTO     `json:"items"`
	UserID *int64            `json:"user_id,omitempty"`
}

// CartItemDTO is a view of CartItem.
type CartItemDTO struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
	PriceCentsSnapshot int64 `json:"price_cents_snapshot"`
}

// AddOrUpdateItem adds or updates an item in the cart.
func (s *Service) AddOrUpdateItem(ctx context.Context, req AddItemRequest) (*CartDTO, error) {
	cart, err := s.carts.GetOrCreateActiveCart(ctx, req.UserID, req.SessionID)
	if err != nil {
		return nil, err
	}

	product, err := s.products.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}

	if err := s.carts.UpsertItem(ctx, cart.ID, req.ProductID, req.Quantity, product.PriceCents); err != nil {
		return nil, err
	}

	fullCart, err := s.carts.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}

	return toDTO(fullCart), nil
}

// GetCart returns a cart with items by ID.
func (s *Service) GetCart(ctx context.Context, cartID int64) (*CartDTO, error) {
	cart, err := s.carts.GetCartWithItems(ctx, cartID)
	if err != nil {
		return nil, err
	}
	return toDTO(cart), nil
}

// RemoveItem removes a product from the cart.
func (s *Service) RemoveItem(ctx context.Context, cartID, productID int64) (*CartDTO, error) {
	if err := s.carts.RemoveItem(ctx, cartID, productID); err != nil {
		return nil, err
	}
	cart, err := s.carts.GetCartWithItems(ctx, cartID)
	if err != nil {
		return nil, err
	}
	return toDTO(cart), nil
}

func toDTO(c *repository.Cart) *CartDTO {
	dto := &CartDTO{
		ID:     c.ID,
		UserID: c.UserID,
	}
	for _, it := range c.Items {
		dto.Items = append(dto.Items, CartItemDTO{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			PriceCentsSnapshot: it.PriceCentsSnapshot,
		})
	}
	return dto
}

