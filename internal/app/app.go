package app

import (
	"context"
	"net/http"

	authsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/auth"
	cartsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/cart"
	catalogsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/catalog"
	ordersvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/order"
	paymentsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/payment"
	apihttp "github.com/Loszect1/Ecommerce---BE-Golang/internal/http"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/oauth"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/config"
)

// Config is an alias to the shared configuration struct.
type Config = config.Config

// New constructs the full HTTP handler stack.
func New(cfg Config) http.Handler {
	log := logger.New()

	ctx := context.Background()

	pool, err := repository.NewPostgresPool(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Error("failed to connect to postgres", err, nil)
		// In a serverless context we prefer failing fast.
		panic(err)
	}

	// Repositories
	userRepo := repository.NewUserRepository(pool)
	userProviderRepo := repository.NewUserProviderRepository(pool)
	refreshStore := repository.NewRefreshTokenStore(pool)
	productRepo := repository.NewProductRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	orderRepo := repository.NewOrderRepository()
	paymentRepo := repository.NewPaymentRepository(pool)
	orderPaymentRepo := repository.NewOrderPaymentRepository(pool)

	// Domain services
	authService := authsvc.NewService(userRepo, userProviderRepo, refreshStore, log, cfg.JWTSecret)
	catalogService := catalogsvc.NewService(productRepo, categoryRepo)
	cartService := cartsvc.NewService(cartRepo, productRepo)
	orderService := ordersvc.NewService(pool, inventoryRepo, orderRepo, cartRepo, log)
	stripeService := paymentsvc.NewStripeService(
		cfg.StripeSecretKey,
		cfg.StripeCurrency,
		cfg.StripeSuccessURLBase,
		cfg.StripeCancelURLBase,
		log,
	)

	oauthCfg := oauth.NewProviderConfig(
		cfg.OAuthGoogleClientID,
		cfg.OAuthGoogleClientSecret,
		cfg.OAuthGoogleRedirectURL,
		cfg.OAuthFacebookClientID,
		cfg.OAuthFacebookClientSecret,
		cfg.OAuthFacebookRedirectURL,
	)

	router := apihttp.NewRouter(apihttp.Dependencies{
		Log:      log,
		Auth:     authService,
		Catalog:  catalogService,
		Cart:     cartService,
		Order:    orderService,
		Payment:  stripeService,
		DB:       pool,
		Payments: paymentRepo,
		Orders:   orderPaymentRepo,
		OAuthCfg: oauthCfg,
		JWTKey:   []byte(cfg.JWTSecret),
		AdminEmails: cfg.AdminEmails,
		StripeWebhookSecret: cfg.StripeWebhookSecret,
	})

	return router
}

