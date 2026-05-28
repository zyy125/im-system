package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/jwt"
	"github.com/zyy125/im-system/pkg/utils"
	"gorm.io/gorm"
)

type authService struct {
	userRepo           repository.UserRepo
	jwtConfig          *config.JWT
	tokenBlacklistRepo repository.TokenBlacklistRepo
	refreshSessionRepo repository.RefreshSessionRepo
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type RegisterResult struct {
	UserID   uint64
	Username string
}

type AuthService interface {
	Register(ctx context.Context, username, password string) (RegisterResult, error)
	Login(ctx context.Context, username, password string) (AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (AuthTokens, error)
	Logout(ctx context.Context, jti, sessionID string, expiresAt time.Time) error
}

var _ AuthService = (*authService)(nil)

func NewAuthService(
	userRepo repository.UserRepo,
	jwtConfig *config.JWT,
	tokenBlacklistRepo repository.TokenBlacklistRepo,
	refreshSessionRepo repository.RefreshSessionRepo,
) AuthService {
	return &authService{
		userRepo:           userRepo,
		jwtConfig:          jwtConfig,
		tokenBlacklistRepo: tokenBlacklistRepo,
		refreshSessionRepo: refreshSessionRepo,
	}
}

func (s *authService) Register(ctx context.Context, username, password string) (RegisterResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return RegisterResult{}, apperr.CredentialsRequired()
	}
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return RegisterResult{}, apperr.InvalidArgument("username already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return RegisterResult{}, err
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return RegisterResult{}, err
	}
	user := model.User{
		Username: username,
		Password: hash,
	}
	if err := s.userRepo.Create(ctx, &user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return RegisterResult{}, apperr.InvalidArgument("username already exists")
		}
		return RegisterResult{}, err
	}
	return RegisterResult{
		UserID:   user.ID,
		Username: user.Username,
	}, nil
}

func (s *authService) Login(ctx context.Context, username, password string) (AuthTokens, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return AuthTokens{}, apperr.CredentialsRequired()
	}
	if s.refreshSessionRepo == nil || s.jwtConfig == nil {
		return AuthTokens{}, apperr.Internal("auth service unavailable", nil)
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AuthTokens{}, apperr.InvalidCredentials()
		}
		return AuthTokens{}, err
	}

	if err = utils.VerifyPassword(password, user.Password); err != nil {
		return AuthTokens{}, apperr.InvalidCredentials()
	}

	sessionID := utils.GenerateUUID()
	refreshSecret, err := utils.GenerateSecureToken()
	if err != nil {
		return AuthTokens{}, err
	}
	accessToken, _, err := jwt.GenerateJWT(
		strconv.FormatInt(int64(user.ID), 10),
		sessionID,
		s.jwtConfig.Secret,
		time.Duration(s.jwtConfig.AccessExpiry)*time.Hour,
	)
	if err != nil {
		return AuthTokens{}, err
	}

	if err := s.refreshSessionRepo.Create(
		ctx,
		sessionID,
		user.ID,
		utils.SHA256Hex(refreshSecret),
		time.Duration(s.jwtConfig.RefreshExpiry)*time.Hour,
	); err != nil {
		return AuthTokens{}, err
	}

	return AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: buildRefreshToken(sessionID, refreshSecret),
		ExpiresIn:    int64((time.Duration(s.jwtConfig.AccessExpiry) * time.Hour) / time.Second),
	}, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (AuthTokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return AuthTokens{}, apperr.RefreshTokenInvalid()
	}
	if s.refreshSessionRepo == nil || s.jwtConfig == nil {
		return AuthTokens{}, apperr.Internal("auth service unavailable", nil)
	}

	sessionID, refreshSecret, ok := splitRefreshToken(refreshToken)
	if !ok {
		return AuthTokens{}, apperr.RefreshTokenInvalid()
	}
	newRefreshSecret, err := utils.GenerateSecureToken()
	if err != nil {
		return AuthTokens{}, err
	}

	userID, rotated, err := s.refreshSessionRepo.Rotate(
		ctx,
		sessionID,
		utils.SHA256Hex(refreshSecret),
		utils.SHA256Hex(newRefreshSecret),
		time.Duration(s.jwtConfig.RefreshExpiry)*time.Hour,
	)
	if err != nil {
		return AuthTokens{}, err
	}
	if !rotated {
		return AuthTokens{}, apperr.RefreshTokenInvalid()
	}

	accessToken, _, err := jwt.GenerateJWT(
		strconv.FormatUint(userID, 10),
		sessionID,
		s.jwtConfig.Secret,
		time.Duration(s.jwtConfig.AccessExpiry)*time.Hour,
	)
	if err != nil {
		return AuthTokens{}, err
	}

	return AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: buildRefreshToken(sessionID, newRefreshSecret),
		ExpiresIn:    int64((time.Duration(s.jwtConfig.AccessExpiry) * time.Hour) / time.Second),
	}, nil
}

func (s *authService) Logout(ctx context.Context, jti, sessionID string, expiresAt time.Time) error {
	if strings.TrimSpace(jti) == "" {
		return apperr.TokenInvalid()
	}
	if s.refreshSessionRepo != nil && sessionID != "" {
		if err := s.refreshSessionRepo.Delete(ctx, sessionID); err != nil {
			return err
		}
	}

	ttl := time.Until(expiresAt)
	if expiresAt.IsZero() || ttl <= 0 {
		return nil
	}
	return s.tokenBlacklistRepo.Blacklist(ctx, jti, ttl)
}

func buildRefreshToken(sessionID, refreshSecret string) string {
	return sessionID + "." + refreshSecret
}

func splitRefreshToken(refreshToken string) (string, string, bool) {
	sessionID, refreshSecret, ok := strings.Cut(strings.TrimSpace(refreshToken), ".")
	if !ok || sessionID == "" || refreshSecret == "" {
		return "", "", false
	}
	return sessionID, refreshSecret, true
}
