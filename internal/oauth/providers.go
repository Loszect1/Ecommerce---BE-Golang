package oauth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ProviderConfig holds OAuth2 configuration for a provider.
type ProviderConfig struct {
	Google   *oauth2.Config
	Facebook *oauth2.Config
}

// NewProviderConfig builds OAuth2 configs for Google and Facebook.
func NewProviderConfig(
	googleClientID, googleClientSecret, googleRedirectURL string,
	facebookClientID, facebookClientSecret, facebookRedirectURL string,
) ProviderConfig {
	var googleCfg *oauth2.Config
	if googleClientID != "" && googleClientSecret != "" && googleRedirectURL != "" {
		googleCfg = &oauth2.Config{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	// Facebook does not have a helper in x/oauth2; configure endpoint manually.
	var facebookCfg *oauth2.Config
	if facebookClientID != "" && facebookClientSecret != "" && facebookRedirectURL != "" {
		facebookCfg = &oauth2.Config{
			ClientID:     facebookClientID,
			ClientSecret: facebookClientSecret,
			RedirectURL:  facebookRedirectURL,
			Scopes:       []string{"email", "public_profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.facebook.com/v12.0/dialog/oauth",
				TokenURL: "https://graph.facebook.com/v12.0/oauth/access_token",
			},
		}
	}

	return ProviderConfig{
		Google:   googleCfg,
		Facebook: facebookCfg,
	}
}

