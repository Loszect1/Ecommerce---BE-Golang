package apihttp

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	authsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/auth"
	cartsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/cart"
	catalogsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/catalog"
	ordersvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/order"
	paymentsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/payment"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/oauth"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// Dependencies bundles all services needed by HTTP handlers.
type Dependencies struct {
	Log      logger.Logger
	Auth     *authsvc.Service
	Catalog  *catalogsvc.Service
	Cart     *cartsvc.Service
	Order    *ordersvc.Service
	Payment  *paymentsvc.StripeService
	DB       *pgxpool.Pool
	Payments repository.PaymentRepository
	Orders   repository.OrderPaymentRepository
	OAuthCfg oauth.ProviderConfig
	JWTKey   []byte
	AdminEmails string
	StripeWebhookSecret string
}

// NewRouter constructs the main HTTP router.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(CORSMiddleware)
	r.Use(LoggingMiddleware(deps.Log))
	r.Use(RecoverMiddleware(deps.Log))
	r.Use(RequestIDMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Auth
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
				handleRegister(w, r, deps)
			})
			r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
				handleLogin(w, r, deps)
			})
			r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
				handleRefresh(w, r, deps)
			})
			r.With(RequireAuthMiddleware(deps.JWTKey)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
				handleMe(w, r, deps)
			})
			// Social login URL endpoints (skeletons).
			r.Get("/oauth/google/url", func(w http.ResponseWriter, r *http.Request) {
				handleOAuthURL(w, r, deps, "google")
			})
			r.Get("/oauth/facebook/url", func(w http.ResponseWriter, r *http.Request) {
				handleOAuthURL(w, r, deps, "facebook")
			})
			r.Get("/oauth/google/callback", func(w http.ResponseWriter, r *http.Request) {
				handleOAuthCallback(w, r, deps, "google")
			})
			r.Get("/oauth/facebook/callback", func(w http.ResponseWriter, r *http.Request) {
				handleOAuthCallback(w, r, deps, "facebook")
			})
		})

		// Catalog
		r.Get("/categories", func(w http.ResponseWriter, r *http.Request) {
			handleListCategories(w, r, deps)
		})
		r.Route("/products", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				handleListProducts(w, r, deps)
			})
			r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				handleGetProduct(w, r, deps)
			})
		})

		// Cart
		r.Route("/cart", func(r chi.Router) {
			r.Post("/items", func(w http.ResponseWriter, r *http.Request) {
				handleAddCartItem(w, r, deps)
			})
			r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				handleGetCart(w, r, deps)
			})
			r.Delete("/{id}/items/{productID}", func(w http.ResponseWriter, r *http.Request) {
				handleRemoveCartItem(w, r, deps)
			})
		})

		// Orders
		r.Route("/orders", func(r chi.Router) {
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				handleCreateOrder(w, r, deps)
			})
		})

		// Payments
		r.Route("/payments", func(r chi.Router) {
			r.Post("/stripe/webhook", func(w http.ResponseWriter, r *http.Request) {
				handleStripeWebhook(w, r, deps)
			})
		})

		// Admin
		r.Route("/admin", func(r chi.Router) {
			r.Use(RequireAuthMiddleware(deps.JWTKey))
			r.Use(RequireAdminMiddleware(deps.Auth, deps.AdminEmails))

			r.Route("/products", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					handleAdminListProducts(w, r, deps)
				})
				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					handleAdminCreateProduct(w, r, deps)
				})
				r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
					handleAdminUpdateProduct(w, r, deps)
				})
				r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
					handleAdminDeleteProduct(w, r, deps)
				})
			})
		})
	})

	return r
}

func parseIDParam(r *http.Request, key string) (int64, error) {
	raw := chi.URLParam(r, key)
	return strconv.ParseInt(raw, 10, 64)
}

