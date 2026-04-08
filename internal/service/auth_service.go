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
}

type AuthService interface {
	Register(ctx context.Context, username, password string) error
	Login(ctx context.Context, username, password string) (string, error)
	Logout(ctx context.Context, jti string) error
}

var _ AuthService = (*authService)(nil)

func NewAuthService(userRepo repository.UserRepo, jwtConfig *config.JWT, tokenBlacklistRepo repository.TokenBlacklistRepo) AuthService {
	return &authService{userRepo: userRepo, jwtConfig: jwtConfig, tokenBlacklistRepo: tokenBlacklistRepo}
}

func (s *authService) Register(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return apperr.CredentialsRequired()
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.userRepo.Create(ctx, &model.User{
		Username: username,
		Password: hash,
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperr.UserAlreadyExists()
		}
		return err
	}
	return nil
}

func (s *authService) Login(ctx context.Context, username, password string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", apperr.CredentialsRequired()
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperr.InvalidCredentials()
		}
		return "", err
	}

	if err = utils.VerifyPassword(password, user.Password); err != nil {
		return "", apperr.InvalidCredentials()
	}

	token, _, err := jwt.GenerateJWT(strconv.FormatInt(int64(user.ID), 10), s.jwtConfig.Secret, time.Duration(s.jwtConfig.Expiry)*time.Hour)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *authService) Logout(ctx context.Context, jti string) error {
	return s.tokenBlacklistRepo.Blacklist(ctx, jti)
}
