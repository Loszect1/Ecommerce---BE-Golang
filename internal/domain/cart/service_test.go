package cart

import (
	"context"
	"testing"
	"time"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

type fakeCartRepo struct {
	nextID int64
	carts  map[int64]*repository.Cart
}

func newFakeCartRepo() *fakeCartRepo {
	return &fakeCartRepo{
		nextID: 1,
		carts:  make(map[int64]*repository.Cart),
	}
}

func (r *fakeCartRepo) GetOrCreateActiveCart(ctx context.Context, userID *int64, sessionID *string) (*repository.Cart, error) {
	for _, c := range r.carts {
		return c, nil
	}
	id := r.nextID
	now := time.Now()
	c := &repository.Cart{
		ID:        id,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.carts[id] = c
	r.nextID++
	return c, nil
}

func (r *fakeCartRepo) UpsertItem(ctx context.Context, cartID, productID int64, quantity int64, priceSnapshot int64) error {
	c := r.carts[cartID]
	for i, it := range c.Items {
		if it.ProductID == productID {
			c.Items[i].Quantity = quantity
			return nil
		}
	}
	c.Items = append(c.Items, repository.CartItem{
		CartID:             cartID,
		ProductID:          productID,
		Quantity:           quantity,
		PriceCentsSnapshot: priceSnapshot,
	})
	return nil
}

func (r *fakeCartRepo) RemoveItem(ctx context.Context, cartID, productID int64) error {
	c := r.carts[cartID]
	out := c.Items[:0]
	for _, it := range c.Items {
		if it.ProductID != productID {
			out = append(out, it)
		}
	}
	c.Items = out
	return nil
}

func (r *fakeCartRepo) GetCartWithItems(ctx context.Context, cartID int64) (*repository.Cart, error) {
	return r.carts[cartID], nil
}

func (r *fakeCartRepo) ClearCart(ctx context.Context, cartID int64) error {
	c := r.carts[cartID]
	c.Items = nil
	return nil
}

type fakeProductRepo struct{}

func (r *fakeProductRepo) List(ctx context.Context, limit, offset int) ([]repository.Product, error) {
	return nil, nil
}

func (r *fakeProductRepo) GetByID(ctx context.Context, id int64) (*repository.Product, error) {
	return &repository.Product{
		ID:         id,
		PriceCents: 1000,
	}, nil
}

func TestAddOrUpdateItem(t *testing.T) {
	cartRepo := newFakeCartRepo()
	productRepo := &fakeProductRepo{}
	svc := NewService(cartRepo, productRepo)

	ctx := context.Background()

	cart, err := svc.AddOrUpdateItem(ctx, AddItemRequest{
		ProductID: 1,
		Quantity:  2,
	})
	if err != nil {
		t.Fatalf("AddOrUpdateItem returned error: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cart.Items))
	}
	if cart.Items[0].Quantity != 2 {
		t.Fatalf("expected quantity 2, got %d", cart.Items[0].Quantity)
	}

	cart, err = svc.AddOrUpdateItem(ctx, AddItemRequest{
		ProductID: 1,
		Quantity:  3,
	})
	if err != nil {
		t.Fatalf("AddOrUpdateItem update returned error: %v", err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 3 {
		t.Fatalf("expected updated quantity 3, got %+v", cart.Items[0])
	}
}

