package apihttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	catalogsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/catalog"
	"github.com/go-chi/chi/v5"
)

func handleAdminListProducts(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	products, err := deps.Catalog.AdminListProducts(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func handleAdminCreateProduct(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	var req catalogsvc.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "slug and name are required")
		return
	}
	if req.PriceCents < 0 {
		writeError(w, http.StatusBadRequest, "price_cents must be >= 0")
		return
	}

	created, err := deps.Catalog.AdminCreateProduct(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func handleAdminUpdateProduct(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req catalogsvc.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ID = id
	if req.Slug == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "slug and name are required")
		return
	}
	if req.PriceCents < 0 {
		writeError(w, http.StatusBadRequest, "price_cents must be >= 0")
		return
	}

	updated, err := deps.Catalog.AdminUpdateProduct(r.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "product not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func handleAdminDeleteProduct(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := deps.Catalog.AdminDeleteProduct(r.Context(), id); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "product not found" {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

