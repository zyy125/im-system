package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/pkg/jwt"
	"gorm.io/gorm"
)

type MockAuthRepo struct {
	users map[uint64]*model.User
}

type MockTokenBlacklistRepo struct {
	tokens map[string]time.Duration
}

type MockRefreshSessionRepo struct {
	sessions map[string]mockRefreshSession
}

type mockRefreshSession struct {
	userID           uint64
	refreshTokenHash string
	ttl              time.Duration
}

func (m *MockAuthRepo) Create(ctx context.Context, user *model.User) error {
	if _, ok := m.users[user.PublicID]; ok {
		return gorm.ErrDuplicatedKey
	}
	m.users[user.PublicID] = user
	return nil
}

func (m *MockAuthRepo) Update(ctx context.Context, user *model.User) error {
	if user == nil {
		return nil
	}
	m.users[user.PublicID] = user
	return nil
}

func (m *MockAuthRepo) GetByID(ctx context.Context, id uint64) (model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return *u, nil
		}
	}
	return model.User{}, gorm.ErrRecordNotFound
}

func (m *MockAuthRepo) GetByPublicID(ctx context.Context, publicID uint64) (model.User, error) {
	for _, u := range m.users {
		if u.PublicID == publicID {
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
	ttl, ok := m.tokens[token]
	if !ok {
		return false, nil
	}
	return ttl > 0, nil
}

func (m *MockTokenBlacklistRepo) Blacklist(ctx context.Context, token string, ttl time.Duration) error {
	m.tokens[token] = ttl
	return nil
}

func (m *MockRefreshSessionRepo) Create(ctx context.Context, sessionID string, userID uint64, refreshTokenHash string, ttl time.Duration) error {
	m.sessions[sessionID] = mockRefreshSession{
		userID:           userID,
		refreshTokenHash: refreshTokenHash,
		ttl:              ttl,
	}
	return nil
}

func (m *MockRefreshSessionRepo) Rotate(ctx context.Context, sessionID, oldRefreshTokenHash, newRefreshTokenHash string, ttl time.Duration) (uint64, bool, error) {
	session, ok := m.sessions[sessionID]
	if !ok || session.refreshTokenHash != oldRefreshTokenHash {
		return 0, false, nil
	}
	session.refreshTokenHash = newRefreshTokenHash
	session.ttl = ttl
	m.sessions[sessionID] = session
	return session.userID, true, nil
}

func (m *MockRefreshSessionRepo) Delete(ctx context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func TestAuthService_Register(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[uint64]*model.User)}
	jwtCfg := &config.JWT{
		Secret:        "test-secret",
		AccessExpiry:  1,
		RefreshExpiry: 24,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]time.Duration)}
	refreshSessionRepo := &MockRefreshSessionRepo{sessions: make(map[string]mockRefreshSession)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo, refreshSessionRepo)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		username := "test-user"
		pwd := "test-password"
		result, err := authService.Register(ctx, username, pwd)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, result.PublicID, uint64(100000000))
		assert.LessOrEqual(t, result.PublicID, uint64(999999999))

		assert.Equal(t, username, authRepo.users[result.PublicID].Username)
	})

	t.Run("DuplicateUsernameAllowed", func(t *testing.T) {
		username := "test-user"
		pwd := "test-password"
		_, err := authService.Register(ctx, username, pwd)
		assert.NoError(t, err)
	})

	t.Run("EmptyUsername", func(t *testing.T) {
		_, err := authService.Register(ctx, "", "password")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthCredentialsRequired, apperr.CodeOf(err))
	})

	t.Run("EmptyPassword", func(t *testing.T) {
		_, err := authService.Register(ctx, "user", "")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthCredentialsRequired, apperr.CodeOf(err))
	})
}

