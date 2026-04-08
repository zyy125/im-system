package service

import (
	"context"
	"errors"

	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/internal/model"
	"github.com/zyy125/im-system/internal/repository"
	"gorm.io/gorm"
)

type userService struct {
	userRepo     repository.UserRepo
	presenceRepo repository.PresenceRepo
}

type UserService interface {
	IsOnline(ctx context.Context, userID uint64) (bool, error)
	GetUserByID(ctx context.Context, userID uint64) (model.User, bool, error)
}

var _ UserService = (*userService)(nil)

func NewUserService(userRepo repository.UserRepo, presenceRepo repository.PresenceRepo) UserService {
	return &userService{userRepo: userRepo, presenceRepo: presenceRepo}
}

func (s *userService) IsOnline(ctx context.Context, userID uint64) (bool, error) {
	if userID == 0 {
		return false, apperr.RequiredOne("user_id")
	}
	return s.presenceRepo.IsOnline(ctx, userID)
}

func (s *userService) GetUserByID(ctx context.Context, userID uint64) (model.User, bool, error) {
	if userID == 0 {
		return model.User{}, false, apperr.RequiredOne("user_id")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, false, apperr.UserNotFound()
		}
		return model.User{}, false, err
	}
	online, err := s.presenceRepo.IsOnline(ctx, user.ID)
	if err != nil {
		return model.User{}, false, err
	}
	return user, online, nil
}
