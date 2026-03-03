package auth

import (
	"context"
	"testing"
	"time"

	"github.com/Loszect1/Ecommerce---BE-Golang/internal/logger"
	"github.com/Loszect1/Ecommerce---BE-Golang/internal/repository"
)

type fakeUserRepo struct {
	byEmail map[string]*repository.User
	byID    map[int64]*repository.User
	nextID  int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail: make(map[string]*repository.User),
		byID:    make(map[int64]*repository.User),
		nextID:  1,
	}
}

func (r *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*repository.User, error) {
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) GetByID(ctx context.Context, id int64) (*repository.User, error) {
	if u, ok := r.byID[id]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) Create(ctx context.Context, user *repository.User) error {
	user.ID = r.nextID
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	r.byID[user.ID] = user
	r.byEmail[user.Email] = user
	r.nextID++
	return nil
}

type fakeRefreshStore struct {
	tokens map[string]*repository.RefreshToken
}

func newFakeRefreshStore() *fakeRefreshStore {
	return &fakeRefreshStore{tokens: make(map[string]*repository.RefreshToken)}
}

func (s *fakeRefreshStore) Insert(ctx context.Context, t *repository.RefreshToken) error {
	s.tokens[t.Token] = t
	return nil
}

func (s *fakeRefreshStore) Revoke(ctx context.Context, token string) error {
	if rt, ok := s.tokens[token]; ok {
		now := time.Now()
		rt.RevokedAt = &now
	}
	return nil
}

func (s *fakeRefreshStore) GetValid(ctx context.Context, token string) (*repository.RefreshToken, error) {
	if rt, ok := s.tokens[token]; ok {
		return rt, nil
	}
	return nil, repository.ErrUserNotFound
}

func TestRegisterAndLogin(t *testing.T) {
	userRepo := newFakeUserRepo()
	refreshStore := newFakeRefreshStore()
	log := logger.New()
	svc := NewService(userRepo, nil, refreshStore, log, "test-secret")

	ctx := context.Background()

	user, tokens, err := svc.Register(ctx, "test@example.com", "StrongPassword123", "Test User")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected email preserved, got %s", user.Email)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens")
	}

	user2, tokens2, err := svc.Login(ctx, "test@example.com", "StrongPassword123")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if user2.ID != user.ID {
		t.Fatalf("expected same user id on login")
	}
	if tokens2.AccessToken == "" {
		t.Fatalf("expected new access token on login")
	}
}

