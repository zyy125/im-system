package service

import (
	"context"
	"strconv"
	"time"

	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/pkg/jwt"
	"github.com/zyy125/im-system/pkg/utils"
)

type UserService struct {
	userRepo *repository.UserRepo
	jwtCfg  *config.JWT
}

func NewUserService(userRepo *repository.UserRepo, jwtCfg *config.JWT) *UserService {
	return &UserService{userRepo: userRepo, jwtCfg: jwtCfg}
}

func (s *UserService) Register(ctx context.Context, username string, pwd string) error {
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
