package catalog

import (
	"context"
	"fmt"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// AdminProductDTO is an admin-facing view of a product.
type AdminProductDTO struct {
	ID           int64   `json:"id"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	PriceCents   int64   `json:"price_cents"`
	CurrencyCode string  `json:"currency_code"`
	MainImageURL *string `json:"main_image_url,omitempty"`
	IsActive     bool    `json:"is_active"`
}

// CreateProductRequest defines inputs for creating a product.
type CreateProductRequest struct {
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	PriceCents   int64   `json:"price_cents"`
	CurrencyCode string  `json:"currency_code"`
	MainImageURL *string `json:"main_image_url,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
	InitialStock *int64  `json:"initial_stock,omitempty"`
}

// UpdateProductRequest defines inputs for updating a product.
type UpdateProductRequest struct {
	ID           int64   `json:"-"`
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	PriceCents   int64   `json:"price_cents"`
	CurrencyCode string  `json:"currency_code"`
	MainImageURL *string `json:"main_image_url,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

func toAdminProductDTO(p *repository.Product) *AdminProductDTO {
	if p == nil {
		return nil
	}
	return &AdminProductDTO{
		ID:           p.ID,
		Slug:         p.Slug,
		Name:         p.Name,
		Description:  p.Description,
		PriceCents:   p.PriceCents,
		CurrencyCode: p.CurrencyCode,
		MainImageURL: p.MainImageURL,
		IsActive:     p.IsActive,
	}
}

// AdminListProducts returns all products including inactive ones.
func (s *Service) AdminListProducts(ctx context.Context, limit, offset int) ([]AdminProductDTO, error) {
	products, err := s.products.ListAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]AdminProductDTO, 0, len(products))
	for _, p := range products {
		p := p
		out = append(out, *toAdminProductDTO(&p))
	}
	return out, nil
}

// AdminCreateProduct creates a product and its inventory row.
func (s *Service) AdminCreateProduct(ctx context.Context, req CreateProductRequest) (*AdminProductDTO, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	initialStock := int64(0)
	if req.InitialStock != nil {
		initialStock = *req.InitialStock
	}

	p := &repository.Product{
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		PriceCents:   req.PriceCents,
		CurrencyCode: req.CurrencyCode,
		MainImageURL: req.MainImageURL,
		IsActive:     isActive,
	}
	if err := s.products.Create(ctx, p, initialStock); err != nil {
		return nil, err
	}
	return toAdminProductDTO(p), nil
}

// AdminUpdateProduct updates mutable product fields.
func (s *Service) AdminUpdateProduct(ctx context.Context, req UpdateProductRequest) (*AdminProductDTO, error) {
	if req.ID <= 0 {
		return nil, fmt.Errorf("id is required")
	}
	existing, err := s.products.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	p := &repository.Product{
		ID:           req.ID,
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		PriceCents:   req.PriceCents,
		CurrencyCode: req.CurrencyCode,
		MainImageURL: req.MainImageURL,
		IsActive:     isActive,
	}
	if err := s.products.Update(ctx, p); err != nil {
		return nil, err
	}
	return toAdminProductDTO(p), nil
}

// AdminDeleteProduct deactivates a product.
func (s *Service) AdminDeleteProduct(ctx context.Context, id int64) error {
	return s.products.Deactivate(ctx, id)
}

