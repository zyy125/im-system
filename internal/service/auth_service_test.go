package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/pkg/jwt"
	"gorm.io/gorm"
)

type MockAuthRepo struct {
	users map[string]*model.User
}

type MockTokenBlacklistRepo struct {
	tokens map[string]bool
}

func (m *MockAuthRepo) Create(ctx context.Context, user *model.User) error {
	if _, ok := m.users[user.Username]; ok {
		return gorm.ErrDuplicatedKey
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockAuthRepo) GetByUsername(ctx context.Context, username string) (model.User, error) {
	user, ok := m.users[username]
	if !ok {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return *user, nil
}

func (m *MockAuthRepo) GetByID(ctx context.Context, id uint64) (model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return *u, nil
		}
	}
	return model.User{}, gorm.ErrRecordNotFound
}

func (m *MockAuthRepo) ListByIDs(ctx context.Context, ids []uint64) ([]model.User, error) {
	if len(ids) == 0 {
		return []model.User{}, nil
	}
	res := make([]model.User, 0, len(ids))
	set := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	for _, u := range m.users {
		if _, ok := set[u.ID]; ok {
			res = append(res, *u)
		}
	}
	return res, nil
}

func (m *MockTokenBlacklistRepo) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	blacklisted, ok := m.tokens[token]
	if !ok {
		return false, nil
	}
	return blacklisted, nil
}

func (m *MockTokenBlacklistRepo) Blacklist(ctx context.Context, token string) error {
	m.tokens[token] = true
	return nil
}

func TestAuthService_Register(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[string]*model.User)}
	jwtCfg := &config.JWT{
		Secret: "test-secret",
		Expiry: 1,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]bool)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		username := "test-user"
		pwd := "test-password"
		err := authService.Register(ctx, username, pwd)
		assert.NoError(t, err)

		user, err := authRepo.GetByUsername(ctx, username)
		assert.NoError(t, err)
		assert.Equal(t, username, user.Username)
	})

	t.Run("Duplicate", func(t *testing.T) {
		username := "test-user"
		pwd := "test-password"
		err := authService.Register(ctx, username, pwd)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeUserAlreadyExist, apperr.CodeOf(err))
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		err := authService.Register(ctx, "", "password")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthCredentialsRequired, apperr.CodeOf(err))
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		err := authService.Register(ctx, "user", "")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthCredentialsRequired, apperr.CodeOf(err))
	})
}

func TestAuthService_Login(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[string]*model.User)}
	jwtCfg := &config.JWT{
		Secret: "test-secret",
		Expiry: 1,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]bool)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo)

	ctx := context.Background()
	username := "login-user"
	pwd := "login-password"

	// Setup user
	err := authService.Register(ctx, username, pwd)
	assert.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		token, err := authService.Login(ctx, username, pwd)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("WrongPassword", func(t *testing.T) {
		token, err := authService.Login(ctx, username, "wrong-password")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthInvalidCredentials, apperr.CodeOf(err))
		assert.Empty(t, token)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		token, err := authService.Login(ctx, "non-existent", pwd)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthInvalidCredentials, apperr.CodeOf(err))
		assert.Empty(t, token)
	})
}

func TestAuthService_Logout(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[string]*model.User)}
	jwtCfg := &config.JWT{
		Secret: "test-secret",
		Expiry: 1,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]bool)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo)

	ctx := context.Background()
	username := "test-user"
	pwd := "test-password"

	err := authService.Register(ctx, username, pwd)
	assert.NoError(t, err)

	token, err := authService.Login(ctx, username, pwd)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jwt.ParseJWT(token, jwtCfg.Secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, claims)

	err = authService.Logout(ctx, claims.ID)
	assert.NoError(t, err)

	blacklisted, err := tokenBlacklistRepo.IsBlacklisted(ctx, claims.ID)
	assert.NoError(t, err)
	assert.True(t, blacklisted)
}
