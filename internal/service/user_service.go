package service

import (
	"context"
	"strconv"
	"time"
	"errors"

	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/jwt"
	"github.com/zyy125/im-system/pkg/utils"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo repository.UserRepo
	jwtCfg  *config.JWT
	tokenBlacklistRepo repository.TokenBlacklistRepo
}

func NewUserService(userRepo repository.UserRepo, jwtCfg *config.JWT, tokenBlacklistRepo repository.TokenBlacklistRepo) *UserService {
	return &UserService{userRepo: userRepo, jwtCfg: jwtCfg, tokenBlacklistRepo: tokenBlacklistRepo}
}

func (s *UserService) Register(ctx context.Context, username string, pwd string) error {
	if username == "" || pwd == "" {
		return errors.New("username or password is empty")
	}
	hash, err := utils.HashPassword(pwd)
	if err != nil {
		return err
	}
	if err := s.userRepo.Create(ctx, &model.User{
		Username: username,
		Password: hash,
	}); err != nil {
		return err
	}
	return nil
}

func (s *UserService) Login(ctx context.Context, username string, pwd string) (string, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.New("user not found")
		}
		return "", err
	}

	if err = utils.VerifyPassword(pwd, user.Password); err != nil {
		return "", err
	}

	token, _, err := jwt.GenerateJWT(strconv.FormatInt(int64(user.ID), 10), s.jwtCfg.Secret, time.Duration(s.jwtCfg.Expiry)*time.Hour)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) Logout(ctx context.Context, jti string) error {
	if err := s.tokenBlacklistRepo.Blacklist(ctx, jti); err != nil {
		return err
	}

	return nil
}