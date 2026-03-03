package apihttp

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
)

// requestIDKey is used to store the request ID in context.
type requestIDKey struct{}

// RequestIDMiddleware adds a request ID to the context and response headers.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middleware.GetReqID(r.Context())
		if id == "" {
			// middleware.RequestID will set one later; we just ensure header presence.
			id = ""
		}
		if id != "" {
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs basic request information.
func LoggingMiddleware(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			fields := map[string]any{
				"method":   r.Method,
				"path":     r.URL.Path,
				"status":   ww.Status(),
				"duration": time.Since(start).String(),
			}
			log.Info("http_request", logger.WithContext(r.Context(), fields))
		})
	}
}

// RecoverMiddleware converts panics into 500 responses.
func RecoverMiddleware(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", nil, logger.WithContext(r.Context(), map[string]any{
						"panic": rec,
					}))
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

