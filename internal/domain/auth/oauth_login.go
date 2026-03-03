package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// LoginWithOAuthProvider links (or creates) a user for the given provider identity and issues tokens.
func (s *Service) LoginWithOAuthProvider(ctx context.Context, provider, providerUserID, email, fullName string) (*UserDTO, *TokenPair, error) {
	if s.providers == nil {
		return nil, nil, fmt.Errorf("user provider store is not configured")
	}
	provider = strings.TrimSpace(provider)
	providerUserID = strings.TrimSpace(providerUserID)
	email = strings.TrimSpace(email)
	fullName = strings.TrimSpace(fullName)

	if provider == "" || providerUserID == "" {
		return nil, nil, fmt.Errorf("provider identity is required")
	}
	if email == "" {
		return nil, nil, fmt.Errorf("email is required")
	}

	// 1) If provider link already exists, just load user and issue tokens.
	link, err := s.providers.GetByProviderUserID(ctx, provider, providerUserID)
	if err == nil && link != nil {
		u, err := s.users.GetByID(ctx, link.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("get user: %w", err)
		}
		dto := &UserDTO{ID: u.ID, Email: u.Email, FullName: u.FullName}
		tokens, err := s.issueTokens(ctx, u.ID, u.Email)
		if err != nil {
			return nil, nil, err
		}
		return dto, tokens, nil
	}
	if err != nil && !errors.Is(err, repository.ErrUserProviderNotFound) {
		return nil, nil, fmt.Errorf("get provider link: %w", err)
	}

	// 2) Find or create user by email.
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, repository.ErrUserNotFound) {
			return nil, nil, fmt.Errorf("get user by email: %w", err)
		}

		hash, err := oauthPasswordHash()
		if err != nil {
			return nil, nil, err
		}
		newUser := &repository.User{
			Email:        email,
			PasswordHash: hash,
			FullName:     fullName,
			IsActive:     true,
		}
		if err := s.users.Create(ctx, newUser); err != nil {
			return nil, nil, fmt.Errorf("create user: %w", err)
		}
		u = newUser
	}

	// 3) Link provider -> user (idempotent on unique conflict).
	link = &repository.UserProvider{
		UserID:         u.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	if err := s.providers.Create(ctx, link); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// Unique violation - likely created concurrently. Re-fetch and continue.
			if _, err2 := s.providers.GetByProviderUserID(ctx, provider, providerUserID); err2 == nil {
				// ok
			} else {
				return nil, nil, fmt.Errorf("get provider link after conflict: %w", err2)
			}
		} else {
			return nil, nil, fmt.Errorf("create provider link: %w", err)
		}
	}

	dto := &UserDTO{ID: u.ID, Email: u.Email, FullName: u.FullName}
	tokens, err := s.issueTokens(ctx, u.ID, u.Email)
	if err != nil {
		return nil, nil, err
	}
	return dto, tokens, nil
}

func oauthPasswordHash() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate oauth password: %w", err)
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash oauth password: %w", err)
	}
	return string(hash), nil
}

