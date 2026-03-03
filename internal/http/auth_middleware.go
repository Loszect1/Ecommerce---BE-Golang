package apihttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// authUserKey is used to store the authenticated user ID in context.
type authUserKey struct{}

// UserIDFromContext extracts a user ID from context if present.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	v := ctx.Value(authUserKey{})
	if v == nil {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

func parseBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", nil
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", jwt.ErrTokenMalformed
	}
	return parts[1], nil
}

func parseUserIDFromJWT(tokenStr string, secret []byte) (int64, error) {
	if tokenStr == "" {
		return 0, jwt.ErrTokenMalformed
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	claims := &jwt.RegisteredClaims{}
	token, err := parser.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return 0, jwt.ErrTokenInvalidClaims
	}
	if claims.Subject == "" {
		return 0, jwt.ErrTokenInvalidClaims
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return 0, jwt.ErrTokenInvalidClaims
	}
	return id, nil
}

// AuthMiddleware validates JWT access tokens and stores user ID in context.
func AuthMiddleware(secret []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := parseBearerToken(r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			if tokenStr == "" {
				next.ServeHTTP(w, r)
				return
			}

			userID, err := parseUserIDFromJWT(tokenStr, secret)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuthMiddleware enforces presence of a valid JWT access token.
func RequireAuthMiddleware(secret []byte) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := parseBearerToken(r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			if tokenStr == "" {
				writeError(w, http.StatusUnauthorized, "missing access token")
				return
			}
			userID, err := parseUserIDFromJWT(tokenStr, secret)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), authUserKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

