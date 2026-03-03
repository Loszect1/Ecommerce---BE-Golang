package catalog

import (
	"context"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// Service provides read-only catalog operations.
type Service struct {
	products   repository.ProductRepository
	categories repository.CategoryRepository
}

// NewService constructs a catalog Service.
func NewService(products repository.ProductRepository, categories repository.CategoryRepository) *Service {
	return &Service{products: products, categories: categories}
}

// ProductDTO is a subset of product fields safe for API responses.
type ProductDTO struct {
	ID           int64   `json:"id"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	PriceCents   int64   `json:"price_cents"`
	CurrencyCode string  `json:"currency_code"`
	MainImageURL *string `json:"main_image_url,omitempty"`
}

// CategoryDTO is a subset of category fields safe for API responses.
type CategoryDTO struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

func (s *Service) ListProducts(ctx context.Context, limit, offset int) ([]ProductDTO, error) {
	products, err := s.products.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	res := make([]ProductDTO, 0, len(products))
	for _, p := range products {
		p := p
		res = append(res, ProductDTO{
			ID:           p.ID,
			Slug:         p.Slug,
			Name:         p.Name,
			Description:  p.Description,
			PriceCents:   p.PriceCents,
			CurrencyCode: p.CurrencyCode,
			MainImageURL: p.MainImageURL,
		})
	}
	return res, nil
}

func (s *Service) GetProduct(ctx context.Context, id int64) (*ProductDTO, error) {
	p, err := s.products.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ProductDTO{
		ID:           p.ID,
		Slug:         p.Slug,
		Name:         p.Name,
		Description:  p.Description,
		PriceCents:   p.PriceCents,
		CurrencyCode: p.CurrencyCode,
		MainImageURL: p.MainImageURL,
	}, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]CategoryDTO, error) {
	cats, err := s.categories.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryDTO, 0, len(cats))
	for _, c := range cats {
		c := c
		out = append(out, CategoryDTO{
			ID:       c.ID,
			Name:     c.Name,
			Slug:     c.Slug,
			ParentID: c.ParentID,
		})
	}
	return out, nil
}

