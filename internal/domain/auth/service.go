package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

// Service implements authentication and token issuance.
type Service struct {
	users      repository.UserRepository
	providers  repository.UserProviderRepository
	refresh    repository.RefreshTokenStore
	log        logger.Logger
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService constructs an auth Service.
func NewService(
	users repository.UserRepository,
	providers repository.UserProviderRepository,
	refresh repository.RefreshTokenStore,
	log logger.Logger,
	jwtSecret string,
) *Service {
	if jwtSecret == "" {
		// In production this must be set; we still construct the service but most operations will fail.
		log.Error("JWT_SECRET is empty", nil, nil)
	}

	return &Service{
		users:      users,
		providers:  providers,
		refresh:    refresh,
		log:        log,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

// UserDTO is a safe-to-return view of a user.
type UserDTO struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// TokenPair contains access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Register creates a new user and returns tokens.
func (s *Service) Register(ctx context.Context, email, password, fullName string) (*UserDTO, *TokenPair, error) {
	if email == "" || password == "" {
		return nil, nil, fmt.Errorf("email and password are required")
	}

	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, nil, fmt.Errorf("email already in use")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	u := &repository.User{
		Email:        email,
		PasswordHash: string(hash),
		FullName:     fullName,
		IsActive:     true,
	}

	if err := s.users.Create(ctx, u); err != nil {
		return nil, nil, err
	}

	dto := &UserDTO{
		ID:       u.ID,
		Email:    u.Email,
		FullName: u.FullName,
	}

	tokens, err := s.issueTokens(ctx, u.ID, u.Email)
	if err != nil {
		return nil, nil, err
	}

	return dto, tokens, nil
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, email, password string) (*UserDTO, *TokenPair, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	dto := &UserDTO{
		ID:       u.ID,
		Email:    u.Email,
		FullName: u.FullName,
	}

	tokens, err := s.issueTokens(ctx, u.ID, u.Email)
	if err != nil {
		return nil, nil, err
	}

	return dto, tokens, nil
}

// Refresh exchanges a refresh token for a new token pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	rt, err := s.refresh.GetValid(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	user, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return s.issueTokens(ctx, user.ID, user.Email)
}

func (s *Service) issueTokens(ctx context.Context, userID int64, email string) (*TokenPair, error) {
	now := time.Now().UTC()

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   fmt.Sprint(userID),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
	}).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   fmt.Sprint(userID),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
	}).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	if err := s.refresh.Insert(ctx, &repository.RefreshToken{
		UserID:    userID,
		Token:     refreshToken,
		ExpiresAt: now.Add(s.refreshTTL),
	}); err != nil {
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GetUserProfile returns a safe-to-return user view for the given user ID.
func (s *Service) GetUserProfile(ctx context.Context, userID int64) (*UserDTO, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &UserDTO{
		ID:       u.ID,
		Email:    u.Email,
		FullName: u.FullName,
	}, nil
}

// ParseAccessToken validates a JWT access token and returns the subject user ID.
func (s *Service) ParseAccessToken(tokenStr string) (int64, error) {
	if tokenStr == "" {
		return 0, fmt.Errorf("token is required")
	}
	if len(s.jwtSecret) == 0 {
		return 0, fmt.Errorf("jwt secret is not configured")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	claims := &jwt.RegisteredClaims{}
	token, err := parser.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	if claims.Subject == "" {
		return 0, fmt.Errorf("invalid token subject")
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid token subject")
	}
	return id, nil
}