func TestAuthService_Login(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[uint64]*model.User)}
	jwtCfg := &config.JWT{
		Secret:        "test-secret",
		AccessExpiry:  1,
		RefreshExpiry: 24,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]time.Duration)}
	refreshSessionRepo := &MockRefreshSessionRepo{sessions: make(map[string]mockRefreshSession)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo, refreshSessionRepo)

	ctx := context.Background()
	username := "login-user"
	pwd := "login-password"

	// Setup user
	_, err := authService.Register(ctx, username, pwd)
	assert.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		publicID := firstPublicID(authRepo.users)
		tokens, err := authService.Login(ctx, publicID, pwd)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, int64(time.Hour/time.Second), tokens.ExpiresIn)
	})

	t.Run("WrongPassword", func(t *testing.T) {
		publicID := firstPublicID(authRepo.users)
		tokens, err := authService.Login(ctx, publicID, "wrong-password")
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthInvalidCredentials, apperr.CodeOf(err))
		assert.Empty(t, tokens.AccessToken)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		tokens, err := authService.Login(ctx, 999999999, pwd)
		assert.Error(t, err)
		assert.Equal(t, apperr.CodeAuthInvalidCredentials, apperr.CodeOf(err))
		assert.Empty(t, tokens.AccessToken)
	})
}

func TestAuthService_RefreshAndLogout(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[uint64]*model.User)}
	jwtCfg := &config.JWT{
		Secret:        "test-secret",
		AccessExpiry:  1,
		RefreshExpiry: 24,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]time.Duration)}
	refreshSessionRepo := &MockRefreshSessionRepo{sessions: make(map[string]mockRefreshSession)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo, refreshSessionRepo)

	ctx := context.Background()
	username := "test-user"
	pwd := "test-password"

	_, err := authService.Register(ctx, username, pwd)
	assert.NoError(t, err)

	publicID := firstPublicID(authRepo.users)
	tokens, err := authService.Login(ctx, publicID, pwd)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)

	claims, err := jwt.ParseJWT(tokens.AccessToken, jwtCfg.Secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, claims)
	assert.NotEmpty(t, claims.SessionID)

	refreshed, err := authService.Refresh(ctx, tokens.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)
	assert.NotEmpty(t, refreshed.RefreshToken)
	assert.NotEqual(t, tokens.RefreshToken, refreshed.RefreshToken)

	_, err = authService.Refresh(ctx, tokens.RefreshToken)
	assert.Error(t, err)
	assert.Equal(t, apperr.CodeAuthRefreshTokenInvalid, apperr.CodeOf(err))

	err = authService.Logout(ctx, claims.ID, claims.SessionID, claims.ExpiresAt.Time)
	assert.NoError(t, err)

	blacklisted, err := tokenBlacklistRepo.IsBlacklisted(ctx, claims.ID)
	assert.NoError(t, err)
	assert.True(t, blacklisted)
	assert.Greater(t, tokenBlacklistRepo.tokens[claims.ID], time.Duration(0))
	_, ok := refreshSessionRepo.sessions[claims.SessionID]
	assert.False(t, ok)
}

func TestAuthService_LogoutSkipsExpiredToken(t *testing.T) {
	authRepo := &MockAuthRepo{users: make(map[uint64]*model.User)}
	jwtCfg := &config.JWT{
		Secret:        "test-secret",
		AccessExpiry:  1,
		RefreshExpiry: 24,
	}
	tokenBlacklistRepo := &MockTokenBlacklistRepo{tokens: make(map[string]time.Duration)}
	refreshSessionRepo := &MockRefreshSessionRepo{sessions: make(map[string]mockRefreshSession)}
	authService := NewAuthService(authRepo, jwtCfg, tokenBlacklistRepo, refreshSessionRepo)

	err := refreshSessionRepo.Create(context.Background(), "session-expired", 1, "hash", time.Hour)
	assert.NoError(t, err)

	err = authService.Logout(context.Background(), "expired-jti", "session-expired", time.Now().Add(-time.Minute))
	assert.NoError(t, err)
	assert.Empty(t, tokenBlacklistRepo.tokens)
	_, ok := refreshSessionRepo.sessions["session-expired"]
	assert.False(t, ok)
}

func firstPublicID(users map[uint64]*model.User) uint64 {
	for publicID := range users {
		return publicID
	}
	return 0
}
