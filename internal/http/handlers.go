package apihttp

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	cartsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/cart"
	ordersvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/order"
)

// Auth handlers

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func handleRegister(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, tokens, err := deps.Auth.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleLogin(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, tokens, err := deps.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func handleRefresh(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tokens, err := deps.Auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func handleMe(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	profile, err := deps.Auth.GetUserProfile(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// Catalog handlers

func handleListProducts(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	products, err := deps.Catalog.ListProducts(r.Context(), 20, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func handleGetProduct(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	product, err := deps.Catalog.GetProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func handleListCategories(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	categories, err := deps.Catalog.ListCategories(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

// Cart handlers

type addCartItemRequest struct {
	UserID    *int64  `json:"user_id,omitempty"`
	SessionID *string `json:"session_id,omitempty"`
	ProductID int64   `json:"product_id"`
	Quantity  int64   `json:"quantity"`
}

func handleAddCartItem(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	var req addCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cart, err := deps.Cart.AddOrUpdateItem(r.Context(), cartsvc.AddItemRequest{
		UserID:    req.UserID,
		SessionID: req.SessionID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func handleGetCart(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	cart, err := deps.Cart.GetCart(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "cart not found")
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func handleRemoveCartItem(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	cartID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cart id")
		return
	}
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}
	cart, err := deps.Cart.RemoveItem(r.Context(), cartID, productID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// Order handlers

type createOrderRequest struct {
	CartID   int64   `json:"cart_id"`
	UserID   *int64  `json:"user_id,omitempty"`
	Currency *string `json:"currency,omitempty"`
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request, deps Dependencies) {
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	currency := ""
	if req.Currency != nil {
		currency = *req.Currency
	}

	result, err := deps.Order.CreateFromCart(r.Context(), ordersvc.CreateOrderRequest{
		CartID:   req.CartID,
		UserID:   req.UserID,
		Currency: currency,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

