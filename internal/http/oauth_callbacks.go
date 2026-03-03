package apihttp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

type oauthProfile struct {
	ProviderUserID string
	Email          string
	FullName       string
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, deps Dependencies, provider string) {
	var cfg *oauth2.Config
	switch provider {
	case "google":
		cfg = deps.OAuthCfg.Google
	case "facebook":
		cfg = deps.OAuthCfg.Facebook
	default:
		writeError(w, http.StatusBadRequest, "unknown provider")
		return
	}
	if cfg == nil {
		writeError(w, http.StatusNotImplemented, "oauth is not configured")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing code")
		return
	}
	if state == "" {
		writeError(w, http.StatusBadRequest, "missing state")
		return
	}
	if !validateOAuthState(r, provider, state) {
		writeError(w, http.StatusBadRequest, "invalid state")
		return
	}
	clearOAuthStateCookie(w, provider)

	token, err := cfg.Exchange(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to exchange code")
		return
	}

	profile, err := fetchOAuthProfile(r.Context(), provider, cfg, token)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to fetch profile")
		return
	}

	user, tokens, err := deps.Auth.LoginWithOAuthProvider(r.Context(), provider, profile.ProviderUserID, profile.Email, profile.FullName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":   user,
		"tokens": tokens,
	})
}

func handleOAuthURL(w http.ResponseWriter, r *http.Request, deps Dependencies, provider string) {
	switch provider {
	case "google":
		if deps.OAuthCfg.Google == nil {
			writeError(w, http.StatusBadRequest, "google oauth is not configured")
			return
		}
		state, err := newOAuthState()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate state")
			return
		}
		setOAuthStateCookie(w, r, provider, state)
		url := deps.OAuthCfg.Google.AuthCodeURL(state)
		writeJSON(w, http.StatusOK, map[string]string{"url": url})
	case "facebook":
		if deps.OAuthCfg.Facebook == nil {
			writeError(w, http.StatusBadRequest, "facebook oauth is not configured")
			return
		}
		state, err := newOAuthState()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate state")
			return
		}
		setOAuthStateCookie(w, r, provider, state)
		url := deps.OAuthCfg.Facebook.AuthCodeURL(state)
		writeJSON(w, http.StatusOK, map[string]string{"url": url})
	default:
		writeError(w, http.StatusBadRequest, "unknown provider")
	}
}

func fetchOAuthProfile(ctx context.Context, provider string, cfg *oauth2.Config, token *oauth2.Token) (*oauthProfile, error) {
	switch provider {
	case "google":
		return fetchGoogleProfile(ctx, cfg, token)
	case "facebook":
		return fetchFacebookProfile(ctx, token)
	default:
		return nil, fmt.Errorf("unknown provider")
	}
}

func fetchGoogleProfile(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*oauthProfile, error) {
	client := cfg.Client(ctx, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request userinfo: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}

	var data struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	return &oauthProfile{
		ProviderUserID: strings.TrimSpace(data.Sub),
		Email:          strings.TrimSpace(data.Email),
		FullName:       strings.TrimSpace(data.Name),
	}, nil
}

func fetchFacebookProfile(ctx context.Context, token *oauth2.Token) (*oauthProfile, error) {
	u := "https://graph.facebook.com/me?fields=id,name,email&access_token=" + token.AccessToken
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request profile: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("profile status %d", resp.StatusCode)
	}

	var data struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &oauthProfile{
		ProviderUserID: strings.TrimSpace(data.ID),
		Email:          strings.TrimSpace(data.Email),
		FullName:       strings.TrimSpace(data.Name),
	}, nil
}

func newOAuthState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func oauthStateCookieName(provider string) string {
	return "oauth_state_" + provider
}

func setOAuthStateCookie(w http.ResponseWriter, r *http.Request, provider, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName(provider),
		Value:    state,
		Path:     "/api/v1/auth/oauth/" + provider + "/callback",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func validateOAuthState(r *http.Request, provider, state string) bool {
	c, err := r.Cookie(oauthStateCookieName(provider))
	if err != nil {
		return false
	}
	return subtleConstantTimeEquals(c.Value, state)
}

func clearOAuthStateCookie(w http.ResponseWriter, provider string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName(provider),
		Value:    "",
		Path:     "/api/v1/auth/oauth/" + provider + "/callback",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func subtleConstantTimeEquals(a, b string) bool {
	// avoid importing crypto/subtle for a tiny check; keep it simple and safe enough for state cookies.
	if len(a) != len(b) {
		return false
	}
	var out byte
	for i := 0; i < len(a); i++ {
		out |= a[i] ^ b[i]
	}
	return out == 0
}

