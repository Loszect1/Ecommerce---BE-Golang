package apihttp

import (
	"net/http"
	"strings"

	authsvc "github.com/Loszect1/Ecommerce---BE-Golang/internal/domain/auth"
)

func parseAdminEmails(csv string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, part := range strings.Split(csv, ",") {
		email := strings.TrimSpace(strings.ToLower(part))
		if email == "" {
			continue
		}
		out[email] = struct{}{}
	}
	return out
}

// RequireAdminMiddleware enforces admin authorization.
//
// Current implementation: allow if the authenticated user's email is included in ADMIN_EMAILS.
func RequireAdminMiddleware(auth *authsvc.Service, adminEmailsCSV string) func(next http.Handler) http.Handler {
	allowed := parseAdminEmails(adminEmailsCSV)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok || userID <= 0 {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			profile, err := auth.GetUserProfile(r.Context(), userID)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if len(allowed) == 0 {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			if _, ok := allowed[strings.ToLower(profile.Email)]; !ok {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

