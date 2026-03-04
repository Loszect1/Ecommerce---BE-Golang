package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	Env string

	HTTPPort string

	PostgresDSN string

	RedisAddr     string
	RedisPassword string

	JWTSecret string

	AdminEmails string

	StripeSecretKey      string
	StripeWebhookSecret  string
	StripeCurrency       string
	StripeSuccessURLBase string
	StripeCancelURLBase  string

	// OAuth
	OAuthGoogleClientID     string
	OAuthGoogleClientSecret string
	OAuthGoogleRedirectURL  string

	OAuthFacebookClientID     string
	OAuthFacebookClientSecret string
	OAuthFacebookRedirectURL  string

	RequestTimeout time.Duration
}

// FromEnv builds Config from environment variables, providing sane defaults where possible.
func FromEnv() Config {
	return Config{
		Env:                    getenv("APP_ENV", "development"),
		HTTPPort:               getenv("HTTP_PORT", "8080"),
		PostgresDSN:            os.Getenv("POSTGRES_DSN"),
		RedisAddr:              getenv("REDIS_ADDR", ""),
		RedisPassword:          os.Getenv("REDIS_PASSWORD"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		AdminEmails:            os.Getenv("ADMIN_EMAILS"),
		StripeSecretKey:        os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:    os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeCurrency:         getenv("STRIPE_CURRENCY", "USD"),
		StripeSuccessURLBase:   os.Getenv("STRIPE_SUCCESS_URL_BASE"),
		StripeCancelURLBase:    os.Getenv("STRIPE_CANCEL_URL_BASE"),
		OAuthGoogleClientID:    os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
		OAuthGoogleClientSecret: os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
		OAuthGoogleRedirectURL: os.Getenv("OAUTH_GOOGLE_REDIRECT_URL"),
		OAuthFacebookClientID: os.Getenv("OAUTH_FACEBOOK_CLIENT_ID"),
		OAuthFacebookClientSecret: os.Getenv("OAUTH_FACEBOOK_CLIENT_SECRET"),
		OAuthFacebookRedirectURL:  os.Getenv("OAUTH_FACEBOOK_REDIRECT_URL"),
		RequestTimeout:            getenvDuration("REQUEST_TIMEOUT_SECONDS", 10*time.Second),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		secs, err := strconv.Atoi(v)
		if err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return fallback
}
