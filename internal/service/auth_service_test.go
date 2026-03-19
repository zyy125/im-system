package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/config"
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
		assert.Equal(t, gorm.ErrDuplicatedKey, err)
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		err := authService.Register(ctx, "", "password")
		assert.Error(t, err)
		assert.Equal(t, "username or password is empty", err.Error())
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		err := authService.Register(ctx, "user", "")
		assert.Error(t, err)
		assert.Equal(t, "username or password is empty", err.Error())
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
		assert.Empty(t, token)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		token, err := authService.Login(ctx, "non-existent", pwd)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
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
